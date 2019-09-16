#!/bin/sh

ni cluster config

NS=$(ni get -p {.namespace})
CLUSTER=$(ni get -p {.cluster.name})

KUBECONFIG=/workspace/${CLUSTER}/kubeconfig

COMMAND=$(ni get -p {.command})
ARGS=$(ni get -p {.args})

helm init --client-only
helm ${COMMAND} ${ARGS} --namespace ${NS} --kubeconfig ${KUBECONFIG}
