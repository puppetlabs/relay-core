#!/bin/sh

helm init --client-only

ni cluster config

NS=$(ni get -p {.namespace})
CLUSTER=$(ni get -p {.cluster.name})
KUBECONFIG=/workspace/"${CLUSTER}"/kubeconfig

TLS_OPTIONS=
CREDENTIALS=$(ni get -p {.credentials})
if [ -n "${CREDENTIALS}" ]; then
    ni credentials config -d $(helm home)
    TLS_OPTIONS="--tls --tls-verify"
fi

CHART=$(ni get -p {.chart})

GIT=$(ni get -p {.git})
if [ -n "${GIT}" ]; then
    ni git clone
    CHART_NAME=$(ni get -p {.git.name})
    CHART_PATH=/workspace/${NAME}/${CHART}
else
    CHART_NAME=$(ni get -p {.name})
    CHART_PATH=${CHART}
fi

ni file -p values -f values-overrides.yaml -o yaml

helm upgrade ${CHART_NAME} ${CHART_PATH} ${TLS_OPTIONS} \
    --install --recreate-pods \
    --namespace ${NS} --kubeconfig ${KUBECONFIG} \
    -f values-overrides.yaml
