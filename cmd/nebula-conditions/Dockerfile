FROM alpine:3
RUN apk --no-cache add bash ca-certificates curl git jq openssh && update-ca-certificates

COPY "./cmd/nebula-conditions/when.sh" "/nebula/when-conditions.sh"
ENTRYPOINT []
CMD ["/bin/bash", "/nebula/when-conditions.sh"]
