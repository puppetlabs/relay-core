FROM golang:1.14-alpine AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -a -installsuffix cgo -o /usr/bin/relay-metadata-api ./cmd/relay-metadata-api

FROM alpine:latest
COPY --from=builder /usr/bin/relay-metadata-api /usr/bin/relay-metadata-api
RUN apk --no-cache add ca-certificates && update-ca-certificates
CMD ["/usr/bin/relay-metadata-api"]
