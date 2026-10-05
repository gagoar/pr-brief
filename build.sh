#!/usr/bin/env bash
# Cross-compile pr-brief into runtime-free binaries under ./bin.
# Needs Go once, on the maintainer or CI machine. Users need nothing installed.
# The Go version is pinned in .go-version so a rebuild reproduces the committed bytes.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$here"
bin="$here/bin"
mkdir -p "$bin"

# ./build.sh gallery   redraw docs/gallery/ from the tool (needs node and network once, for mermaid-cli)
# ./build.sh assets    redraw docs/assets/*.png from docs/assets/*.html (needs Chrome)
case "${1:-}" in
  gallery)
    prb="$here/bin/pr-brief"
    dir="$here/docs/gallery"
    mmdc="npx -y @mermaid-js/mermaid-cli@11"
    sha() { shasum -a 256 "$1" | awk '{print $1}'; }
    entries=""
    for t in $("$prb" theme list | awk '{print $1}'); do
      echo "drawing $t..."
      "$prb" diagram --flow "$dir/flow.json" --theme "$t" --raw > "$dir/$t.mmd" 2>/dev/null
      bg="$("$prb" theme show "$t" --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["derived"]["BG"])')"
      $mmdc -i "$dir/$t.mmd" -o "$dir/$t.svg" -b transparent >/dev/null
      $mmdc -i "$dir/$t.mmd" -o "$dir/$t.png" -s 2 -b "$bg" >/dev/null
      entries="$entries${entries:+,}
    \"$t\": \"$(sha "$dir/$t.mmd")\""
    done
    printf '{\n  "flow": "%s",\n  "renderer": "@mermaid-js/mermaid-cli@11",\n  "themes": {%s\n  }\n}\n' "$(sha "$dir/flow.json")" "$entries" > "$dir/manifest.json"
    echo "done: docs/gallery/ (commit the .mmd, .svg, .png files and manifest.json)"
    exit 0
    ;;
  assets)
    chrome="${CHROME:-}"
    for c in "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" google-chrome chromium chromium-browser; do
      [ -n "$chrome" ] && break
      if [ -x "$c" ] || command -v "$c" >/dev/null 2>&1; then chrome="$c"; fi
    done
    [ -n "$chrome" ] || { echo "build.sh: set CHROME to a Chrome or Chromium binary" >&2; exit 1; }
    shot() { # html width height out
      "$chrome" --headless --hide-scrollbars --force-device-scale-factor=2 --window-size="$2,$3" \
        --screenshot="$here/docs/assets/$4" "file://$here/docs/assets/$1" >/dev/null 2>&1
      echo "wrote docs/assets/$4"
    }
    shot hero.html?mode=light 1100 640 hero-light.png
    shot hero.html?mode=dark 1100 640 hero-dark.png
    shot social.html 1280 640 social.png
    shot og.html 1200 630 og.png
    exit 0
    ;;
esac

want="go$(tr -d '[:space:]' < .go-version)"
have="$(go env GOVERSION)"
if [ "$have" != "$want" ] && [ "${ALLOW_GO_MISMATCH:-}" != "1" ]; then
  echo "build.sh: need $want (see .go-version), have $have. Set ALLOW_GO_MISMATCH=1 to build anyway." >&2
  exit 1
fi

version="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' .claude-plugin/plugin.json | head -1)"
[ -n "$version" ] || { echo "build.sh: no version in .claude-plugin/plugin.json" >&2; exit 1; }

build() {
  local goos="$1" goarch="$2" ext="${3:-}"
  echo "building $goos/$goarch..."
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=$version" \
    -o "$bin/pr-brief-$goos-$goarch$ext" ./cmd/pr-brief
}

build darwin arm64
build darwin amd64
build linux amd64
build linux arm64
build windows amd64 .exe

echo "done: pr-brief $version"
ls -l "$bin" | awk 'NR>1 {print $5, $9}'
