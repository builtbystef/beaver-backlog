#!/bin/sh
# Renders og/card.html to public/og.png at the Open Graph size, 1200 x 630.
# Needs Google Chrome or Chromium on the PATH.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
out="$here/../public/og.png"

for candidate in google-chrome google-chrome-stable chromium chromium-browser; do
	if command -v "$candidate" >/dev/null 2>&1; then
		chrome=$candidate
		break
	fi
done
: "${chrome:?no Chrome or Chromium on the PATH}"

"$chrome" --headless=new --hide-scrollbars --allow-file-access-from-files \
	--force-device-scale-factor=1 --window-size=1200,630 \
	--screenshot="$out" "file://$here/card.html" 2>/dev/null

echo "wrote $out"
