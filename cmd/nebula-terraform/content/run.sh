#!/bin/sh

DIRECTORY=$(ni get -p {.directory})
WORKSPACE=$(ni get -p {.workspace})
WORKSPACE_FILE=workspace.${WORKSPACE}.tfvars.json

CREDENTIALS=$(ni get -p {.credentials})
if [ -n "${CREDENTIALS}" ]; then
    ni credentials config
    export GOOGLE_APPLICATION_CREDENTIALS=/workspace/credentials.json
fi

GIT=$(ni get -p {.git})
if [ -n "${GIT}" ]; then
    ni git clone
    NAME=$(ni get -p {.git.name})
    WORKSPACE_PATH=/workspace/${NAME}/${DIRECTORY}
else
    WORKSPACE_PATH=${DIRECTORY}
fi

ni file -p vars -f ${WORKSPACE_PATH}/${WORKSPACE_FILE} -o json

cd ${WORKSPACE_PATH}

export TF_IN_AUTOMATION=true

terraform init
terraform workspace new ${WORKSPACE}
terraform workspace select ${WORKSPACE}
terraform apply -auto-approve