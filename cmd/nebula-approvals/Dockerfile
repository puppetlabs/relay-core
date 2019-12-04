FROM alpine:3
RUN apk --no-cache add bash ca-certificates curl git jq openssh && update-ca-certificates

COPY "./cmd/nebula-approvals/step.sh" "/nebula/step-approvals.sh"
ENTRYPOINT []
CMD ["/bin/bash", "/nebula/step-approvals.sh"]
