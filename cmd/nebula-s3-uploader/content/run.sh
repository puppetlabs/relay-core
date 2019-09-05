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

BUCKET="$( $NI get -p '{ .bucket }' )"
[ -z "${BUCKET}" ] && usage 'spec: please specify a value for `.bucket`, the name of the S3 bucket to upload to'

KEY="$( $NI get -p '{ .key }' )"

ni aws config -d "${WORKDIR}/.aws"
eval "$( ni aws env -d "${WORKDIR}/.aws" )"

declare -a CP_ARGS

SOURCE_PATH="$( $NI get -p '{ .sourcePath }' )"
if [ -n "${SOURCE_PATH}" ]; then
  ni git clone -d "${WORKDIR}/repo" || err 'could not clone git repository'
  [[ ! -d "${WORKDIR}/repo" ]] && usage 'spec: please specify `git`, the Git repository to use to resolve the source path'

  SOURCE_PATH="$( realpath "${WORKDIR}/repo/$( $NI get -p '{ .git.name }' )/${SOURCE_PATH}" )"
  if [[ "$?" != 0 ]] || [[ "${SOURCE_PATH}" != "${WORKDIR}/repo/"* ]]; then
    err 'spec: `sourcePath` does not contain a valid reference to a file in the specified repository'
  fi

  [ -d "${SOURCE_PATH}" ] && CP_ARGS+=( --recursive )
else
  SOURCE_PATH=-

  SOURCE_CONTENT="$( $NI get -p '{ .sourceContent }' )"
  [ -z "${SOURCE_CONTENT}" ] && usage 'spec: please specify one of `sourceContent`, inline data to upload, or `sourcePath`, the path within a Git repository to upload'

  [ -z "${KEY}" ] && usage 'spec: please specify `key`, the S3 key to upload to, when using `sourceContent`'
fi

declare -a FILTERS="( $( $NI get | $JQ -r 'try .filters[] | select(.type == "include" or .type == "exclude") | "--\( .type )=\( .pattern )" | @sh' ) )"
[[ ${#FILTERS[@]} -gt 0 ]] && CP_ARGS+=( "${FILTERS[@]}" )

ACL="$( $NI get -p '{ .acl }' )"
[ -n "${ACL}" ] && CP_ARGS+=( "--acl=${ACL}" )

STORAGE_CLASS="$( $NI get -p '{ .storageClass }' )"
[ -n "${STORAGE_CLASS}" ] && CP_ARGS+=( "--storage-class=${STORAGE_CLASS}" )

CONTENT_TYPE="$( $NI get -p '{ .contentType }' )"
[ -n "${CONTENT_TYPE}" ] && CP_ARGS+=( "--content-type=${CONTENT_TYPE}" )

CACHE_CONTROL="$( $NI get -p '{ .cacheControl }' )"
[ -n "${CACHE_CONTROL}" ] && CP_ARGS+=( "--cache-control=${CACHE_CONTROL}" )

CONTENT_DISPOSITION="$( $NI get -p '{ .contentDisposition }' )"
[ -n "${CONTENT_DISPOSITION}" ] && CP_ARGS+=( "--content-disposition=${CONTENT_DISPOSITION}" )

CONTENT_ENCODING="$( $NI get -p '{ .contentEncoding }' )"
[ -n "${CONTENT_ENCODING}" ] && CP_ARGS+=( "--content-encoding=${CONTENT_ENCODING}" )

CONTENT_LANGUAGE="$( $NI get -p '{ .contentLanguage }' )"
[ -n "${CONTENT_LANGUAGE}" ] && CP_ARGS+=( "--content-language=${CONTENT_LANGUAGE}" )

EXPIRES="$( $NI get -p '{ .expires }' )"
[ -n "${EXPIRES}" ] && CP_ARGS+=( "--expires=${EXPIRES}" )

METADATA="$( $NI get | jq '.metadata | select(type == "object")' )"
[ -n "${METADATA}" ] && CP_ARGS+=( "--metadata=${METADATA}" )

$AWS s3 cp \
  "${SOURCE_PATH}" "s3://${BUCKET}/${KEY}" \
  "${CP_ARGS[@]}" \
  --no-follow-symlinks <<<"${SOURCE_CONTENT:-}"

log "Uploaded all requested content to bucket ${BUCKET}"
