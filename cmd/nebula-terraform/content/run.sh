#!/bin/bash

DIRECTORY=$(ni get -p {.directory})
WORKSPACE=$(ni get -p {.workspace})
WORKSPACE_FILE=workspace.${WORKSPACE}.tfvars.json
JQ=jq
NI=ni

CREDENTIALS=$(ni get -p {.credentials})
if [ -n "${CREDENTIALS}" ]; then
    ni credentials config

    PROVIDER=$(ni get -p {.provider})
    if [ "${PROVIDER}" == "aws" ]; then
        export AWS_SHARED_CREDENTIALS_FILE=/workspace/credentials
    else
        export GOOGLE_APPLICATION_CREDENTIALS=/workspace/credentials.json
    fi
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

declare -a BACKEND_CONFIGS="( $( $NI get | $JQ -r 'try .backendConfig | to_entries[] | "-backend-config=\( .key )=\( .value )" | @sh' ) )"

terraform init ${BACKEND_CONFIGS[@]}
terraform workspace new ${WORKSPACE}
terraform workspace select ${WORKSPACE}
terraform apply -auto-approve

keys=$(terraform output -json | jq -r '. | keys | .[]')
for key in ${keys}; do
    value=$(terraform output ${key})
    if [[ "${value}" == *$'\n'* ]]; then
        value=$(echo "${value}" | base64 | tr -d '\n')
    fi
    ni output set --key "${key}" --value "${value}"
done
