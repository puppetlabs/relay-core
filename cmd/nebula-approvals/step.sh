#!/bin/bash

#
# Commands
#

JQ="${JQ:-jq}"

#
# Variables
#

STATE_URL_PATH="${STATE_URL_PATH:-state}"
STATE_KEY_NAME="${STATE_KEY_NAME:-state}"
VALUE_KEY_NAME="${VALUE_KEY_NAME:-value}"
APPROVAL_KEY_NAME="${APPROVAL_KEY_NAME:-approved}"
APPROVED_FLAG="${APPROVED_FLAG:-true}"
REJECTED_FLAG="${REJECTED_FLAG:-false}"
POLLING_INTERVAL="${POLLING_INTERVAL:-10s}"
POLLING_ITERATIONS="${POLLING_ITERATIONS:-720}"

#
# Script
#

for i in $(seq ${POLLING_ITERATIONS}); do
  STATE=$(curl "$METADATA_API_URL/${STATE_URL_PATH}/${STATE_KEY_NAME}")
  VALUE=$(echo $STATE | $JQ --arg value "$VALUE_KEY_NAME" -r '.[$value]')
  APPROVAL=$(echo $VALUE | $JQ --arg approval "$APPROVAL_KEY_NAME" -r '.[$approval]')
  if [ "$APPROVAL" = ${APPROVED_FLAG} ]; then
    exit 0
  else
    if [ "$APPROVAL" = ${REJECTED_FLAG} ]; then
      exit 1
    fi
  fi
  sleep ${POLLING_INTERVAL}
done

exit 1