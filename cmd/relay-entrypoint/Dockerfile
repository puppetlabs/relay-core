FROM golang:1.14-alpine AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
WORKDIR /build
COPY . .
RUN go build -a -installsuffix cgo -o /var/lib/puppet/relay/entrypoint ./cmd/relay-entrypoint

FROM relaysh/core AS source

FROM gcr.io/distroless/base:debug-nonroot

COPY --from=builder /var/lib/puppet/relay/entrypoint /var/lib/puppet/relay/entrypoint
COPY --from=source /usr/local/bin/ni /var/lib/puppet/relay/ni

ENTRYPOINT ["/var/lib/puppet/relay/entrypoint"]
