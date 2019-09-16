#!/bin/bash
set -euo pipefail

#
# Commands
#

AWS="${AWS:-aws}"
JQ="${JQ:-jq}"
MKDIR_P="${MKDIR_P:-mkdir -p}"
NI="${NI:-ni}"
ZIP_="${ZIP_:-zip}"

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

FUNCTION_NAME="$( $NI get -p '{ .functionName }' )"
[ -z "${FUNCTION_NAME}" ] && usage 'spec: please specify a value for `functionName`, the name of the function to create or update'

ni aws config -d "${WORKDIR}/.aws"
eval "$( ni aws env -d "${WORKDIR}/.aws" )"

declare -a CREATE_ARGS=( "--function-name=${FUNCTION_NAME}" )
declare -a UPDATE_CODE_ARGS=( "--function-name=${FUNCTION_NAME}" )
declare -a UPDATE_CONFIG_ARGS=( "--function-name=${FUNCTION_NAME}" )

RUNTIME="$( $NI get -p '{ .runtime }' )"
[ -z "${RUNTIME}" ] && usage 'spec: please specify a value for `runtime`, the Lambda runtime environment to use'

HANDLER="$( $NI get -p '{ .handler }' )"
[ -z "${HANDLER}" ] && usage 'spec: please specify a value for `handler`, the code entry point for the function when executed'

CREATE_ARGS+=( "--runtime=${RUNTIME}" "--handler=${HANDLER}" )
UPDATE_CONFIG_ARGS+=( "--runtime=${RUNTIME}" "--handler=${HANDLER}" )

DESCRIPTION="$( $NI get -p '{ .description }' )"
[ -n "${DESCRIPTION}" ] && {
  CREATE_ARGS+=( "--description=${DESCRIPTION}" )
  UPDATE_CONFIG_ARGS+=( "--description=${DESCRIPTION}" )
}

TIMEOUT_SECONDS="$( $NI get -p '{ .timeoutSeconds }' )"
[ -n "${TIMEOUT_SECONDS}" ] && {
  CREATE_ARGS+=( "--timeout=${TIMEOUT_SECONDS}" )
  UPDATE_CONFIG_ARGS+=( "--timeout=${TIMEOUT_SECONDS}" )
}

MEMORY_SIZE_MB="$( $NI get -p '{ .memorySizeMB }' )"
[ -n "${MEMORY_SIZE_MB}" ] && {
  CREATE_ARGS+=( "--memory-size=${MEMORY_SIZE_MB}" )
  UPDATE_CONFIG_ARGS+=( "--memory-size=${MEMORY_SIZE_MB}" )
}

VPC_CONFIG="$( $NI get | $JQ 'select((.vpc.subnetIDs | type) == "array" and (.vpc.securityGroupIDs | type) == "array") | { SubnetIds: .vpc.subnetIDs, SecurityGroupIds: .vpc.securityGroupIDs }' )"
[ -n "${VPC_CONFIG}" ] && {
  CREATE_ARGS+=( "--vpc-config=${VPC_CONFIG}" )
  UPDATE_CONFIG_ARGS+=( "--vpc-config=${VPC_CONFIG}" )
}

DEAD_LETTER_CONFIG="$( $NI get | $JQ 'select((.deadLetterQueue.targetARN | type) == "string") | { TargetArn: .deadLetterQueue.targetARN }' )"
[ -n "${DEAD_LETTER_CONFIG}" ] && {
  CREATE_ARGS+=( "--dead-letter-config=${DEAD_LETTER_CONFIG}" )
  UPDATE_CONFIG_ARGS+=( "--dead-letter-config=${DEAD_LETTER_CONFIG}" )
}

ENV="$( $NI get | $JQ '{ Variables: ({} + .env) }' )"
[ -n "${ENV}" ] && {
  CREATE_ARGS+=( "--environment=${ENV}" )
  UPDATE_CONFIG_ARGS+=( "--environment=${ENV}" )
}

