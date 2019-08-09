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
