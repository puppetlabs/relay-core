#!/bin/bash
set -euo pipefail

#
# Commands
#

AWK="${AWK:-awk}"
CURL="${CURL:-curl}"
JQ="${JQ:-jq}"
NI="${NI:-ni}"
SLEEP="${SLEEP:-sleep}"

#
# Variables
#

WORKDIR="${WORKDIR:-/nebula}"

#
#
#

trap 'jobs -p | xargs -I{} kill -- {}' SIGINT SIGTERM EXIT

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

MASTER_URL="$( $NI get -p '{ .masterURL }' )"
[ -z "${MASTER_URL}" ] && usage 'spec: please specify a value for `masterURL`, the HTTP URL to your Jenkins master'

get_header() {
  local REQ_HEADER="$1"

  $AWK -v IGNORECASE=1 -v RS='\r\n' -v header="${REQ_HEADER}" -F': ' '$1 == header { print $2 }'
}

# Check whether this seems like Jenkins
JENKINS_VERSION="$( $CURL -sI "${MASTER_URL}/login" | get_header X-Jenkins )"
[ -z "${JENKINS_VERSION}" ] && {
  err 'spec: value for `masterURL` does not seem like a Jenkins server (missing X-Jenkins header)'
}

log "detected Jenkins version ${JENKINS_VERSION}"

CREDENTIALS_METHOD="$( $NI get -p '{ .credentials.method }' )"
case "${CREDENTIALS_METHOD}" in
http)
  CREDENTIALS_HTTP="$( $NI get -p '{ .credentials.user }:{ .credentials.token }' )"
  ;;
ssh)
  err 'spec: SSH credentials method is not currently implemented'
  ;;
*)
  err 'spec: unknown credentials method `'"${CREDENTIALS_METHOD}"'`; please specify one of `http` or `ssh`'
  ;;
esac

do_request() {
  $CURL -s -u "${CREDENTIALS_HTTP}" "$@"
}

do_csrf() {
  # https://wiki.jenkins.io/display/JENKINS/Remote+access+API
  do_request -f "${MASTER_URL}/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,':',//crumb)"
}

do_api() {
  local REQ_METHOD="$1"
  local REQ_URL="$2"
  local REQ_DATA="${3:-}"
  local REQ_ADDITIONAL_OPTS=(${@:4})

  declare -a CURL_OPTS

  local CSRF_CRUMB="$( do_csrf || true )"
  [ -n "${CSRF_CRUMB}" ] && CURL_OPTS+=( -H "${CSRF_CRUMB}" )

  CURL_OPTS+=( -X "${REQ_METHOD}" )
  [ -n "${REQ_DATA}" ] && CURL_OPTS+=( -d "${REQ_DATA}" )
  [[ ${REQ_ADDITIONAL_OPTS[@]} ]] && CURL_OPTS+=( "${REQ_ADDITIONAL_OPTS[@]}" )
  CURL_OPTS+=( "${REQ_URL}" )

  do_request -f "${CURL_OPTS[@]}"
}

do_api_master() {
  local REQ_METHOD="$1"
  local REQ_PATH="$2"
  local REQ_DATA="${3:-}"
  local REQ_ADDITIONAL_OPTS=(${@:4})

  do_api "${REQ_METHOD}" "${MASTER_URL}/${REQ_PATH##/}" "${REQ_DATA}" "${REQ_ADDITIONAL_OPTS[@]}"
}

JENKINS_USER="$( do_api_master GET /me/api/json | $JQ -r '.id' )" || {
  err 'spec: `credentials.user` and `credentials.token` do not appear to successfully authenticate'
}

log "authenticated to Jenkins as ${JENKINS_USER}"

JOB="$( $NI get -p '{ .job }' )"
[ -z "${JOB}" ] && usage 'spec: please specify a value for `job`, the identifier for the Jenkins job to build'

JOB_DISPLAY_NAME_QUOTED=$( do_api_master GET /job/${JOB}/api/json | $JQ '.fullDisplayName' ) || {
  err 'spec: `job` does not appear to be a valid identifier for a Jenkins job'
}

