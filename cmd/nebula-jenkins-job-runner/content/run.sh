#!/bin/bash
set -euo pipefail

#
# Commands
#

AWK="${AWK:-awk}"
CURL="${CURL:-curl}"
JQ="${JQ:-jq}"
NI="${NI:-ni}"
RM_F="${RM_F:-rm -f}"
SED="${SED:-sed}"
SLEEP="${SLEEP:-sleep}"
XMLSTARLET="${XMLSTARLET:-xmlstarlet}"

#
# Variables
#

WORKDIR="${WORKDIR:-/workspace}"

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

QUEUE_OPT_WAIT_FOR="$( $NI get -p '{ .queueOptions.waitFor }' )"
QUEUE_OPT_WAIT_FOR="${QUEUE_OPT_WAIT_FOR:-build}"

case "${QUEUE_OPT_WAIT_FOR}" in
none|build|downstreams)
  ;;
*)
  err 'spec: invalid value for `queueOptions.waitFor`; please specify one of "none", "build", or "downstreams"'
  ;;
esac

do_request() {
  $CURL -s -u "${CREDENTIALS_HTTP}" "$@"
}

do_csrf() {
  # https://wiki.jenkins.io/display/JENKINS/Remote+access+API
  #
  # Note that we can't concat() as described in the document above, because many
  # Jenkins instances have these type of XPath results denied by policy.
  #
  # The error you'll see:
  #
  # Reason: primitive XPath result sets forbidden; implement
  # jenkins.security.SecureRequester
  do_request -f "${MASTER_URL}/crumbIssuer/api/xml" | \
    $XMLSTARLET sel -t -v '//crumbRequestField' -o ': ' -v '//crumb' -n 2>/dev/null
}

