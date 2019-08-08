#!/bin/bash

export RELEASE_MANIFEST=ci-release-manifest
export TRAVIS_BRANCH="${TRAVIS_BRANCH:-$(git branch | grep \* | cut -d ' ' -f2)}"
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

declare -A NEBULA_WORKFLOWS

[ -r "nebula-deploy.sh" ] && source "nebula-deploy.sh"

NEBULA_WORKFLOWS[master]=nebula-system-deploy-prod-1
NEBULA_WORKFLOWS[development]=nebula-system-deploy-stage-1

NEBULA_WORKFLOW=${NEBULA_WORKFLOWS["$TRAVIS_BRANCH"]:-}
if [ -z "${NEBULA_WORKFLOW}" ]; then
    export NO_DOCKER_PUSH=yes
else
    export NEBULA_WORKFLOW
fi

if [ -n "${DIRTY}" ]; then
        export NO_DOCKER_PUSH=yes
fi

fail() {
    echo "ERROR: ${1}"
    exit 1
}
