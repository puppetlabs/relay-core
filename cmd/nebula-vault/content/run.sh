#!/bin/sh

export VAULT_ADDR=$(ni get -p {.vault.url})

COMMAND=$(ni get -p {.command})
ARGS=$(ni get -p {.args})

GIT=$(ni get -p {.git})
if [ -n "${GIT}" ]; then
    ni git clone
    NAME=$(ni get -p {.git.name})
    WORKSPACE_PATH=/workspace/${NAME}
    cd ${WORKSPACE_PATH}
fi

TOKEN=$(ni get -p {.vault.token})
if [ -n "${TOKEN}" ]; then
    vault login ${TOKEN}
fi

vault ${COMMAND} ${ARGS}