do_api() {
  local REQ_METHOD="$1"
  local REQ_URL="$2"
  local REQ_DATA="${3:-}"
  local REQ_ADDITIONAL_OPTS=(${@:4})

  declare -a CURL_OPTS

  local CSRF_CRUMB_HEADER="$( do_csrf || true )"
  [ -n "${CSRF_CRUMB_HEADER}" ] && CURL_OPTS+=( -H "${CSRF_CRUMB_HEADER}" )

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

QUEUE_URL="$( do_api_master POST "/job/${JOB}/buildWithParameters?${JOB_PARAMS}" '' --dump-header - -o /dev/null | get_header Location )" || {
  err 'could not enqueue build'
}

log "enqueued build at ${QUEUE_URL}"

[[ "${QUEUE_OPT_WAIT_FOR}" == "none" ]] && exit 0

QUEUE_OPT_TIMEOUT_SECS="$( $NI get -p '{ .queueOptions.timeoutSeconds }' )"
QUEUE_OPT_TIMEOUT_SECS="${QUEUE_OPT_TIMEOUT_SECS:-3600}"

QUEUE_OPT_CANCEL_ON_TIMEOUT="$( $NI get -p '{ .queueOptions.cancelOnTimeout }' )"
QUEUE_OPT_CANCEL_ON_TIMEOUT="${QUEUE_OPT_CANCEL_ON_TIMEOUT:-false}"

QUEUE_WAIT_INTERVAL=5 # seconds
BUILD_WAIT_INTERVAL=1 # seconds

WAITS=( "queue:${QUEUE_URL}" )
declare -A SEEN

to_xpath_string() {
  # XPath remarkably doesn't support escaping characters, so we have to use the
  # concat() function to do it for us.
  $SED -e $'s/"/", \'"\', "/g' -e 's/^/concat("/' -e 's/$/", "")/'
}

wait_for_build() {
  local BUILD_URL="$1"

  if [[ -v SEEN[${BUILD_URL%%/}] ]]; then
    log "already visited ${BUILD_URL}, continuing"
    return
  fi
  SEEN[${BUILD_URL%%/}]=1

  # This isn't the same as the job name passed in, because we might be waiting
  # on downstreams-of-downstreams...
  local JOB_URL="$( $SED -e 's#\(.*/job/[^/]*\)/.*#\1#' <<<"${BUILD_URL}" )"
  local JOB_DATA=$( do_api GET "${JOB_URL}/api/json" '' -L ) || {
    err 'unable to find job for build'
  }

  local JOB_NAME="$( jq -r '.name' <<<"${JOB_DATA}" )"

  log "build of ${JOB_NAME} started at ${BUILD_URL}; the log stream follows"

  local BUILD_OFFSET=0

  # Create a FIFO for log output and start streaming it using `cat`
  local BUILD_LOG="${WORKDIR}/log"

  # Just in case a previous run didn't clean this up...
  $RM_F "${BUILD_LOG}"

  mkfifo "${BUILD_LOG}"
  exec 3<>"${BUILD_LOG}"
  ( exec 3>&-; cat "${BUILD_LOG}" & )

  while true; do
    local BUILD_LOG_DATA="$( do_api GET "${BUILD_URL%%/}/logText/progressiveText?start=${BUILD_OFFSET}" '' --dump-header - -o "${BUILD_LOG}" -L )"
    BUILD_OFFSET="$( get_header X-Text-Size <<<"${BUILD_LOG_DATA}" )"
    [[ "$( get_header X-More-Data <<<"${BUILD_LOG_DATA}" )" == "true" ]] || break

    $SLEEP $BUILD_WAIT_INTERVAL
  done

  exec 3>&- 3<&-
  wait

  $RM_F "${BUILD_LOG}"

  # Get the final build status from the API
  local BUILD_DATA="$( do_api GET "${BUILD_URL%%/}/api/json" '' -L )" || {
    err 'failed to retrieve build data'
  }

  local BUILD_ID=$( $JQ -r '.id' <<<"${BUILD_DATA}" )
  local BUILD_RESULT="$( $JQ -r '.result // empty' <<<"${BUILD_DATA}" )"

  log "build complete and returned ${BUILD_RESULT}"
  [[ "${BUILD_RESULT}" == "SUCCESS" ]] || err 'build failed'

  # If we will wait for downstream builds, they should now be in the queue
  if [[ "${QUEUE_OPT_WAIT_FOR}" == "downstreams" ]]; then
    # Find downstreams by looking for actions that have an upstream of this job
    # and build. It seems that some downstreams become enqueued and some just
    # start executing on the same builder node. Not sure why.
    #
    # Also, we have to use XPath for this -- the JSON API doesn't support it.
    local ESCAPED_JOB_NAME="$( to_xpath_string <<<"${JOB_NAME}" )"

    # Queue
    local QUEUE_XPATH="//item[action/cause/upstreamProject=${ESCAPED_JOB_NAME} and action/cause/upstreamBuild=${BUILD_ID}]/id"
    local ESCAPED_QUEUE_XPATH="$( jq -rR '@uri' <<<"${QUEUE_XPATH}" )"

    while IFS= read -r QUEUE_ID; do
      [ -z "${QUEUE_ID}" ] && continue

      WAITS+=( "queue:${MASTER_URL}/queue/item/${QUEUE_ID}" )
    done < <( do_api_master GET "/queue/api/xml?depth=1&xpath=${ESCAPED_QUEUE_XPATH}&wrapper=queue" | $XMLSTARLET sel -t -v '//id/text()' -n 2>/dev/null )

    # Builds
    local BUILD_XPATH="//build[action/cause/upstreamProject=${ESCAPED_JOB_NAME} and action/cause/upstreamBuild=${BUILD_ID}]/url"
    local ESCAPED_BUILD_XPATH="$( jq -rR '@uri' <<<"${BUILD_XPATH}" )"

    while IFS= read -r DOWNSTREAM_JOB_URL; do
      while IFS= read -r BUILD_URL; do
        [ -z "${BUILD_URL}" ] && continue

        WAITS+=( "build:${BUILD_URL}" )
      done < <( do_api GET "${DOWNSTREAM_JOB_URL}/api/xml?depth=1&xpath=${ESCAPED_BUILD_XPATH}&wrapper=build" | $XMLSTARLET sel -t -v '//url/text()' -n 2>/dev/null )
    done < <( jq -r 'try .downstreamProjects[] | .url' <<<"${JOB_DATA}" )
  fi
}

try_cancel_queued() {
  if [[ "${QUEUE_OPT_CANCEL_ON_TIMEOUT}" != "true" ]]; then
    return
  fi

  for WAIT in "${WAITS[@]}"; do
    IFS=':' read -r TYPE URL <<<"${WAIT}"
    [[ "${TYPE}" == "queue" ]] || continue

    local QUEUE_DATA="$( do_api GET "${URL%%/}/api/json" '' -L )" || continue

    local QUEUE_ID="$( $JQ -r '.id // empty' <<<"${QUEUE_DATA}" )"
    [ -n "${QUEUE_ID}" ] || continue
    [ -z "$( $JQ -r '.executable.url // empty' <<<"${QUEUE_DATA}" )" ] || continue

    do_api_master POST "/queue/cancelItem?id=${QUEUE_ID}" '' -o /dev/null && {
      log "timed out: canceled build in queue at item ${QUEUE_ID}"
    } || {
      log "timed out: attempted to cancel build in queue at item ${QUEUE_ID}, but failed"
    }
  done
}

wait_for_queue() {
  local QUEUE_URL="$1"

  log "waiting up to ${QUEUE_OPT_TIMEOUT_SECS} seconds for next build to start"

  local QUEUE_WAITED=0 # seconds
  local QUEUE_ID
  local BUILD_URL

  while true; do
    local API_START=$SECONDS

    local QUEUE_DATA="$( do_api GET "${QUEUE_URL%%/}/api/json" '' -L )" || {
      err 'failed to retrieve queue data'
    }

    QUEUE_ID="$( $JQ -r '.id // empty' <<<"${QUEUE_DATA}" )"
    [ -z "${QUEUE_ID}" ] && err 'could not find item ID from queue data'

    BUILD_URL="$( $JQ -r '.executable.url // empty' <<<"${QUEUE_DATA}" )"
    [ -n "${BUILD_URL}" ] && break

    local API_WAITED=$(( $SECONDS - $API_START ))

    # Maybe it's been canceled?
    if [[ "$( $JQ -r '.cancelled // false' <<<"${QUEUE_DATA}" )" == "true" ]]; then
      err 'build canceled'
    fi

    QUEUE_WAITED=$(( $QUEUE_WAITED + $API_WAITED + $QUEUE_WAIT_INTERVAL ))
    if [[ $QUEUE_WAITED -gt "${QUEUE_OPT_TIMEOUT_SECS}" ]]; then
      try_cancel_queued
      err 'timed out waiting for build to start'
    fi

    $SLEEP $QUEUE_WAIT_INTERVAL
  done

  wait_for_build "${BUILD_URL}"
}

while [[ "${#WAITS[@]}" -gt 0 ]]; do
  IFS=':' read -r TYPE URL <<<"${WAITS[0]}"

  case "${TYPE}" in
  build)
    wait_for_build "${URL}"
    ;;
  queue)
    wait_for_queue "${URL}"
    ;;
  esac

  WAITS=( "${WAITS[@]:1}" )
done
