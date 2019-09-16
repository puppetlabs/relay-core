#!/bin/bash
set -euo pipefail

#
# Commands
#

AZ="${AZ:-az}"
JQ="${JQ:-jq}"
NI="${NI:-ni}"

#
# Variables
#

WORKDIR="${WORKDIR:-/workspace}"

#
#
#

log() {
  echo "[$( date -Iseconds )] $@"
}

err() {
  log "error: $@" >&2
  exit 2
}

usage() {
  echo "usage: $@" >&2
  exit 1
}

TENANT_ID="$( $NI get -p '{ .azure.tenantID }' )"
[ -z "${TENANT_ID}" ] && usage 'spec: your `azure` hash is missing the `tenantID` key'

USERNAME="$( $NI get -p '{ .azure.username }' )"
[ -z "${USERNAME}" ] && usage 'spec: your `azure` hash is missing the `username` key'

PASSWORD="$( $NI get -p '{ .azure.password }' )"
PASSWORD_OR_CERT="${PASSWORD}"
CERT="$( $NI get -p '{ .azure.cert }' )"
[ -n "${PASSWORD}" ] && [ -n "${CERT}" ] && usage 'spec: you must specify only one of `password` or `cert` keys'
if [ -n "${CERT}" ] ; then
  PASSWORD_OR_CERT="${WORKDIR}/certificate.pem"
  touch "${PASSWORD_OR_CERT}"
  chmod 0600 "${PASSWORD_OR_CERT}"
  echo "${CERT}" > "${PASSWORD_OR_CERT}"
fi
[ -z "${PASSWORD_OR_CERT}" ] && usage 'spec: your `azure` hash must specify either a `password` or a `cert` key'

$AZ login --service-principal --tenant "${TENANT_ID}" --username "${USERNAME}" --password "${PASSWORD_OR_CERT}"

DEPLOYMENT_NAME="$( $NI get -p '{ .deploymentName }' )"
[ -z "${DEPLOYMENT_NAME}" ] && usage 'spec: please specify a value for `deploymentName`'

RESOURCE_GROUP="$( $NI get -p '{ .resourceGroup }' )"
LOCATION="$( $NI get -p '{ .location }' )"
[ -z "${RESOURCE_GROUP}" ] && [ -z "${LOCATION}" ] && usage 'spec: you must specify either a `resourceGroup` or a `location` key'
[ -n "${RESOURCE_GROUP}" ] && [ -n "${LOCATION}" ] && usage 'spec: you must specify only one of `resourceGroup` or `location` key'

TEMPLATE_FILE="$( $NI get -p '{ .templateFile }' )"
if [ -n "${TEMPLATE_FILE}" ]; then
  ni git clone -d "${WORKDIR}/repo" || err 'could not clone git repository'
  [[ ! -d "${WORKDIR}/repo" ]] && usage 'spec: please specify `git`, the Git repository to use to resolve the template file'

  TEMPLATE_FILE="$( realpath "${WORKDIR}/repo/$( $NI get -p '{ .git.name }' )/${TEMPLATE_FILE}" )"
  if [[ "$?" != 0 ]] || [[ "${TEMPLATE_FILE}" != "${WORKDIR}/repo/"* ]]; then
    err 'spec: `templateFile` does not contain a valid reference to a file in the specified repository'
  fi
else
  TEMPLATE_FILE="${WORKDIR}/inline.template"

  TEMPLATE="$( $NI get -p '{ .template }' | tee "${TEMPLATE_FILE}" )"
  [ -z "${TEMPLATE}" ] && usage 'spec: please specify one of `template`, an inline template, or `templateFile`, the template file to deploy'
fi

declare -a DEPLOY_ARGS

mapfile -t PARAMETERS < <( $NI get | $JQ -r 'try .parameters | to_entries[] | "\( .key )=\( .value )"' )
[[ ${#PARAMETERS[@]} -gt 0 ]] && DEPLOY_ARGS+=( --parameter "${PARAMETERS[@]}" )

if [ -n "${RESOURCE_GROUP}" ] ; then
  ## Deploy to resource group
  $AZ group deployment create \
    --resource-group "${RESOURCE_GROUP}" \
    --name "${DEPLOYMENT_NAME}" \
    --template-file "${TEMPLATE_FILE}" \
    "${DEPLOY_ARGS[@]}"
else
  ## Deploy to subscription
  $AZ deployment create \
    --location "${LOCATION}" \
    --name "${DEPLOYMENT_NAME}" \
    --template-file "${TEMPLATE_FILE}" \
    "${DEPLOY_ARGS[@]}"
fi
