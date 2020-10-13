import * as pulumi from "@pulumi/pulumi";
import * as gcp from "@pulumi/gcp";
import * as k8s from "@pulumi/kubernetes";

const appName = "orb-snyk-app";
let imageTag = "latest";
let circleSha1 = process.env["CIRCLE_SHA1"]
if (circleSha1) {
    imageTag = circleSha1;
}

const dockerImage = "ariv3ra/snyk-pulumi-pipelines:" + imageTag;
const machineType = "g1-small";

// get the latest master version
const masterVersion = gcp.container.getEngineVersions().then(it => it.latestMasterVersion)

// create a gke cluster
const cluster = new gcp.container.Cluster(appName, {
    initialNodeCount: 3,
    minMasterVersion: masterVersion,
    nodeVersion: masterVersion,
    nodeConfig: {
        preemptible: true,
        machineType: machineType,
        oauthScopes: [
            "https://www.googleapis.com/auth/compute",
            "https://www.googleapis.com/auth/devstorage.read_only",
            "https://www.googleapis.com/auth/logging.write",
            "https://www.googleapis.com/auth/monitoring",
        ],
    },
})

export const kubeconfig = pulumi.all([ cluster.name, cluster.endpoint, cluster.masterAuth ]).apply(([ name, endpoint, auth ]) => {
    const context = `${gcp.config.project}_${gcp.config.zone}_${name}`;
    return `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${auth.clusterCaCertificate}
    server: https://${endpoint}
  name: ${context}
contexts:
- context:
    cluster: ${context}
    user: ${context}
  name: ${context}
current-context: ${context}
kind: Config
preferences: {}
users:
- name: ${context}
  user:
    auth-provider:
      config:
        cmd-args: config config-helper --format=json
        cmd-path: gcloud
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
`;
});

// create a provider with the kubeconfig from our cluster to use for the deployment
const k8sProvider = new k8s.Provider("gkeK8s", {
    kubeconfig: kubeconfig,
});

// create a namespace for our application
const ns = new k8s.core.v1.Namespace(appName, {}, { provider: k8sProvider });

// create a Deployment for our application
const appLabels = { appClass: appName };
const deployment = new k8s.apps.v1.Deployment(appName,
    {
        metadata: {
            namespace: ns.metadata.name,
            labels: appLabels,
        },
        spec: {
            replicas: 3,
            selector: { matchLabels: appLabels },
            template: {
                metadata: {
                    labels: appLabels,
                },
                spec: {
                    containers: [
                        {
                            name: appName,
                            image: dockerImage,
                            ports: [{ name: "port-8080", containerPort: 5000 }],
                        },
                    ],
                },
            },
        },
    },
    {
        provider: k8sProvider,
    },
);

// create a LoadBalancer Service for the deployment
const service = new k8s.core.v1.Service(appName,
    {
        metadata: {
            labels: appLabels,
            namespace: ns.metadata.name,
        },
        spec: {
            type: "LoadBalancer",
            ports: [{ port: 80, targetPort: 5000 }],
            selector: appLabels,
        },
    },
    {
        provider: k8sProvider,
    },
);

// export the IP address of our service
export const appEndpointIp = service.status.loadBalancer.ingress[0].ip;
