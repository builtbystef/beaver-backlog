#!/usr/bin/env bash
#
# Asserts what a snapshot release produced: the six platforms, the archive and
# checksum naming, what each archive carries, and that the link-time flags
# reached the version command. Run it after
# `goreleaser release --snapshot --clean`; CI does exactly that.
#
# The naming is a contract the installers download by, so it is asserted here
# rather than left to whichever defaults GoReleaser happens to ship.

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
dist=${1:-$repo_root/dist}

fail() {
  echo "check-release: $*" >&2
  exit 1
}

[ -f "$dist/metadata.json" ] || fail "no $dist/metadata.json: run goreleaser release --snapshot --clean first"
version=$(jq -r .version "$dist/metadata.json")
if [ -z "$version" ] || [ "$version" = null ]; then
  fail "metadata.json names no version"
fi

# The platform set is spelled out rather than derived from the build, so that
# dropping one from the configuration fails here instead of passing quietly.
platforms="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64"

archive_for() {
  case "$1" in
    windows/*) echo "beaver_${version}_${1%/*}_${1#*/}.zip" ;;
    *) echo "beaver_${version}_${1%/*}_${1#*/}.tar.gz" ;;
  esac
}

# Every archive: the expected name, the expected payload, and a binary whose
# recorded build settings say it was cross-compiled for that platform with cgo
# off. `go version -m` reads that from the binary itself, so it holds for the
# platforms this machine cannot run.
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

for platform in $platforms; do
  goos=${platform%/*}
  goarch=${platform#*/}
  archive=$(archive_for "$platform")
  [ -f "$dist/$archive" ] || fail "missing archive $archive"

  binary=beaver
  if [ "$goos" = windows ]; then
    binary=beaver.exe
  fi
  out="$work/$goos-$goarch"
  mkdir -p "$out"
  case "$archive" in
    *.zip) unzip -q "$dist/$archive" -d "$out" ;;
    *) tar -xzf "$dist/$archive" -C "$out" ;;
  esac
  for want in "$binary" README.md LICENSE; do
    [ -f "$out/$want" ] || fail "$archive is missing $want"
  done

  settings=$(go version -m "$out/$binary")
  for want in "GOOS=$goos" "GOARCH=$goarch" "CGO_ENABLED=0"; do
    grep -q "build[[:space:]]*$want\$" <<<"$settings" || fail "$archive: $binary was not built with $want"
  done
done

want_count=$(wc -w <<<"$platforms")
count=$(find "$dist" -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' \) | wc -l)
[ "$count" -eq "$want_count" ] || fail "found $count archives, want $want_count"

# The checksums file: one SHA-256 line per archive, in the format `sha256sum -c`
# and the installers both read.
checksums="beaver_${version}_checksums.txt"
[ -f "$dist/$checksums" ] || fail "missing $checksums"
lines=$(wc -l <"$dist/$checksums")
[ "$lines" -eq "$want_count" ] || fail "$checksums has $lines lines, want $want_count"
if grep -qvE '^[0-9a-f]{64}  beaver_.+$' "$dist/$checksums"; then
  fail "$checksums has a line that is not '<hash>  <filename>'"
fi
# sha256sum is GNU; macOS spells the same check shasum -a 256.
if command -v sha256sum >/dev/null 2>&1; then
  verify=(sha256sum --check --quiet)
else
  verify=(shasum -a 256 --check --quiet)
fi
(cd "$dist" && "${verify[@]}" "$checksums") || fail "$checksums does not match the archives"

# The link-time flags: the archived binary for this machine names the commit it
# was built from and does not call itself a dev build.
host_os=$(go env GOOS)
host_arch=$(go env GOARCH)
host="$work/$host_os-$host_arch/beaver"
if [ -x "$host" ]; then
  reported=$("$host" version --format json)
  commit=$(jq -r .commit <<<"$reported")
  built_version=$(jq -r .version <<<"$reported")
  head=$(git -C "$repo_root" rev-parse HEAD)
  if [ -z "$commit" ] || [ "${head#"$commit"}" = "$head" ]; then
    fail "version reports commit $commit, want a prefix of $head"
  fi
  [ "$built_version" != dev ] || fail "version reports a dev build, so the version flag never reached the binary"
else
  fail "no archive for this machine ($host_os/$host_arch), so the injected build metadata cannot be run"
fi

echo "check-release: $want_count platforms, $checksums, and injected build metadata all as expected ($version)"
