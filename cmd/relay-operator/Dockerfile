FROM golang:1.13-alpine AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
WORKDIR /build
COPY . .
RUN go build -a -installsuffix cgo -mod vendor -o /usr/bin/relay-operator ./cmd/relay-operator

FROM alpine:latest
COPY --from=builder /usr/bin/relay-operator /usr/bin/relay-operator
RUN apk --no-cache add ca-certificates && update-ca-certificates
CMD ["/usr/bin/relay-operator"]
