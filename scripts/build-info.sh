#!/usr/bin/env bash

export GITHUB_REF="${GITHUB_REF:-$(git symbolic-ref HEAD)}"
export GITHUB_EVENT_NAME="${GITHUB_EVENT_NAME:-}"
export NO_DOCKER_PUSH="${NO_DOCKER_PUSH:-yes}"

RELAY_CORE_BRANCH=
if [[ "${GITHUB_REF}" == refs/heads/* ]]; then
    RELAY_CORE_BRANCH="${GITHUB_REF#refs/heads/}"
fi
export RELAY_CORE_BRANCH

export RELAY_CORE_RELEASE_LATEST=
[ "${RELAY_CORE_BRANCH}" = "master" ] && [ "${GITHUB_EVENT_NAME}" = "push" ] && export RELAY_CORE_RELEASE_LATEST=true

DIRTY=
[ -n "$(git status --porcelain --untracked-files=no)" ] && DIRTY="-dirty"

TAG=$(git tag -l --contains HEAD | head -n 1)
if [ -n "${TAG}" ]; then
        export VERSION="${TAG}${DIRTY}"
    else
        export VERSION="$(git rev-parse --short HEAD)${DIRTY}"
fi

if [[ "$GITHUB_EVENT_NAME" == "push" ]]; then
        export NO_DOCKER_PUSH=
fi

declare -A RELAY_WORKFLOWS

[ -r "$(dirname "$0")/relay-deploy.sh" ] && source "$(dirname "$0")/relay-deploy.sh"

RELAY_WORKFLOWS[master]=nebula-prod-1
RELAY_WORKFLOWS[development]=nebula-stage-1

if [ -n "${RELAY_CORE_BRANCH}" ]; then
    RELAY_WORKFLOW=${RELAY_WORKFLOWS["$RELAY_CORE_BRANCH"]:-}
    if [ -z "${RELAY_WORKFLOW}" ]; then
        export NO_DOCKER_PUSH=yes
    else
        export RELAY_WORKFLOW
    fi
else
    export NO_DOCKER_PUSH=yes
fi

if [ -n "${DIRTY}" ]; then
    export NO_DOCKER_PUSH=yes
fi

fail() {
    echo "ERROR: ${1}"
    exit 1
}
