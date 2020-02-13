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
CONDITION_FLAG_KEY_NAME="${CONDITION_FLAG_KEY_NAME:-condition}"
CONDITION_FLAG="${CONDITION_FLAG:-true}"
WAIT="${WAIT:-true}"
POLLING_INTERVAL="${POLLING_INTERVAL:-5s}"
POLLING_ITERATIONS="${POLLING_ITERATIONS:-1080}"

#
# Script
#

for i in $(seq ${POLLING_ITERATIONS}); do
  STATE=$(curl "$METADATA_API_URL/${STATE_URL_PATH}/${STATE_KEY_NAME}")
  VALUE=$(echo $STATE | $JQ --arg value "$VALUE_KEY_NAME" -r '.[$value]')
  CONDITION=$(echo $VALUE | $JQ --arg condition "$CONDITION_KEY_NAME" -r '.[$condition]')
  if [ -n "${CONDITION}" ]; then
    if [ "$CONDITION" = ${CONDITION_FLAG} ]; then
      exit 0
    else
      exit 1
    fi
  fi
  if [ "$WAIT" == "true" ]; then
    sleep ${POLLING_INTERVAL}
  else
    exit 1
  fi
done

exit 1