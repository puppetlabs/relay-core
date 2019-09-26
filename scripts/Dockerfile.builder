FROM golang:1.13-alpine
ENV GO111MODULE on
ENV CGO_ENABLED 0
RUN apk update && apk --no-cache add ca-certificates && update-ca-certificates
WORKDIR /go/src/github.com/puppetlabs/nebula-tasks
COPY . .
RUN go build -a -installsuffix cgo -mod vendor -o ni ./include/ni
