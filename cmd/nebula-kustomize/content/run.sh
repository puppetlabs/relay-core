#!/bin/sh

NS=$(ni get -p {.namespace})
CLUSTER=$(ni get -p {.cluster.name})
KUBECONFIG=/workspace/${CLUSTER}/kubeconfig

ni cluster config

WORKSPACE_PATH=$(ni get -p {.path})

GIT=$(ni get -p {.git})
if [ -n "${GIT}" ]; then
    ni git clone
    NAME=$(ni get -p {.git.name})
    WORKSPACE_PATH=/workspace/${NAME}/${WORKSPACE_PATH}
fi

kubectl kustomize ${WORKSPACE_PATH}
kubectl apply -k ${WORKSPACE_PATH} --namespace ${NS} --kubeconfig ${KUBECONFIG}