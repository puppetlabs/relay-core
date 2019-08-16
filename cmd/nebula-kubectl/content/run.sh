#!/bin/sh

NS=$(ni get -p {.namespace})
CLUSTER=$(ni get -p {.cluster.name})
KUBECONFIG=/workspace/${CLUSTER}/kubeconfig

ni cluster config

COMMAND=$(ni get -p {.command})
FILE=
if [ "${COMMAND}" == "apply" ]; then
    ARGS="-f"
    FILE=$(ni get -p {.file})
else
    ARGS=$(ni get -p {.args})
fi

FILE_PATH=${FILE}

GIT=$(ni get -p {.git})
if [ -n "${GIT}" ]; then
    ni git clone
    NAME=$(ni get -p {.git.name})
    FILE_PATH=/workspace/${NAME}/${FILE}
fi

kubectl ${COMMAND} ${ARGS} ${FILE_PATH} --namespace ${NS} --kubeconfig ${KUBECONFIG}