#!/bin/bash
set -euo pipefail

#
# Commands
#

AWS="${AWS:-aws}"
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

STACK_NAME="$( $NI get -p '{ .stackName }' )"
[ -z "${STACK_NAME}" ] && usage 'spec: please specify a value for `stackName`, the CloudFormation stack to create or update'

ni aws config -d "${WORKDIR}/.aws"
eval "$( ni aws env -d "${WORKDIR}/.aws" )"

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

S3_BUCKET="$( $NI get -p '{ .s3.bucket }' )"
[ -n "${S3_BUCKET}" ] && DEPLOY_ARGS+=( --s3-bucket "${S3_BUCKET}" )

S3_PREFIX="$( $NI get -p '{ .s3.prefix }' )"
[ -n "${S3_PREFIX}" ] && DEPLOY_ARGS+=( --s3-prefix "${S3_PREFIX}" )

PARAMETERS=( $( $NI get | $JQ -r 'try .parameters | to_entries[] | "\( .key )=\( .value )"' ) )
[[ ${#PARAMETERS[@]} -gt 0 ]] && DEPLOY_ARGS+=( --parameter-overrides "${PARAMETERS[@]}" )

CAPABILITIES=( $( $NI get | $JQ -r 'try .capabilities[]' ) )
[[ ${#CAPABILITIES[@]} -gt 0 ]] && DEPLOY_ARGS+=( --capabilities "${CAPABILITIES[@]}" )

TAGS=( $( $NI get | $JQ -r 'try .tags | to_entries[] | "\( .key )=\( .value )"' ) )
[[ ${#TAGS[@]} -gt 0 ]] && DEPLOY_ARGS+=( --tags "${TAGS[@]}" )

$AWS cloudformation deploy \
  --stack-name "${STACK_NAME}" \
  --template-file "${TEMPLATE_FILE}" \
  --no-fail-on-empty-changeset \
  "${DEPLOY_ARGS[@]}"
