FROM alpine:3.21

RUN apk add --no-cache openssh-server

RUN echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "PermitEmptyPasswords yes" >> /etc/ssh/sshd_config && \
    echo "PermitRootLogin yes" >> /etc/ssh/sshd_config && \
    echo "KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org" >> /etc/ssh/sshd_config

RUN passwd -d root
RUN ssh-keygen -A

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D", "-e"]
