#!/bin/sh
#
# Installs a released beaver binary: no Go toolchain, no sudo, one command.
#
#   curl -fsSL https://beaverbacklog.com/install.sh | sh
#
# It resolves a release, downloads the archive built for this platform, verifies
# its SHA-256 against the release's published checksums file, and unpacks the
# binary into ~/.local/bin. Nothing is installed until the checksum matches.
#
# POSIX sh on purpose: it has to run under dash, ash and busybox as well as
# bash. It never reads stdin, so piping it into a shell behaves exactly like
# running a saved copy.

set -eu

REPO="builtbystef/beaver-backlog"

version=${BEAVER_VERSION:-}
install_dir=${BEAVER_INSTALL_DIR:-}

fail() {
  printf 'install.sh: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage: install.sh [--version VERSION]

Installs the beaver binary from a published GitHub release.

Options:
  --version VERSION   Release to install (default: the latest release).
  -h, --help          Print this help.

Environment:
  BEAVER_VERSION      Same as --version.
  BEAVER_INSTALL_DIR  Where to install (default: ~/.local/bin).
EOF
}

while [ $# -gt 0 ]; do
  case $1 in
    --version)
      [ $# -ge 2 ] || fail "--version needs a value"
      version=$2
      shift 2
      ;;
    --version=*)
      version=${1#--version=}
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      fail "unknown argument: $1"
      ;;
  esac
done

os=$(uname -s)
case $os in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "unsupported operating system: $os" ;;
esac

arch=$(uname -m)
case $arch in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) fail "unsupported architecture: $arch" ;;
esac

# One fetch primitive over whichever tool the machine has, writing to stdout so
# the caller decides between a variable and a file.
if command -v curl >/dev/null 2>&1; then
  http_get() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  http_get() { wget -qO- "$1"; }
else
  fail "neither curl nor wget is installed, and one of them is needed to download the release"
fi

# The archive names carry the version, so the tag has to be known before
# anything can be fetched; the releases API is what names the latest one.
if [ -z "$version" ]; then
  latest=$(http_get "https://api.github.com/repos/$REPO/releases/latest") ||
    fail "could not reach the GitHub releases API to find the latest release"
  version=$(printf '%s\n' "$latest" |
    sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
  [ -n "$version" ] || fail "the GitHub releases API named no latest release of $REPO"
fi

# Tags are v-prefixed; the names inside a release are not.
case $version in
  v*) tag=$version ;;
  *) tag=v$version ;;
esac
version=${tag#v}

archive="beaver_${version}_${os}_${arch}.tar.gz"
checksums="beaver_${version}_checksums.txt"
base="https://github.com/$REPO/releases/download/$tag"

command -v tar >/dev/null 2>&1 || fail "tar is not installed, and it is needed to unpack the release"

tmp=$(mktemp -d 2>/dev/null || mktemp -d -t beaver-install)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

printf 'Downloading beaver %s for %s/%s\n' "$version" "$os" "$arch"
http_get "$base/$archive" >"$tmp/$archive" ||
  fail "could not download $base/$archive: check that release $tag exists and publishes a $os/$arch build"
http_get "$base/$checksums" >"$tmp/$checksums" ||
  fail "could not download $base/$checksums, so the download cannot be verified"

if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$tmp/$archive" | cut -d ' ' -f 1)
elif command -v shasum >/dev/null 2>&1; then
  got=$(shasum -a 256 "$tmp/$archive" | cut -d ' ' -f 1)
else
  fail "neither sha256sum nor shasum is installed, so the download cannot be verified"
fi
want=$(awk -v name="$archive" '$2 == name { print $1 }' "$tmp/$checksums")
[ -n "$want" ] || fail "$checksums has no entry for $archive"
[ "$got" = "$want" ] ||
  fail "checksum mismatch for $archive: got $got, expected $want. Nothing was installed."

tar -xzf "$tmp/$archive" -C "$tmp" || fail "could not unpack $archive"
[ -f "$tmp/beaver" ] || fail "$archive does not contain a beaver binary"

if [ -z "$install_dir" ]; then
  [ -n "${HOME:-}" ] || fail "HOME is not set: name a destination with BEAVER_INSTALL_DIR"
  install_dir=$HOME/.local/bin
fi
mkdir -p "$install_dir" || fail "could not create $install_dir"
chmod +x "$tmp/beaver"
mv "$tmp/beaver" "$install_dir/beaver" || fail "could not install into $install_dir"

printf 'Installed beaver %s to %s\n' "$version" "$install_dir/beaver"
case ":${PATH:-}:" in
  *":$install_dir:"*) ;;
  *)
    # The $PATH in the suggested line is for the reader's shell to expand, not
    # this one, so it stays literal.
    # shellcheck disable=SC2016
    printf '\n%s is not on your PATH. Add this to your shell profile:\n\n    export PATH="%s:$PATH"\n\n' \
      "$install_dir" "$install_dir"
    ;;
esac
