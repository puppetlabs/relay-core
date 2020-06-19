#!/usr/bin/env bash

export TRAVIS_BRANCH="${TRAVIS_BRANCH:-$(git branch | grep \* | cut -d ' ' -f2)}"
export TRAVIS_PULL_REQUEST="${TRAVIS_PULL_REQUEST:-}"
export RELAY_CORE_BUILD_DIR="${RELAY_CORE_BUILD_DIR:-.build}"
export NO_DOCKER_PUSH="${NO_DOCKER_PUSH:-yes}"

export RELAY_CORE_RELEASE_LATEST=
[ "${TRAVIS_BRANCH}" = "master" ] && [ "${TRAVIS_PULL_REQUEST}" = "false" ] && export RELAY_CORE_RELEASE_LATEST=true

DIRTY=
[ -n "$(git status --porcelain --untracked-files=no)" ] && DIRTY="-dirty"

TAG=$(git tag -l --contains HEAD | head -n 1)
if [ -n "${TAG}" ]; then
        export VERSION="${TAG}${DIRTY}"
    else
        export VERSION="$(git rev-parse --short HEAD)${DIRTY}"
fi

if [[ "$TRAVIS_PULL_REQUEST" == "false" ]]; then
        export NO_DOCKER_PUSH=
fi

declare -A RELAY_WORKFLOWS

[ -r "$(dirname "$0")/relay-deploy.sh" ] && source "$(dirname "$0")/relay-deploy.sh"

RELAY_WORKFLOWS[master]=nebula-prod-1
RELAY_WORKFLOWS[development]=nebula-stage-1

RELAY_WORKFLOW=${RELAY_WORKFLOWS["$TRAVIS_BRANCH"]:-}
if [ -z "${RELAY_WORKFLOW}" ]; then
    export NO_DOCKER_PUSH=yes
else
    export RELAY_WORKFLOW
fi

if [ -n "${DIRTY}" ]; then
    export NO_DOCKER_PUSH=yes
fi

fail() {
    echo "ERROR: ${1}"
    exit 1
}
