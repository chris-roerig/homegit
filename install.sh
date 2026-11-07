#!/bin/sh
# homegit installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/chris-roerig/homegit/main/install.sh | sh
#
# Safer alternative (inspect before running):
#   curl -fsSL https://raw.githubusercontent.com/chris-roerig/homegit/main/install.sh -o install.sh
#   less install.sh      # review it
#   sh install.sh
#
# Environment overrides:
#   HOMEGIT_INSTALL_DIR  install location (default: /usr/local/bin if writable,
#                        otherwise $HOME/.local/bin)
#   HOMEGIT_VERSION      specific tag to install (default: latest release)

set -eu

REPO="chris-roerig/homegit"
BINARY="homegit"

info() { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
err()  { printf 'error: %s\n' "$*" >&2; exit 1; }

# --- Detect OS and architecture, map to release asset names ----------------
detect_platform() {
  os=$(uname -s 2>/dev/null || echo unknown)
  arch=$(uname -m 2>/dev/null || echo unknown)

  case "$os" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *) err "unsupported operating system: $os. Windows users: download homegit-windows-amd64.exe from the releases page." ;;
  esac

  case "$arch" in
    x86_64 | amd64) arch="amd64" ;;
    arm64 | aarch64) arch="arm64" ;;
    *) err "unsupported architecture: $arch" ;;
  esac

  ASSET="${BINARY}-${os}-${arch}"
}

# --- Pick an HTTP download tool --------------------------------------------
http_get() {
  # $1 = url, writes body to stdout
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$1"
  else
    err "neither curl nor wget is available"
  fi
}

http_download() {
  # $1 = url, $2 = output path
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    err "neither curl nor wget is available"
  fi
}

# --- Resolve the release tag -----------------------------------------------
resolve_version() {
  if [ "${HOMEGIT_VERSION:-}" != "" ]; then
    VERSION="$HOMEGIT_VERSION"
    return
  fi
  api="https://api.github.com/repos/${REPO}/releases/latest"
  # Extract "tag_name": "vX.Y.Z" without requiring jq.
  VERSION=$(http_get "$api" | grep '"tag_name"' | head -n1 | sed 's/.*"tag_name"[^"]*"\([^"]*\)".*/\1/')
  [ "$VERSION" != "" ] || err "could not determine the latest release tag"
}

# --- Choose an install directory -------------------------------------------
choose_install_dir() {
  if [ "${HOMEGIT_INSTALL_DIR:-}" != "" ]; then
    INSTALL_DIR="$HOMEGIT_INSTALL_DIR"
    return
  fi
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
}

# --- Verify checksum --------------------------------------------------------
sha256_of() {
  # $1 = file; prints the hex digest
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    echo ""
  fi
}

verify_checksum() {
  # $1 = binary path, $2 = checksums file, $3 = asset name
  want=$(grep " ${3}\$" "$2" 2>/dev/null | awk '{print $1}' | head -n1)
  if [ -z "$want" ]; then
    # Some checksum files list the bare asset name without a leading path.
    want=$(grep "${3}" "$2" 2>/dev/null | awk '{print $1}' | head -n1)
  fi
  if [ -z "$want" ]; then
    warn "no checksum found for ${3}; skipping verification"
    return 0
  fi
  got=$(sha256_of "$1")
  if [ -z "$got" ]; then
    warn "no sha256 tool available; skipping checksum verification"
    return 0
  fi
  if [ "$want" != "$got" ]; then
    err "checksum mismatch for ${3} (expected $want, got $got)"
  fi
  info "Checksum verified."
}

main() {
  detect_platform
  resolve_version
  choose_install_dir

  base="https://github.com/${REPO}/releases/download/${VERSION}"
  info "Installing ${BINARY} ${VERSION} (${ASSET}) to ${INSTALL_DIR}"

  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT INT TERM

  http_download "${base}/${ASSET}" "${tmp}/${BINARY}" || err "failed to download ${ASSET}"
  # checksums.txt is best-effort; don't fail the whole install if absent.
  if http_download "${base}/checksums.txt" "${tmp}/checksums.txt" 2>/dev/null; then
    verify_checksum "${tmp}/${BINARY}" "${tmp}/checksums.txt" "${ASSET}"
  else
    warn "could not download checksums.txt; skipping verification"
  fi

  chmod +x "${tmp}/${BINARY}"
  mkdir -p "$INSTALL_DIR"

  target="${INSTALL_DIR}/${BINARY}"
  if mv "${tmp}/${BINARY}" "$target" 2>/dev/null; then
    :
  else
    warn "cannot write to ${INSTALL_DIR} directly; retrying with sudo"
    sudo mv "${tmp}/${BINARY}" "$target" || err "failed to install to ${target}"
  fi

  info ""
  info "✓ Installed ${BINARY} to ${target}"

  # Warn if the install dir isn't on PATH.
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *) warn "${INSTALL_DIR} is not on your PATH. Add it, e.g.:
    echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.profile" ;;
  esac

  info ""
  info "Next steps:"
  info "  homegit setup     # first-time configuration"
  info "  homegit help      # see all commands"
}

main "$@"
