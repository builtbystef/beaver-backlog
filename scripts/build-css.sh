#!/usr/bin/env bash
#
# Compiles the design system's source stylesheet into the asset the binary
# embeds. Run it after editing internal/web/styles/tailwind.css or any template,
# and commit the result: building beaver needs Go and nothing else, so the CLI
# below is a developer's tool, never part of `go build` (ADR 0006).
#
# The CLI is pinned by version and verified by checksum; a floating "latest"
# would rewrite the committed stylesheet on someone else's machine for reasons
# nobody chose. It is cached outside the repository so the working tree stays
# clean and a second run costs no download.

set -euo pipefail

VERSION="v4.3.3"

# sha256, as published in the release's sha256sums.txt.
checksum_for() {
  case "$1" in
    tailwindcss-linux-x64)        echo dc61b3ac6b8c9ca874c0cc4c57b2409791a64c5540404ca5f5367360babc313a ;;
    tailwindcss-linux-x64-musl)   echo a04d34ceacc8f52cbe8920ad846cdeb61d3d0021dba32db0d1f77c9d9fad7a6c ;;
    tailwindcss-linux-arm64)      echo 55fd0b241214eff3de1e8ee4f22796662f2d2e7a49bcfca7477cfd0bac398195 ;;
    tailwindcss-linux-arm64-musl) echo 71ea4be79c9de9827545682df3e040053fb535d37c71ed2cfdedf9385a0868e0 ;;
    tailwindcss-macos-x64)        echo 7922e0953f2110c05976e3bf58f14e643d90427575e766b7d433f5f80cbee7e1 ;;
    tailwindcss-macos-arm64)      echo cdf646702987a743464dff4d9c60fd4480d1c1e73dd819a9a67f1078815dce9d ;;
    tailwindcss-windows-x64.exe)  echo e0e260ce048014e9268f6237ff18f8ccf02cef521cbd0ae04e82c2cdf7aa3955 ;;
    *) return 1 ;;
  esac
}

# asset_name picks the release asset for this machine. Alpine and friends need
# the musl build, which is what the absence of glibc's loader tells us.
asset_name() {
  local os arch
  os=$(uname -s)
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) arch=x64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo "unsupported architecture: $arch" >&2; return 1 ;;
  esac
  case "$os" in
    Linux)
      if [ -n "$(ls /lib/ld-musl-* 2>/dev/null)" ]; then
        echo "tailwindcss-linux-$arch-musl"
      else
        echo "tailwindcss-linux-$arch"
      fi
      ;;
    Darwin) echo "tailwindcss-macos-$arch" ;;
    MINGW*|MSYS*|CYGWIN*) echo "tailwindcss-windows-x64.exe" ;;
    *) echo "unsupported operating system: $os" >&2; return 1 ;;
  esac
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source_css="$repo_root/internal/web/styles/tailwind.css"
output_css="$repo_root/internal/web/assets/tailwind.css"

asset=$(asset_name)
want=$(checksum_for "$asset")
cache="${XDG_CACHE_HOME:-$HOME/.cache}/beaver-backlog/tailwindcss"
cli="$cache/$asset-$VERSION"

if [ ! -x "$cli" ]; then
  echo "fetching Tailwind CLI $VERSION ($asset)"
  mkdir -p "$cache"
  tmp="$cli.download.$$"
  trap 'rm -f "$tmp"' EXIT
  curl -fsSL -o "$tmp" \
    "https://github.com/tailwindlabs/tailwindcss/releases/download/$VERSION/$asset"
  got=$(sha256_of "$tmp")
  if [ "$got" != "$want" ]; then
    echo "checksum mismatch for $asset: got $got, want $want" >&2
    exit 1
  fi
  chmod +x "$tmp"
  mv "$tmp" "$cli"
  trap - EXIT
fi

"$cli" --input "$source_css" --output "$output_css"
echo "wrote ${output_css#"$repo_root"/}"
