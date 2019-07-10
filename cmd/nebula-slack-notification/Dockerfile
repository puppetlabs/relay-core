FROM alpine:3.9

COPY main.linux-amd64 /bin/nebula-slack-notification

# Temporarily add jq
RUN apk --no-cache add ca-certificates jq
