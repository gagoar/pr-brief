#!/usr/bin/env bash
# Maintainer-only: compare internal/stelint against upstream ste-lint.py.
# Needs python3 and (unless STE_UPSTREAM_SCRIPT is set) an authenticated gh.
# Not used by CI or by a plain `go test ./...`.
set -euo pipefail
cd "$(dirname "$0")/.."
STE_PARITY=1 go test ./internal/stelint -run TestParity -count=1 "$@"
