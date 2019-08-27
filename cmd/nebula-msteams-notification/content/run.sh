#!/bin/sh

MESSAGE=$(ni get -p '{.message}')
WEBHOOK_URL=$(ni get -p '{.webhookURL}')

if [ -z "${WEBHOOK_URL}" ]; then
  echo "No webhookURL specified"
  exit 1
fi
if [ -z "${MESSAGE}" ]; then
  echo "No message specified"
  exit 1
fi

URL_PATH="$(echo "${WEBHOOK_URL}" | sed -e "s/https\{0,1\}:\/\/[^\/]*\(\/[^?&#]*\).*/\1/")"

CURL_BODY="{\"TextFormat\": \"markdown\", \"text\": \"${MESSAGE}\"}"

OUTPUT=$(curl -v -f -d "${CURL_BODY}" "${WEBHOOK_URL}" 2>&1)
RETURN=$?
echo "$OUTPUT" | sed -e "s#${URL_PATH}#***WEBHOOK URL REDACTED***#g"
echo "exit ${RETURN}"
exit "$RETURN"
