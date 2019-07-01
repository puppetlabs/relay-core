#!/bin/bash

set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"

export GOOS=linux
export GOARCH=amd64
export GO111MODULE=on
export CGO_ENABLED=0

cd "$root"

go build -installsuffix cgo -mod readonly -o ./cmd/nebula-slack-notification/main."$GOOS-$GOARCH" ./cmd/nebula-slack-notification/main.go

cd "$root/cmd/nebula-slack-notification"
docker build -t nebula-slack-notification .
