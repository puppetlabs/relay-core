FROM golang:1.13-alpine AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
WORKDIR /build
COPY . .
RUN go build -a -installsuffix cgo -mod vendor -o /usr/bin/nebula-workflow-controller ./cmd/nebula-workflow-controller

FROM alpine:latest
COPY --from=builder /usr/bin/nebula-workflow-controller /usr/bin/nebula-workflow-controller
RUN apk --no-cache add ca-certificates && update-ca-certificates
CMD ["/usr/bin/nebula-workflow-controller"]
