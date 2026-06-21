#!/usr/bin/env bash
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

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

root=""
for candidate in $roots; do
  if download "${candidate%/}/core-index.json" "$tmp/core-index.json"; then
    root="$candidate"
    break
  fi
done
[ -n "$root" ] || die "could not download core-index.json"

selection="$(
  awk -v want_channel="$channel" -v want_os="$goos" -v want_arch="$goarch" '
    function value(line) {
      sub(/^[^:]*:[[:space:]]*"/, "", line)
      sub(/".*$/, "", line)
      return line
    }
    /"version":/ { version = value($0) }
    /"channel":/ { in_release = want_channel == "" || value($0) == want_channel }
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
channel_label="${channel:-any channel}"
[ -n "$selection" ] || die "no ctyun release found for $goos/$goarch on $channel_label"

version="$(printf '%s\n' "$selection" | awk -F '\t' '{print $1}')"
artifact_url="$(printf '%s\n' "$selection" | awk -F '\t' '{print $2}')"
artifact_sha="$(printf '%s\n' "$selection" | awk -F '\t' '{print $3}')"
archive="$tmp/ctyun.tar.gz"
download "$(join_url "$root" "$artifact_url")" "$archive"

actual_sha="$(sha256_file "$archive")"
[ "$actual_sha" = "$artifact_sha" ] || die "checksum mismatch for $artifact_url"

extract_dir="$tmp/extract"
mkdir -p "$extract_dir" "$install_dir"
tar -xzf "$archive" -C "$extract_dir"
[ -f "$extract_dir/ctyun" ] || die "archive does not contain ctyun"

if command -v install >/dev/null 2>&1; then
  install -m 0755 "$extract_dir/ctyun" "$install_dir/ctyun"
else
  cp "$extract_dir/ctyun" "$install_dir/ctyun"
  chmod 0755 "$install_dir/ctyun"
fi

printf 'Installed ctyun %s to %s\n' "$version" "$install_dir/ctyun"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'Add %s to PATH before running ctyun from a new shell.\n' "$install_dir" ;;
esac