TRACE="$( $NI get -p '{ .trace }' )"
[ "${TRACE}" == "true" ] && {
  CREATE_ARGS+=( "--tracing-config=Mode=Active" )
  UPDATE_CONFIG_ARGS+=( "--tracing-config=Mode=Active" )
} || {
  CREATE_ARGS+=( "--tracing-config=Mode=PassThrough" )
  UPDATE_CONFIG_ARGS+=( "--tracing-config=Mode=PassThrough" )
}

TAGS="$( $NI get | $JQ '.tags | select(type == "object") // empty' )"
[ -n "${TAGS}" ] && {
  CREATE_ARGS+=( "--tags=${TAGS}" )
  UPDATE_CONFIG_ARGS+=( "--tags=${TAGS}" )
}

declare -a LAYER_ARNS="( $( $NI get | $JQ -r 'try .layerARNs[] | @sh' ) )"
[[ ${#LAYER_ARNS[@]} -gt 0 ]] && {
  CREATE_ARGS+=( --layers "${LAYER_ARNS[@]}" )
  UPDATE_CONFIG_ARGS+=( --layers "${LAYER_ARNS[@]}" )
}

SOURCE_PATH="$( $NI get -p '{ .sourcePath }' )"
SOURCE_CONTENT="$( $NI get -p '{ .source.content }' )"
if [ -n "${SOURCE_PATH}" ]; then
  ni git clone -d "${WORKDIR}/repo" || err 'could not clone git repository'
  [[ ! -d "${WORKDIR}/repo" ]] && usage 'spec: please specify `git`, the Git repository to use to find the deployment package'

  SOURCE_PATH="$( realpath "${WORKDIR}/repo/$( $NI get -p '{ .git.name }' )/${SOURCE_PATH}" )"
  if [[ "$?" != 0 ]] || [[ "${SOURCE_PATH}" != "${WORKDIR}/repo/"* ]]; then
    err 'spec: `sourcePath` does not contain a valid reference to a path in the specified repository'
  fi
elif [ -n "${SOURCE_CONTENT}" ]; then
  SOURCE_NAME="$( $NI get -p '{ .source.name }' )"
  [ -z "${SOURCE_NAME}" ] && usage 'spec: specify a value for `source.name`, the file name of the content to use as a deployment'

  SOURCE_NAME="${SOURCE_NAME##/}"

  SOURCE_PATH="${WORKDIR}/deployment"
  $MKDIR_P "${SOURCE_PATH}/$( dirname "${SOURCE_NAME}" )"
  cat >"${SOURCE_PATH}/${SOURCE_NAME}" <<<"${SOURCE_CONTENT}"

  log "Added inline source code to deployment package at ${SOURCE_PATH}/${SOURCE_NAME}"
fi

if [ -n "${SOURCE_PATH}" ]; then
  log "Using path ${SOURCE_PATH} for function code"

  if [ -d "${SOURCE_PATH}" ]; then
    (
      pushd "${SOURCE_PATH}" >/dev/null
      $ZIP_ -r "${WORKDIR}/deployment.zip" . -x '*/.git/'
    )

    SOURCE_PATH="${WORKDIR}/deployment.zip"

    log "Added all content to deployment file ${SOURCE_PATH}"
  fi

  CREATE_ARGS+=( "--zip-file=fileb://${SOURCE_PATH}" )
  UPDATE_CODE_ARGS+=( "--zip-file=fileb://${SOURCE_PATH}" )
else
  SOURCE_S3_BUCKET="$( $NI get -p '{ .sourceS3.bucket }' )"
  SOURCE_S3_KEY="$( $NI get -p '{ .sourceS3.key }' )"
  SOURCE_S3_OBJECT_VERSION="$( $NI get -p '{ .sourceS3.objectVersion }' )"

  [ -z "${SOURCE_S3_BUCKET}" ] && usage 'spec: specify a value for `sourceS3.bucket`, the S3 bucket that contains the deployment'
  [ -z "${SOURCE_S3_KEY}" ] && usage 'spec: specify a value for `sourceS3.key`, the key of the object that contains the deployment'

  log "Using s3://${SOURCE_S3_BUCKET}/${SOURCE_S3_KEY##/} for function code"

  CREATE_CODE_ARG="$( $JQ -n \
    --arg bucket "${SOURCE_S3_BUCKET}" \
    --arg key "${SOURCE_S3_KEY}" \
    --arg version "${SOURCE_S3_OBJECT_VERSION}" \
    '{ S3Bucket: $bucket, S3Key: $key, S3ObjectVersion: $version } | with_entries(select(.value != ""))'
  )"
  CREATE_ARGS+=( "--code=${CREATE_CODE_ARG}" )
  UPDATE_CODE_ARGS+=( "--s3-bucket=${SOURCE_S3_BUCKET}" "--s3-key=${SOURCE_S3_KEY}" )
  [ -n "${SOURCE_S3_OBJECT_VERSION}" ] && UPDATE_CODE_ARGS+=( "--s3-object-version=${SOURCE_S3_OBJECT_VERSION}" )
fi

EXECUTION_ROLE_ARN="$( $NI get -p '{ .executionRoleARN }' )"
if [ -z "${EXECUTION_ROLE_ARN}" ]; then
  ASSUME_ROLE_POLICY_DOCUMENT='{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": ["sts:AssumeRole"],
        "Principal": {"Service": ["lambda.amazonaws.com"]}
      }
    ]
  }'

  EXECUTION_ROLE_NAME="$( $NI get -p '{ .executionRole.name }' )"
  [ -z "${EXECUTION_ROLE_NAME}" ] && EXECUTION_ROLE_NAME="${FUNCTION_NAME}"

  log "Provisioning role ${EXECUTION_ROLE_NAME} for Lambda function ${FUNCTION_NAME}..."

  EXECUTION_ROLE_DATA="$(
    $AWS iam get-role --role-name "${EXECUTION_ROLE_NAME}" 2>/dev/null || \
      $AWS iam create-role \
        --role-name "${EXECUTION_ROLE_NAME}" \
        --assume-role-policy-document "${ASSUME_ROLE_POLICY_DOCUMENT}"
  )" || err 'could not create execution role, do you have permissions to do so?'
  EXECUTION_ROLE_ARN="$( $JQ -r '.Role.Arn' <<<"${EXECUTION_ROLE_DATA}" )"

  $AWS iam attach-role-policy --role-name "${EXECUTION_ROLE_NAME}" --policy-arn "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  $AWS iam attach-role-policy --role-name "${EXECUTION_ROLE_NAME}" --policy-arn "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
  $AWS iam attach-role-policy --role-name "${EXECUTION_ROLE_NAME}" --policy-arn "arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess"

  EXECUTION_ROLE_POLICY="$( $NI get -p '{ .executionRole.policy }' )"
  if [ -n "${EXECUTION_ROLE_POLICY}" ]; then
    $AWS iam put-role-policy \
      --role-name "${EXECUTION_ROLE_NAME}" \
      --policy-name LambdaExecutionPolicy \
      --policy-document="${EXECUTION_ROLE_POLICY}"
  fi

  log "Role provisioned at ${EXECUTION_ROLE_ARN}"
else
  log "Using role at ${EXECUTION_ROLE_ARN} for Lambda function ${FUNCTION_NAME}"
fi

CREATE_ARGS+=( "--role=${EXECUTION_ROLE_ARN}" )
UPDATE_CONFIG_ARGS+=( "--role=${EXECUTION_ROLE_ARN}" )

if $AWS lambda get-function --function-name "${FUNCTION_NAME}" >/dev/null 2>&1; then
  log "Function ${FUNCTION_NAME} already exists; updating..."

  $AWS lambda update-function-code "${UPDATE_CODE_ARGS[@]}" >/dev/null
  $AWS lambda update-function-configuration "${UPDATE_CONFIG_ARGS[@]}"

  log "Updated"
else
  log "Function ${FUNCTION_NAME} does not exist; creating..."

  $AWS lambda create-function "${CREATE_ARGS[@]}"

  log "Created"
fi
