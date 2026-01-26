FROM docker:dind

RUN apk add --no-cache openssh-server

RUN echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "PermitEmptyPasswords yes" >> /etc/ssh/sshd_config && \
    echo "PermitRootLogin yes" >> /etc/ssh/sshd_config && \
    echo "KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org" >> /etc/ssh/sshd_config && \
    echo "DisableForwarding no" >> /etc/ssh/sshd_config && \
    echo "PermitOpen any" >> /etc/ssh/sshd_config && \
    echo "PermitListen any" >> /etc/ssh/sshd_config && \
    echo "" >> /etc/ssh/sshd_config && \
    echo "Match all" >> /etc/ssh/sshd_config && \
    echo "    AllowTcpForwarding yes" >> /etc/ssh/sshd_config && \
    echo "    DisableForwarding no" >> /etc/ssh/sshd_config

RUN mkdir -p /etc/docker && \
    echo "{ \"features\": { \"containerd-snapshotter\": true, \"cdi\": true } }" > /etc/docker/daemon.json

RUN passwd -d root
RUN ssh-keygen -A

EXPOSE 22
EXPOSE 8080

ENTRYPOINT ["/bin/sh", "-ec", "\
    /usr/sbin/sshd -D & \
    exec /usr/local/bin/dockerd-entrypoint.sh \"$@\" \
    "]