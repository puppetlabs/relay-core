#!/bin/bash

#
# Commands
#

JQ="${JQ:-jq}"

#
# Variables
#

CONDITIONS_URL="${CONDITIONS_URL:-conditions}"
VALUE_NAME="${VALUE_NAME:-success}"
POLLING_INTERVAL="${POLLING_INTERVAL:-5s}"
POLLING_ITERATIONS="${POLLING_ITERATIONS:-1080}"

#
# Script
#

for i in $(seq ${POLLING_ITERATIONS}); do
  CONDITIONS=$(curl "$METADATA_API_URL/${CONDITIONS_URL}")
  VALUE=$(echo $CONDITIONS | $JQ --arg value "$VALUE_NAME" -r '.[$value]')
  if [ -n "${VALUE}" ]; then
    if [ "$VALUE" = "true" ]; then
      exit 0
    fi
    if [ "$VALUE" = "false" ]; then
      exit 1
    fi
  fi
  sleep ${POLLING_INTERVAL}
done

exit 1