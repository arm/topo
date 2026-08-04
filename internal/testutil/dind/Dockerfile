FROM docker:dind

RUN apk add --no-cache openssh-server util-linux

RUN echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "PermitEmptyPasswords yes" >> /etc/ssh/sshd_config && \
    echo "PermitRootLogin yes" >> /etc/ssh/sshd_config && \
    echo "KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org" >> /etc/ssh/sshd_config

RUN passwd -d root

EXPOSE 22

ENTRYPOINT ["/bin/sh", "-ec", "\
    /usr/sbin/sshd -D & \
    exec /usr/local/bin/dockerd-entrypoint.sh \"$@\" \
    "]
