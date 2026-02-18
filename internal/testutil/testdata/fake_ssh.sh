#!/bin/sh
set -eu

if [ -n "${SSH_TEST_ARGS_FILE:-}" ]; then
  printf '%s\n' "$*" >> "$SSH_TEST_ARGS_FILE"
fi

mode="default"
for arg in "$@"; do
  case "$arg" in
    PreferredAuthentications=publickey) mode="public" ;;
    PreferredAuthentications=password) mode="password" ;;
    PasswordAuthentication=no) mode="knownhost" ;;
  esac
done

stdout_var=""
stderr_var=""
exit_var=""
case "$mode" in
  public)
    stdout_var="SSH_TEST_PUBLIC_STDOUT"
    stderr_var="SSH_TEST_PUBLIC_STDERR"
    exit_var="SSH_TEST_PUBLIC_EXIT"
    ;;
  password)
    stdout_var="SSH_TEST_PASSWORD_STDOUT"
    stderr_var="SSH_TEST_PASSWORD_STDERR"
    exit_var="SSH_TEST_PASSWORD_EXIT"
    ;;
  knownhost)
    stdout_var="SSH_TEST_KNOWNHOST_STDOUT"
    stderr_var="SSH_TEST_KNOWNHOST_STDERR"
    exit_var="SSH_TEST_KNOWNHOST_EXIT"
    ;;
  *)
    stdout_var="SSH_TEST_DEFAULT_STDOUT"
    stderr_var="SSH_TEST_DEFAULT_STDERR"
    exit_var="SSH_TEST_DEFAULT_EXIT"
    ;;
esac

eval "stdout=\${$stdout_var-}"
eval "stderr=\${$stderr_var-}"
eval "exit_code=\${$exit_var-1}"

if [ -n "${stdout-}" ]; then
  printf '%s' "$stdout"
fi
if [ -n "${stderr-}" ]; then
  printf '%s' "$stderr" >&2
fi

exit "$exit_code"
