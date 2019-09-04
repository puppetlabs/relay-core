#!/bin/bash
set -euo pipefail

# For using `gawk` on macos
AWK="${AWK:-awk}"

# Required
USERNAME="${USERNAME:-$(ni get -p '{.username}')}"
PASSWORD="${PASSWORD:-$(ni get -p '{.password}')}"
URL="${URL:-$(ni get -p '{.url}')}"
ISSUE="${ISSUE:-$(ni get -p '{.issue}')}"

#Optional
RESOLUTION_STATUS="${RESOLUTION_STATUS:-$(ni get -p '{.resolution.status}')}"
RESOLUTION_COMMENT="${RESOLUTION_COMMENT:-$(ni get -p '{.resolution.comment}')}"
if [ -z "${RESOLUTION_STATUS}" ] ; then
  RESOLUTION_STATUS="Closed"
fi
if [ -z "${RESOLUTION_COMMENT}" ] ; then
  RESOLUTION_COMMENT="Resolved by Nebula"
fi

for var in USERNAME PASSWORD URL ISSUE ; do
  if [ -z "${!var}" ]; then
    echo "No ${var,,} specified"
    exit 1
  fi
done

err() {
  echo "error: $@" >&2
  exit 2
}

get_header() {
  local REQ_HEADER="$1"

  $AWK -v IGNORECASE=1 -v RS='\r\n' -v header="${REQ_HEADER}" -F': ' '$1 == header { print $2 }'
}

do_request() {
  curl -L -s -u "${USERNAME}:${PASSWORD}" -H "Content-Type: application/json" "$@"
}

# Only make one call to avoid triggering CAPTCHA
FIRST_CONTACT=$(do_request -D- -X GET "${URL%%/}/rest/api/2/myself")

# Check jira server.
AREQUESTID=$(echo "${FIRST_CONTACT}" | get_header "X-AREQUESTID")
if [ -z "${AREQUESTID}" ] ; then
  err "spec: url does not appear to be a jira instance: ${URL}"
fi

# Check login credentials
LOGIN_REASON=$(echo "${FIRST_CONTACT}" | get_header "X-Seraph-LoginReason")
LOGIN_DENIED_REASON=$(echo "${FIRST_CONTACT}" | get_header 'X-Authentication-Denied-Reason')
if [ "AUTHENTICATED_FAILED" = "${LOGIN_REASON}" ] ; then
  if [ -n "${LOGIN_DENIED_REASON}" ] ; then
    err "spec: Authentication for user/password on ${URL} failed: ${LOGIN_DENIED_REASON}"
  else
    err "spec: Authentication for user/password on ${URL} failed."
  fi
fi

echo "Authenticated..."

# List the possible transitions for this issue type and find the ID of the transition having the name set by RESOLUTION_STATUS
RESOLUTION_ID="$(do_request -X GET "${URL%%/}/rest/api/2/issue/${ISSUE}/transitions" | jq -r ".transitions[] | select(.name == \"${RESOLUTION_STATUS}\") | .id")"

if [ -z "${RESOLUTION_ID}" ] ; then
  err "Cannot find transition ID of any issue status named \"${RESOLUTION_STATUS}\""
fi

echo "Found issue transition ID for \"${RESOLUTION_STATUS}\": ${RESOLUTION_ID}"

# POST a transition using the previously-found ID
PAYLOAD=$(jq -n --arg id "${RESOLUTION_ID}" --arg comment "${RESOLUTION_COMMENT}" '{
  "update": {
    "comment": [
      {
        "add": {
          "body": $comment
        }
      }
    ]
  },
  "transition": {
    "id": $id
  }
}')
do_request -X POST "${URL%%/}/rest/api/2/issue/${ISSUE}/transitions" --data "${PAYLOAD}"

echo "Submitted issue transition..."

ISSUE_STATUS="$(do_request -X GET "${URL%%/}/rest/api/2/issue/${ISSUE}" | jq -r '.fields.status.name')"
if [ "${ISSUE_STATUS}" != "${RESOLUTION_STATUS}" ] ; then
  err "Issue does not appear to have been set to ${RESOLUTION_STATUS}: ${ISSUE_STATUS}"
fi

echo "Issue verified as \"${RESOLUTION_STATUS}\""
