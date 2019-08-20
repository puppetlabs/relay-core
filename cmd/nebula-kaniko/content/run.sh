#!/bin/sh

export GOOGLE_APPLICATION_CREDENTIALS=/workspace/credentials.json

ni git clone
ni credentials config

NAME=$(ni get -p {.git.name})
DOCKERFILE=$(ni get -p {.git.dockerfile})
DESTINATION=$(ni get -p {.destination})

WORKSPACE=/workspace/${NAME}
DOCKERFILE=${WORKSPACE}/${DOCKERFILE:-Dockerfile}

/kaniko/executor --dockerfile=${DOCKERFILE} --context=${WORKSPACE} --destination=${DESTINATION}