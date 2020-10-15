FROM kindest/node:v1.17.5
RUN set -eux; \
    curl -sSL -o /usr/local/sbin/runsc https://storage.googleapis.com/gvisor/releases/release/latest/runsc && \
    chmod +x /usr/local/sbin/runsc && \
    curl -sSL -o /usr/local/sbin/containerd-shim-runsc-v1 https://storage.googleapis.com/gvisor/releases/release/latest/containerd-shim-runsc-v1 && \
    chmod +x /usr/local/sbin/containerd-shim-runsc-v1
