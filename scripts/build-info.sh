#!/usr/bin/env bash

export TRAVIS_BRANCH="${TRAVIS_BRANCH:-$(git branch | grep \* | cut -d ' ' -f2)}"
export TRAVIS_PULL_REQUEST="${TRAVIS_PULL_REQUEST:-}"
export NEBULA_TASKS_BUILD_DIR="${NEBULA_TASKS_BUILD_DIR:-.build}"
export NO_DOCKER_PUSH="${NO_DOCKER_PUSH:-yes}"

export NEBULA_TASKS_RELEASE_LATEST=
[ "${TRAVIS_BRANCH}" = "master" ] && [ "${TRAVIS_PULL_REQUEST}" = "false" ] && export NEBULA_TASKS_RELEASE_LATEST=true

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

declare -A NEBULA_WORKFLOWS

[ -r "nebula-deploy.sh" ] && source "nebula-deploy.sh"

NEBULA_WORKFLOWS[master]=nebula-prod-1
NEBULA_WORKFLOWS[development]=nebula-stage-1

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
