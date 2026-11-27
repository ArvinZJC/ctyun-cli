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
    sha256sum < "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 < "$1" | awk '{print $1}'
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

# Parse the index with POSIX awk, already used for checksum output. Object keys
# address child nodes by name; array elements retain their published order.
select_release() {
  LC_ALL=C awk -v os="$goos" -v arch="$goarch" -v pin="$channel" '
    function hex4(    digits, i, n) {
      digits = tolower(substr(token, pos, 4))
      if (length(digits) != 4 || digits ~ /[^0-9a-f]/) exit 1
      pos += 4
      for (i = 1; i <= 4; i++)
        n = n * 16 + index("0123456789abcdef", substr(digits, i, 1)) - 1
      return n
    }
    # LC_ALL=C makes %c emit one byte on BSD, GNU and BusyBox awk.
    function unicode_escape(    n, low, bytes) {
      n = hex4()
      if (n >= 55296 && n <= 56319) {
        if (substr(token, pos, 2) != "\\u") exit 1
        pos += 2
        low = hex4()
        if (low < 56320 || low > 57343) exit 1
        n = 65536 + (n - 55296) * 1024 + low - 56320
      } else if (n >= 56320 && n <= 57343) exit 1
      # Some awk implementations cannot store NUL. Preserve it as a control
      # marker so it cannot disappear from a key or a selected field.
      if (n == 0) return "\001"
      if (n < 128) return sprintf("%c", n)
      bytes = sprintf("%c", 128 + n % 64)
      n = int(n / 64)
      if (n < 32) return sprintf("%c", 192 + n) bytes
      bytes = sprintf("%c", 128 + n % 64) bytes
      n = int(n / 64)
      if (n < 16) return sprintf("%c", 224 + n) bytes
      return sprintf("%c%c", 240 + int(n / 64), 128 + n % 64) bytes
    }
    function string_value(    c, s) {
      while (pos <= length(token)) {
        c = substr(token, pos++, 1)
        if (c == "\"") return s
        if (c ~ /[[:cntrl:]]/ && c != "\177") exit 1
        if (c == "\\") {
          c = substr(token, pos++, 1)
          if (c == "u") c = unicode_escape()
          else if (c in escapes) c = escapes[c]
          else if (c != "\\" && c != "/" && c != "\"") exit 1
        }
        s = s c
      }
      exit 1
    }
    # Put newlines around complete JSON tokens. Input records cannot contain a
    # newline, so odd split fields are gaps and may contain only whitespace.
    function advance(    i, status) {
      while (part > count) {
        status = getline
        if (status < 0) exit 1
        if (status == 0) { token = ""; return }
        gsub(/"([^"\\]|\\.)*"|-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?|true|false|null|[][{}:,]/, "\n&\n")
        count = split($0, parts, "\n")
        for (i = 1; i <= count; i += 2)
          if (parts[i] !~ /^[ \t\r]*$/) exit 1
        part = 2
      }
      token = parts[part]
      part += 2
      if (substr(token, 1, 1) == "\"") {
        pos = 2
        decoded = (token ~ /[\\[:cntrl:]]/ ? string_value() : substr(token, 2, length(token) - 2))
        token = "\""
      } else if (token ~ /^[-0-9tfn]/) token = "literal"
    }
    function consume(expected) {
      if (token != expected) exit 1
      advance()
    }
    function parse_value(    id, kind, end, key, n) {
      id = ++nodes
      kind = type[id] = token
      if (kind == "{" || kind == "[") {
        end = (kind == "{" ? "}" : "]")
        advance()
        while (token != end) {
          if (n) consume(",")
          key = ++n
          if (kind == "{") {
            key = decoded
            consume("\"")
            consume(":")
          }
          child[id, key] = parse_value()
        }
        size[id] = n
      } else if (kind == "\"") strings[id] = decoded
      else if (kind != "literal") exit 1
      advance()
      return id
    }
    function field(id, key) { return strings[child[id, key]] }
    BEGIN {
      escapes["b"] = "\b"; escapes["f"] = "\f"; escapes["n"] = "\n"
      escapes["r"] = "\r"; escapes["t"] = "\t"
      part = 2
      advance()
      root = parse_value()
      releases = child[root, "releases"]
      if (token != "" || type[root] != "{" || type[releases] != "[") exit 1
      channels = split(pin == "" ? "stable beta alpha" : pin, order, " ")
      for (k = 1; k <= channels; k++) {
        for (i = 1; i <= size[releases]; i++) {
          release = child[releases, i]
          if (field(release, "channel") != order[k]) continue
          artifacts = child[release, "artifacts"]
          if (type[artifacts] != "[") exit 1
          for (j = 1; j <= size[artifacts]; j++) {
            artifact = child[artifacts, j]
            if (field(artifact, "os") != os || field(artifact, "arch") != arch) continue
            version = field(release, "version")
            url = field(artifact, "url")
            sha = field(artifact, "sha256")
            if (version == "" || url == "" || sha == "" || (version url sha) ~ /[[:cntrl:]]/) exit 1
            printf "%s\t%s\t%s\t%s\n", order[k], version, url, sha
            exit
          }
        }
      }
    }
  '
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

selection="$(select_release < "$tmp/core-index.json")" || die "invalid core-index.json from $root"
channel_label="${channel:-any channel}"
[ -n "$selection" ] || die "no ctyun release found for $goos/$goarch on $channel_label. If you pinned a channel, run unset CTYUN_INSTALL_CHANNEL and retry."

IFS=$'\t' read -r candidate_channel version artifact_url artifact_sha <<< "$selection"
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

install_dir="$(CDPATH='' cd -- "$install_dir" && pwd -P)"
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
  printf "  export PATH='%s':\"\$PATH\"\n" "$quoted_dir"
fi
printf 'Verify the installation and view the available commands:\n  ctyun --version\n  ctyun --help\n'
printf 'To browse product plugins in all release channels, run:\n  ctyun plugin list --available --channel all\n'
