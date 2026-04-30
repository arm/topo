#!/bin/sh
{
set -eu

USAGE="POSIX-portable idempotent installer for topo.
Downloads a release from the Arm artifactory server and installs the application.

Usage:
  sh install.sh [--version VERSION] [--path DIRECTORY]

Options:
  --version VERSION   Install a specific version (e.g. v4.0.0). Default: latest.
  --path DIRECTORY    Install to a custom directory. Default: \$HOME/.local/bin."

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

resolve_install_dir() {
  install_dir="${1:-$HOME/.local/bin}"
  if ! mkdir -p "$install_dir" 2>/dev/null; then
    echo "Error: cannot create directory: ${install_dir}" >&2
    exit 1
  fi
  if ! install_dir="$(cd "$install_dir" && pwd -L)"; then
    echo "Error: cannot resolve directory: ${install_dir}" >&2
    exit 1
  fi
  echo "$install_dir"
}

is_dir_on_path() {
  install_dir="$1"

  case ":${PATH:-}:" in
    *":${install_dir}:"*) return 0 ;;
    *) return 1 ;;
  esac
}

path_setup_command() {
  install_dir="$1"

  case "${SHELL:-}" in
    */zsh)
      echo "echo 'export PATH=\"\$PATH:${install_dir}\"' >> ~/.zshrc; export PATH=\"\$PATH:${install_dir}\""
      ;;
    */bash)
      echo "echo 'export PATH=\"\$PATH:${install_dir}\"' >> ~/.bashrc; export PATH=\"\$PATH:${install_dir}\""
      ;;
    */fish)
      echo "fish_add_path -U \"${install_dir}\""
      ;;
    *)
      echo "echo 'export PATH=\"\$PATH:${install_dir}\"' >> ~/.profile; export PATH=\"\$PATH:${install_dir}\""
      ;;
  esac
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
  echo "Downloaded ${BINARY_NAME} ${version} to ${install_dir}/${BINARY_NAME}"

  if is_dir_on_path "$install_dir"; then
    echo "Run '${BINARY_NAME} --help' to get started"
  else
    echo ""
    echo "Warning: ${install_dir} is not on your PATH"
    echo "Add it for the current session and future sessions with:"
    echo "  $(path_setup_command "$install_dir")"
    echo ""
  fi
}

main() {
  parse_args "$@"

  version="$(resolve_version "$ARG_VERSION")"
  echo "Installing ${BINARY_NAME} ${version}"

  url="$(build_download_url "$version")"
  install_dir="$(resolve_install_dir "$ARG_INSTALL_DIR")"

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  download_and_extract "$url" "$tmpdir"
  install_binary "${tmpdir}/${BINARY_NAME}" "$install_dir" "$version"
}

main "$@"
}
