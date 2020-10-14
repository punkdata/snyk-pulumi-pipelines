import * as gcp from "@pulumi/gcp";

let imageTag = "latest";
let circleSha1 = process.env["CIRCLE_SHA1"]
if (circleSha1) {
    imageTag = circleSha1;
}

let dockerImage = "ariv3ra/orb-pulumi-gcp:" + imageTag;
const metadata = {"gce-container-declaration":"spec:\n  containers:\n    - name: test-docker\n      image: '" + dockerImage +"'\n      stdin: false\n      tty: false\n  restartPolicy: Always\n"}

const addr = new gcp.compute.Address("orb-pulumi-gcp");

const network = new gcp.compute.Network("network");

const firewall = new gcp.compute.Firewall("firewall", {
    network: network.selfLink,
    allows: [{
        protocol: "tcp",
        ports: ["22", "5000"]
    }]
});

const instance = new gcp.compute.Instance("orb-pulumi-gcp", {
    name: "orb-pulumi-gcp",
    machineType: "g1-small",
    bootDisk: {
        initializeParams: {
            image: "projects/cos-cloud/global/images/cos-stable-69-10895-62-0",
        },
    },
    networkInterfaces: [{
        network: network.id,
        accessConfigs: [{
            natIp: addr.address
        }]
    }],
    metadata: metadata,
})

export const instanceName = instance.name;
export const instanceMetadata = instance.metadata;
export const instanceNetwork = instance.networkInterfaces;
export const externalIp = addr.address;
