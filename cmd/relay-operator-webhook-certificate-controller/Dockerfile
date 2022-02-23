FROM golang:1.17-alpine AS builder
ENV CGO_ENABLED 0
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -a -installsuffix cgo -o /usr/bin/relay-operator-webhook-certificate-controller ./cmd/relay-operator-webhook-certificate-controller

FROM alpine:latest
COPY --from=builder /usr/bin/relay-operator-webhook-certificate-controller /usr/bin/relay-operator-webhook-certificate-controller
RUN apk --no-cache add ca-certificates && update-ca-certificates
CMD ["/usr/bin/relay-operator-webhook-certificate-controller"]
