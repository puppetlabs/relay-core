# Nebula Tasks

The code in this repository supports the tailored task images created by
Puppet for use in Nebula workflows. Some of the code is used to support
APIs that the tasks can use to get metadata, secrets and other data needed
to run.

## Builds

The build system will check each directory inside `./cmd` and look for a
`build-info.sh` file. If one exists, it will source that file and plug it's
configuration into a docker image template `scripts/Dockerfile.template`. This
allows all required binaries to be build into their own docker images. By default
the final image is based on an alpine container specified in `scripts/Dockerfile.include`.
If a directory inside `./cmd` contains a `Dockerfile.include`, that will be used as
the final build image instead.

### build-info.sh

Current supported configuration variables are listed below.

- `DOCKER_CMD`: this is the name of the command in the image to execute as part of the
  `CMD` directive (i.e. `nebula-k8s-provisioner`).
- `DOCKER_REPO`: this is a docker hub repo path (i.e. `gcr.io/nebula-tasks/nebula-k8s-provisioner`).

### Dockerfile.include

The build system uses a multi-stage build approach and calls the build container `builder`.
The base builder is not swappable at the moment. This would be a nice to have, but since
all of the sub-projects in this repo are written in Go, a standard Go base image is the only
one needed. See the base Dockerfile.include for an example.

## Tasks

### projectnebula/slack-notification

A task that will send a message to a slack channel.

| Parameter | Description | Default | Required |
|-----------|-------------|---------|----------|
| `apiToken` | The slack legacy API token to use | None | True |
| `channel` | The channel to send the message to | None | True |
| `message` | The message to send | None | True |
| `username` | The username to use in slack | None | True |

### projectnebula/k8s-provisioner

A task that creates and manages Kubernetes clusters in cloud platforms.

Note: This task can cost you money. It will provision resources in your cloud platform account
which can incur charges for resource use.

Current supported platforms:
- Google Cloud Platform (GCP)
- AWS

| Parameter | Description | Default | Required |
|-----------|-------------|---------|----------|
| `provider` | The cloud provider to use. Currently "aws" or "gcp". | None | True |
| `project` | The GCP project ID. Not used with AWS. | None | True for GCP |
| `clusterName` | A name for your cluster. This must be a FQDN. You can use a root domain in route53 or GCP domain name service or you can set the domain to `k8s.local` if you don't want to use one of your roots. | None | True |
| `credentials` | A map of credentials used for platform authentication | None | True |
| `credentials.gcpServiceAccountFile` | The GCP service account json | None | True |
| `credentials.awsAccessKeyID` | The aws access key ID | None | True |
| `credentials.awsSecretAccessKey` | The aws secret access key | None | True |
| `credentials.sshPublicKey` | An ssh public key to install on the virtual machine instances that run the cluster | None | True for AWS |
| `stateStoreName` | a storage bucket name to store cluster state. This configuration will use the storage system of your cloud provider, so if you are using AWS, then s3 will be used. If you are using GCP then gs will be used. If the bucket exists, then the task will try to just use it, otherwise the task will attempt to create the bucket. Multiple clusters can use the same state storage as long as the clusterName's are different. | None | True |
| `masterCount` | A count of how many master nodes to provision. | 1 | False |
| `nodeCount` | A count of how many agent nodes to provision. | 3 | False |
| `zones` | A list of zones in the cloud platform to run node instances in. | None | True (at least one) |
| `region` | A platform region to use when provisioning a cluster. | None | True |

TODO:
- [ ] integrate outputs system by storing master certificates used in kubeconfig for descendant tasks
- [ ] add support for Digital Ocean (supported by kops directly as alpha)