log "building job ${JOB_DISPLAY_NAME_QUOTED}"

JOB_PARAMS="$( $NI get | $JQ -r '[ try .parameters | to_entries[] | @uri "\( .key )=\( .value )" ] | join("&")' )"

BUILD_QUEUE_URL="$( do_api_master POST "/job/${JOB}/buildWithParameters?${JOB_PARAMS}" '' --dump-header - -o /dev/null | get_header Location )" || {
  err 'could not enqueue build'
}

log "enqueued build at ${BUILD_QUEUE_URL}"

BUILD_QUEUE_TIMEOUT_SECS="$( $NI get -p '{ .queueOptions.timeoutSeconds }' )"
[ -n "${BUILD_QUEUE_TIMEOUT_SECS}" ] && log "waiting up to ${BUILD_QUEUE_TIMEOUT_SECS} seconds for build to start"

BUILD_QUEUE_WAIT_INTERVAL=5 # seconds
BUILD_QUEUE_WAITED=0
while true; do
  BUILD_QUEUE_API_START=$SECONDS

  BUILD_QUEUE_DATA="$( do_api GET "${BUILD_QUEUE_URL%%/}/api/json" '' -L )" || {
    err 'failed to retrieve queue data'
  }
  BUILD_URL="$( $JQ -r '.executable.url // empty' <<<"${BUILD_QUEUE_DATA}" )"
  [ -n "${BUILD_URL}" ] && break

  BUILD_QUEUE_API_WAITED=$(( $SECONDS - $BUILD_QUEUE_API_START ))

  # Maybe it's been canceled?
  [[ "$( $JQ -r '.cancelled // false' <<<"${BUILD_QUEUE_DATA}" )" == "true" ]] && err 'build cancelled'

  BUILD_QUEUE_WAITED=$(( $BUILD_QUEUE_WAITED + $BUILD_QUEUE_API_WAITED + $BUILD_QUEUE_WAIT_INTERVAL ))
  [[ $BUILD_QUEUE_WAITED -gt "${BUILD_QUEUE_TIMEOUT_SECS:-3600}" ]] && {
    if [[ "$( $NI get -p '{ .queueOptions.cancelOnTimeout }' )" == "true" ]]; then
      BUILD_QUEUE_ID="$( $JQ -r '.id // empty' <<<"${BUILD_QUEUE_DATA}" )"
      [ -n "${BUILD_QUEUE_ID}" ] && do_api_master POST "/queue/cancelItem?id=${BUILD_QUEUE_ID}" '' -o /dev/null && {
        log 'timed out: canceled build in queue'
      } || {
        log 'timed out: attempted to cancel build in queue, but failed'
      }
    fi

    err 'timed out waiting for build to start'
  }

  $SLEEP $BUILD_QUEUE_WAIT_INTERVAL
done

log "build started at ${BUILD_URL}; the log stream follows"

BUILD_WAIT_INTERVAL=1 # seconds
BUILD_OFFSET=0

# Create a FIFO for log output and start streaming it using `cat`
BUILD_LOG="${WORKDIR}/log"

mkfifo "${BUILD_LOG}"
exec 3<>"${BUILD_LOG}"
( exec 3>&- ; cat "${BUILD_LOG}" & )

while true; do
  BUILD_LOG_DATA="$( do_api GET "${BUILD_URL%%/}/logText/progressiveText?start=${BUILD_OFFSET}" '' --dump-header - -o "${BUILD_LOG}" -L )"
  BUILD_OFFSET="$( get_header X-Text-Size <<<"${BUILD_LOG_DATA}" )"
  [[ "$( get_header X-More-Data <<<"${BUILD_LOG_DATA}" )" == "true" ]] || break

  $SLEEP $BUILD_WAIT_INTERVAL
done

exec 3>&- 3<&-
wait

# Get the final build status from the API
BUILD_RESULT="$( do_api GET "${BUILD_URL%%/}/api/json" '' -L | $JQ -r '.result // empty' )"
log "build complete and returned ${BUILD_RESULT}"

[[ "${BUILD_RESULT}" != "SUCCESS" ]] && err "build failed"
exit 0
