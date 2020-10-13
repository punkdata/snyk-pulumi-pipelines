package main

import (
	"path/filepath"
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
			"gcp:region":    "us-east1",
		},
		ExpectRefreshChanges: true,
		Quick:                true,
		RetryFailedSteps:     true,
	}
	return gkeBase
}

