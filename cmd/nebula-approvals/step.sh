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
CONDITION="${CONDITION:-condition}"
POLLING_INTERVAL="${POLLING_INTERVAL:-5s}"
POLLING_ITERATIONS="${POLLING_ITERATIONS:-1080}"

#
# Script
#

for i in $(seq ${POLLING_ITERATIONS}); do
  STATE=$(curl "$METADATA_API_URL/${STATE_URL_PATH}/${STATE_KEY_NAME}")
  VALUE=$(echo $STATE | $JQ --arg value "$VALUE_KEY_NAME" -r '.[$value]')
  CONDITION_VALUE=$(echo $VALUE | $JQ --arg condition "$CONDITION" -r '.[$condition]')
  if [ -n "${CONDITION_VALUE}" ]; then
    if [ "$CONDITION_VALUE" = "true" ]; then
      exit 0
    fi
    if [ "$CONDITION_VALUE" = "false" ]; then
      exit 1
    fi
  fi
  sleep ${POLLING_INTERVAL}
done

exit 1