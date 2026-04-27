#!/bin/sh
{
set -eu

USAGE="POSIX-portable bootstrap installer for topo.
Downloads a release from the Arm artifactory server and places the binary
on the current user's PATH.

Usage:
  sh install.sh [--version VERSION] [--path DIRECTORY]

Options:
  --version VERSION   Install a specific version (e.g. v4.0.0). Default: latest.
  --path DIRECTORY    Install the binary into DIRECTORY instead of auto-detecting."

BASE_URL="https://artifacts.tools.arm.com/topo"
BINARY_NAME="topo"

has_cmd() { 
  command -v "$1" >/dev/null 2>&1; 
}

fetch() {
  if has_cmd curl; then
    curl -fsSL "$1"
  elif has_cmd wget; then
    wget -qO- "$1"
  else
    echo "Error: curl or wget is required" >&2
    exit 1
  fi
}

download() {
  if has_cmd curl; then
    curl -fsSL -o "$2" "$1"
  elif has_cmd wget; then
    wget -qO "$2" "$1"
  else
    echo "Error: curl or wget is required" >&2
    exit 1
  fi
}

parse_args() {
  version=""
  ARG_VERSION=""
  ARG_INSTALL_DIR=""

  while [ $# -gt 0 ]; do
    case "$1" in
      --version)
        [ $# -ge 2 ] || { echo "Error: --version requires a value" >&2; exit 1; }
        ARG_VERSION="$2"; shift 2 ;;
      --path)
        [ $# -ge 2 ] || { echo "Error: --path requires a value" >&2; exit 1; }
        ARG_INSTALL_DIR="$2"; shift 2 ;;
      -h|--help)
        # print the script's header comment, stripping leading "# ", as usage information
        echo "$USAGE"; exit 0 ;;
      *)
        echo "Unknown option: $1" >&2; exit 1 ;;
    esac
  done
}

resolve_version() {
  version="$1"

  if [ -z "$version" ]; then
    echo "Resolving latest version..." >&2
    page="$(fetch "${BASE_URL}/")"
    version="$(echo "$page" | sed -n 's/.*\(v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\).*/\1/p' | sort -u -V | tail -n1)"
    if [ -z "$version" ]; then
      echo "Error: could not determine latest version from ${BASE_URL}/" >&2
      exit 1
    fi
  fi

  case "$version" in
    v*) ;;
    *)  version="v${version}" ;;
  esac

  echo "$version"
}

build_download_url() {
  version="$1"

  case "$(uname -s)" in
    Linux*)  os="linux" ;;
    Darwin*) os="macos" ;;
    *)       echo "Error: unsupported operating system: $(uname -s)" >&2; exit 1 ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64)  arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *)             echo "Error: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac

  case "$os" in
    macos) archive_os="darwin" ;;
    *)     archive_os="$os" ;;
  esac

  archive="${BINARY_NAME}_${archive_os}_${arch}.tar.gz"
  echo "${BASE_URL}/${version}/${os}/${archive}"
}

exit_if_topo_already_installed() {
  existing="$(command -v "$BINARY_NAME" 2>/dev/null || true)"
  if [ -z "$existing" ]; then
    return
  fi

  echo "${BINARY_NAME} is already installed at ${existing}." >&2
  echo "Use '${BINARY_NAME} upgrade' to update the existing installation, or pass --path to download to somewhere else." >&2
  exit 0
}

resolve_install_dir() {
  install_dir="$1"

  if [ -n "$install_dir" ]; then
    mkdir -p "$install_dir" 2>/dev/null || {
      echo "Error: cannot create directory: ${install_dir}" >&2
      exit 1
    }
    echo "$install_dir"
    return
  fi

  preferred_dir="$HOME/.local/bin"

  # try conventional user-local directories already on PATH.
  for candidate in "$preferred_dir" "$HOME/bin"; do
    case ":${PATH}:" in
      *":${candidate}:"*)
        mkdir -p "$candidate" 2>/dev/null && echo "$candidate" && return
        ;;
    esac
  done

  echo "Error: could not find a user-writable directory on PATH." >&2
  echo "Provide one explicitly with --path, or add $preferred_dir to your PATH with the following command:" >&2
  echo "  export PATH=\"\$PATH:$preferred_dir\"" >&2
  echo "To persist across terminal restarts, add the export line to your shell's configuration file (e.g. ~/.bashrc, ~/.zshrc)" >&2
  exit 1
}

download_and_extract() {
  url="$1"
  tmpdir="$2"
  archive="$(basename "$url")"
  dst="${tmpdir}/${archive}"

  echo "Downloading ${url}..." >&2
  download "$url" "$dst"

  tar -xzf "$dst" -C "$tmpdir" "$BINARY_NAME" 2>/dev/null \
    || tar -xzf "$dst" -C "$tmpdir"

  if [ ! -f "${tmpdir}/${BINARY_NAME}" ]; then
    echo "Error: ${BINARY_NAME} binary not found in archive" >&2
    exit 1
  fi
}

install_binary() {
  src="$1"
  install_dir="$2"
  version="$3"

  install -m 0755 "$src" "${install_dir}/${BINARY_NAME}"
  echo "Installed ${BINARY_NAME} ${version} to ${install_dir}/${BINARY_NAME}"

  if has_cmd "$BINARY_NAME"; then
    echo "Run '${BINARY_NAME} --help' to get started"
  else
    abs_dir=$(cd "$install_dir" && pwd -L)
    echo ""
    echo "Warning: ${abs_dir} is not on your PATH"
    echo "Add it to your current session with:"
    echo "  export PATH=\"\$PATH:${abs_dir}\""
    echo "To persist across terminal restarts, add the export line to your shell's configuration file (e.g. ~/.bashrc, ~/.zshrc)"
    echo ""
  fi
}

main() {
  parse_args "$@"

  if [ -z "$ARG_INSTALL_DIR" ]; then
    exit_if_topo_already_installed "$ARG_INSTALL_DIR"
  fi  

  install_dir="$(resolve_install_dir "$ARG_INSTALL_DIR")"
  version="$(resolve_version "$ARG_VERSION")"
  echo "Installing ${BINARY_NAME} ${version}"

  url="$(build_download_url "$version")"

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  download_and_extract "$url" "$tmpdir"
  install_binary "${tmpdir}/${BINARY_NAME}" "$install_dir" "$version"
}

main "$@"
}
