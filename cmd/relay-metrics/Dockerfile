FROM golang:1.17-alpine AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -a -installsuffix cgo -o /usr/bin/relay-metrics ./cmd/relay-metrics

FROM alpine:latest
COPY --from=builder /usr/bin/relay-metrics /usr/bin/relay-metrics
RUN apk --no-cache add ca-certificates && update-ca-certificates
CMD ["/usr/bin/relay-metrics"]

