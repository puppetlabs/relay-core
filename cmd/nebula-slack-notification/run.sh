#!/bin/bash

set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"

docker run --rm -e "SLACK_TOKEN=$SLACK_TOKEN" nebula-slack-notification -channel '#bmaher' -message 'Hello from nebula-slack-notification run.sh'
