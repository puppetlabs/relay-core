#!/bin/sh

CREDENTIALS=$(ni get -p {.credentials})
if [ -n "${CREDENTIALS}" ]; then
    ni credentials config
    export AWS_SHARED_CREDENTIALS_FILE=/workspace/credentials
fi

WORKSPACE_PATH=$(ni get -p {.path})

GIT=$(ni get -p {.git})
if [ -n "${GIT}" ]; then
    ni git clone
    NAME=$(ni get -p {.git.name})
    WORKSPACE_PATH=/workspace/${NAME}/${WORKSPACE_PATH}
fi

cd ${WORKSPACE_PATH}

CLUSTER=$(ni get -p {.cluster.name})
REGION=$(ni get -p {.cluster.region})

ecs-cli compose --project-name ${CLUSTER} service up --cluster ${CLUSTER} --cluster-config ${CLUSTER} --launch-type FARGATE --region ${REGION}