#!/bin/bash
set -euo pipefail

export GOOGLE_APPLICATION_CREDENTIALS=/workspace/credentials.json

ni git clone
ni credentials config

NAME=$(ni get -p {.git.name})
CONTEXT=$(ni get -p {.context})
DOCKERFILE=$(ni get -p {.dockerfile})
DESTINATION=$(ni get -p {.destination})

declare -a BUILD_ARGS_OPTIONS="($(ni get | jq -r 'try .buildArgs | to_entries[] | @sh "--build-arg=\( .key )=\( .value )"'))"

CONTEXT="/workspace/${NAME}/${CONTEXT}"
DOCKERFILE="${CONTEXT}/${DOCKERFILE:-Dockerfile}"

/kaniko/executor \
    ${BUILD_ARGS_OPTIONS[@]} \
    --dockerfile=${DOCKERFILE} \
    --context=${CONTEXT} \
    --destination=${DESTINATION}
