#!/usr/bin/env bash
#
# Copyright (c) 2026 IsArvin.
# This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
#

set -euo pipefail

github_root="https://github.com/ArvinZJC/ctyun-cli/releases/download/core"
gitee_root="https://gitee.com/ArvinZJC/ctyun-cli/releases/download/core"
channel="${CTYUN_INSTALL_CHANNEL:-}"
source="${CTYUN_INSTALL_SOURCE:-auto}"
install_dir="${CTYUN_INSTALL_DIR:-$HOME/.local/bin}"

die() {
  printf 'ctyun install: %s\n' "$*" >&2
  exit 1
}

download() {
  local url="$1"
  local output="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$output" "$url"
  else
    die "curl or wget is required"
  fi
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    die "sha256sum or shasum is required"
  fi
}

join_url() {
  case "$2" in
    http://*|https://*) printf '%s\n' "$2" ;;
    *) printf '%s/%s\n' "${1%/}" "$2" ;;
  esac
}

goos="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$goos" in
  darwin) goos="darwin" ;;
  linux) goos="linux" ;;
  *) die "unsupported OS: $goos" ;;
esac

goarch="$(uname -m | tr '[:upper:]' '[:lower:]')"
case "$goarch" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) die "unsupported architecture: $goarch" ;;
esac

case "$source" in
  auto) roots="$github_root $gitee_root" ;;
  github) roots="$github_root" ;;
  gitee) roots="$gitee_root" ;;
  *) die "CTYUN_INSTALL_SOURCE must be auto, github, or gitee" ;;
esac
if [ -n "${CTYUN_INSTALL_BASE_URL:-}" ]; then
  roots="$CTYUN_INSTALL_BASE_URL"
fi

case "$channel" in
  ""|stable|beta|alpha) ;;
  *) die "CTYUN_INSTALL_CHANNEL must be stable, beta, or alpha" ;;
esac

tmp="$(mktemp -d)"
staged=""
cleanup() {
  if [ -n "$staged" ]; then
    rm -f -- "$staged"
  fi
  rm -rf -- "$tmp"
}
trap cleanup EXIT

root=""
for candidate in $roots; do
  if download "${candidate%/}/core-index.json" "$tmp/core-index.json"; then
    root="$candidate"
    break
  fi
done
[ -n "$root" ] || die "could not download core-index.json. Check your connection and CTYUN_INSTALL_SOURCE, then rerun the installer."

channel_order="$channel"
if [ -z "$channel_order" ]; then
  channel_order="stable beta alpha"
fi

selection=""
for candidate_channel in $channel_order; do
  selection="$(
    awk -v want_channel="$candidate_channel" -v want_os="$goos" -v want_arch="$goarch" '
    function value(line) {
      sub(/^[^:]*:[[:space:]]*"/, "", line)
      sub(/".*$/, "", line)
      return line
    }
    /"version":/ { version = value($0) }
    /"channel":/ { in_release = value($0) == want_channel }
    in_release && /"os":/ { artifact_os = value($0) }
    in_release && /"arch":/ { artifact_arch = value($0) }
    in_release && /"url":/ { artifact_url = value($0) }
    in_release && /"sha256":/ {
      artifact_sha = value($0)
      if (artifact_os == want_os && artifact_arch == want_arch) {
        print version "\t" artifact_url "\t" artifact_sha
        exit
      }
    }
    ' "$tmp/core-index.json"
  )"
  [ -n "$selection" ] && break
done
channel_label="${channel:-any channel}"
[ -n "$selection" ] || die "no ctyun release found for $goos/$goarch on $channel_label. If you pinned a channel, run unset CTYUN_INSTALL_CHANNEL and retry."

version="$(printf '%s\n' "$selection" | awk -F '\t' '{print $1}')"
artifact_url="$(printf '%s\n' "$selection" | awk -F '\t' '{print $2}')"
artifact_sha="$(printf '%s\n' "$selection" | awk -F '\t' '{print $3}')"
archive="$tmp/ctyun.tar.gz"
printf 'Downloading ctyun %s (%s, %s/%s)...\n' "$version" "$candidate_channel" "$goos" "$goarch"
download "$(join_url "$root" "$artifact_url")" "$archive" || die "could not download $artifact_url. Check your connection to the release source and retry."

actual_sha="$(sha256_file "$archive")"
[ "$actual_sha" = "$artifact_sha" ] || die "checksum mismatch for $artifact_url. Retry from a trusted release source. If it fails again, report the version and source."

extract_dir="$tmp/extract"
mkdir -p "$extract_dir" "$install_dir"
tar -xzf "$archive" -C "$extract_dir" || die "could not extract the archive. Check that tar is available and the temporary directory has free space, then retry."
[ -f "$extract_dir/ctyun" ] || die "archive does not contain ctyun"
[ ! -d "$install_dir/ctyun" ] || die "install target is a directory: $install_dir/ctyun. Set CTYUN_INSTALL_DIR to another writable directory and retry."

install_dir="$(CDPATH= cd -- "$install_dir" && pwd -P)"
staged="$(mktemp "$install_dir/.ctyun-install.XXXXXXXX")"
cp "$extract_dir/ctyun" "$staged" || die "could not copy ctyun. Check free space in the installation directory and retry."
chmod 0755 "$staged"
mv -f -- "$staged" "$install_dir/ctyun"
staged=""

printf 'Installed ctyun %s to %s\n' "$version" "$install_dir/ctyun"
if ! [ "$(command -v ctyun || true)" -ef "$install_dir/ctyun" ]; then
  printf 'In Bash or Zsh, run this in your current shell. To keep it for future sessions, also add it to your shell startup file.\n'
  # Escape embedded single quotes for a literal Bash or Zsh PATH entry.
  quoted_dir="$(printf '%s' "$install_dir" | sed "s/'/'\\\\''/g")"
  printf "  export PATH='%s':%s\n" "$quoted_dir" '"$PATH"'
fi
printf 'Verify the installation and view the available commands:\n  ctyun --version\n  ctyun --help\n'
printf 'To browse product plugins in all release channels, run:\n  ctyun plugin list --available --channel all\n'
