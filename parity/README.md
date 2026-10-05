# Parity check against upstream

`internal/stelint` ports `scripts/ste-lint.py` from
github.com/danyuchn/asd-ste100-skill. This optional check proves the Go
output is byte-identical to upstream's `--json` output. It is for
maintainers; normal `go test ./...` needs no Python and skips it.

## Run

```
parity/run.sh
# or
STE_PARITY=1 go test ./internal/stelint -run TestParity -count=1
```

The test is skipped unless `STE_PARITY=1` is set and `python3` is on PATH.

It compares, byte for byte:

- every `internal/stelint/testdata/*.md` and `*.txt` file,
- one run with `--baseline` and `--disable`,
- a list of hand-written Unicode and Markdown snippets (fed on stdin),
- a seeded random corpus (300 inputs by default).

## Upstream script

By default the test downloads `scripts/ste-lint.py` at the pinned commit with
`gh api` into a temp directory. The pinned SHA is the `upstreamSHA` constant in
`internal/stelint/parity_test.go` and in `internal/stelint/UPSTREAM.md`.
It needs read access to a public repo only.

| Variable | Effect |
| --- | --- |
| `STE_PARITY=1` | Turns the test on. |
| `STE_UPSTREAM_SCRIPT=/path/ste-lint.py` | Use a local copy instead of `gh`. |
| `STE_PARITY_N=4000` | Size of the random corpus. |

## Updating the pin

1. Read the upstream diff since the pinned SHA.
2. Port rule or behaviour changes.
3. Change `upstreamSHA` and `UPSTREAM.md`, regenerate `testdata/golden/*.json`
   with the new upstream script, and run `parity/run.sh`.
