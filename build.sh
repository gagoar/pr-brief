#!/usr/bin/env bash
# Cross-compile pr-brief into runtime-free binaries under ./bin.
# Needs Go once, on the maintainer or CI machine. Users need nothing installed.
# The Go version is pinned in .go-version so a rebuild reproduces the committed bytes.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$here"
bin="$here/bin"
mkdir -p "$bin"

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
