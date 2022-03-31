FROM golang:1.18-alpine AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -buildvcs=false -a -installsuffix cgo -o /relay/runtime/tools/entrypoint ./cmd/relay-runtime-tools

FROM relaysh/core AS source

FROM gcr.io/distroless/base:debug-nonroot

COPY --from=builder /relay/runtime/tools/entrypoint /relay/runtime/tools/entrypoint
COPY --from=source /usr/local/bin/ni /relay/runtime/tools/ni
