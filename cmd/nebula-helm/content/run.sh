#!/bin/sh

ni cluster config

NS=$(ni get -p {.namespace})
CLUSTER=$(ni get -p {.cluster.name})

KUBECONFIG=/workspace/${CLUSTER}/kubeconfig

TLS_OPTIONS=
CREDENTIALS=$(ni get -p {.credentials})
if [ -n "${CREDENTIALS}" ]; then
    ni credentials config -d $(helm home)
    TLS_OPTIONS="--tls --tls-verify"
fi

COMMAND=$(ni get -p {.command})
ARGS=$(ni get -p {.args})

helm init --client-only
helm ${COMMAND} ${ARGS} ${TLS_OPTIONS} \
    --namespace ${NS} --kubeconfig ${KUBECONFIG}
