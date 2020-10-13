package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pulumi/pulumi/pkg/v2/testing/integration"
	"github.com/stretchr/testify/assert"
)

func TestAccAppAndInfra(t *testing.T) {
	test := getGoogleBase(t).
		With(integration.ProgramTestOptions{
			Dir: filepath.Join(getCwd(t), "..", "pulumi", "gcp", "orb-snyk"),
			ExtraRuntimeValidation: func(t *testing.T, stack integration.RuntimeValidationStackInfo) {
				maxWait := 10 * time.Minute
				endpoint := stack.Outputs["appEndpointIp"].(string)
				assertHTTPResultWithRetry(t, endpoint, nil, maxWait, func(body string) bool {
					return assert.Contains(t, body, "Welcome to CI/CD")
				})
			},
		})

	integration.ProgramTest(t, &test)
}

func getGoogleBase(t *testing.T) integration.ProgramTestOptions {
	gkeBase := integration.ProgramTestOptions{
		Config: map[string]string{
			"gcp:project": "cicd-workshops",
			"gcp:zone":    "us-east1-d",
			"gcp:region":  "us-east1",
		},
		ExpectRefreshChanges: true,
		Quick:                true,
		RetryFailedSteps:     true,
	}
	return gkeBase
}

func getCwd(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	return cwd
}

func assertHTTPResultWithRetry(t *testing.T, output interface{}, headers map[string]string, maxWait time.Duration, check func(string) bool) bool {
	return assertHTTPResultShapeWithRetry(t, output, headers, maxWait, func(string) bool { return true }, check)
}

func assertHTTPResultShapeWithRetry(t *testing.T, output interface{}, headers map[string]string, maxWait time.Duration,
	ready func(string) bool, check func(string) bool) bool {
	hostname, ok := output.(string)
	if !assert.True(t, ok, fmt.Sprintf("expected `%s` output", output)) {
		return false
	}

	if !(strings.HasPrefix(hostname, "http://") || strings.HasPrefix(hostname, "https://")) {
		hostname = fmt.Sprintf("http://%s", hostname)
	}

	startTime := time.Now()
	count, sleep := 0, 0
	for true {
		now := time.Now()
		req, err := http.NewRequest("GET", hostname, nil)
		if !assert.NoError(t, err) {
			return false
		}

		for k, v := range headers {
			// Host header cannot be set via req.Header.Set(), and must be set
			// directly.
			if strings.ToLower(k) == "host" {
				req.Host = v
				continue
			}
			req.Header.Set(k, v)
		}

		client := &http.Client{Timeout: time.Second * 10}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			if !assert.NotNil(t, resp.Body, "resp.body was nil") {
				return false
			}

			// Read the body
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if !assert.NoError(t, err) {
				return false
			}

			bodyText := string(body)

			// Even if we got 200 and a response, it may not be ready for assertion yet - that's specific per test.
			if ready(bodyText) {
				// Verify it matches expectations
				return check(bodyText)
			}
		}
		if now.Sub(startTime) >= maxWait {
			fmt.Printf("Timeout after %v. Unable to http.get %v successfully.", maxWait, hostname)
			return false
		}
		count++
		// delay 10s, 20s, then 30s and stay at 30s
		if sleep > 30 {
			sleep = 30
		} else {
			sleep += 10
		}
		time.Sleep(time.Duration(sleep) * time.Second)
		fmt.Printf("Http Error: %v\n", err)
		fmt.Printf("  Retry: %v, elapsed wait: %v, max wait %v\n", count, now.Sub(startTime), maxWait)
	}

	return false
}
