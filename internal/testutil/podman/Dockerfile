FROM ubuntu:24.04

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends openssh-server podman && \
    rm -rf /var/lib/apt/lists/* && \
    ssh-keygen -A && \
    passwd -d root && \
    printf '[storage]\ndriver = "vfs"\nrunroot = "/run/containers/storage"\ngraphroot = "/var/lib/containers/storage"\n' > /etc/containers/storage.conf && \
    printf '\nPermitRootLogin yes\nPermitEmptyPasswords yes\nPasswordAuthentication yes\nUsePAM no\n' >> /etc/ssh/sshd_config

EXPOSE 22

ENTRYPOINT ["/bin/sh", "-ec", "mkdir -p /run/sshd /run/podman; /usr/sbin/sshd; exec podman system service --time=0 unix:///run/podman/podman.sock"]
