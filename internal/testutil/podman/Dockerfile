FROM ubuntu:24.04

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends openssh-server podman && \
    rm -rf /var/lib/apt/lists/* && \
    ssh-keygen -A && \
    passwd -d root

# Podman runs inside Docker in these tests. The vfs driver avoids nested OverlayFS,
# while explicit storage paths ensure its runtime and image data stay writable.
RUN echo '[storage]' > /etc/containers/storage.conf && \
    echo 'driver = "vfs"' >> /etc/containers/storage.conf && \
    echo 'runroot = "/run/containers/storage"' >> /etc/containers/storage.conf && \
    echo 'graphroot = "/var/lib/containers/storage"' >> /etc/containers/storage.conf && \
    echo "PermitRootLogin yes" >> /etc/ssh/sshd_config && \
    echo "PermitEmptyPasswords yes" >> /etc/ssh/sshd_config && \
    echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "UsePAM no" >> /etc/ssh/sshd_config

EXPOSE 22

ENTRYPOINT ["/bin/sh", "-ec", "mkdir -p /run/sshd /run/podman; /usr/sbin/sshd; exec podman system service --time=0 unix:///run/podman/podman.sock"]
