#!/bin/bash

export RELEASE_MANIFEST=ci-release-manifest
export BRANCH="${TRAVIS_BRANCH:-$(git branch | grep \* | cut -d ' ' -f2)-local}"
export TRAVIS_EVENT_TYPE="${TRAVIS_EVENT_TYPE:-}"

DIRTY=
[ -n "$(git status --porcelain --untracked-files=no)" ] && DIRTY="-dirty"

TAG=$(git tag -l --contains HEAD | head -n 1)
if [ -n "${TAG}" ]; then
        export VERSION="${TAG}${DIRTY}"
    else
        export VERSION="$(git rev-parse --short HEAD)${DIRTY}"
fi

if [[ "$TRAVIS_EVENT_TYPE" == "pull_request" ]]; then
        export NO_DOCKER_PUSH=yes
fi

if [[ "$BRANCH" != "master" ]]; then
        export NO_DOCKER_PUSH=yes
fi

if [ -n "${DIRTY}" ]; then
        export NO_DOCKER_PUSH=yes
fi

fail() {
    echo "ERROR: ${1}"
    exit 1
}
