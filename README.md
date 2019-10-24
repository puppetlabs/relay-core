# Nebula Tasks

The code in this repository supports Puppet's tailored task images for use in Nebula workflows. The code also supports the
APIs that the tasks use to get metadata, secrets, and other data.

## Builds

The build system checks each directory inside `./cmd` and looks for a
`build-info.sh` file. If one exists, it sources that file and plugs its
configuration into a Docker image template, `scripts/Dockerfile.template`. This
allows the build system to build the required binaries into their own Docker images. By default,
the final image is based on an Alpine container specified in `scripts/Dockerfile.include`.
If a directory inside `./cmd` contains a `Dockerfile.include`, the build system uses that configuration as
the final build image instead.

### build-info.sh

Currently, the supported configuration variables are:

- `DOCKER_CMD`: The name of the command in the image to execute as part of the
  `CMD` directive. For example, `nebula-k8s-provisioner`.
- `DOCKER_REPO`: A DockerHub repo path. For example, `gcr.io/nebula-tasks/nebula-k8s-provisioner`.

### Dockerfile.include

The build system uses a multi-stage build approach and calls the build container `builder`.
The base builder is not swappable at the moment. It would be nice to have a swappable builder in the future, but since
all of the sub-projects in this repo are written in Go, a standard Go base image is all that's needed at the moment. See the base `Dockerfile.include` for an example.
