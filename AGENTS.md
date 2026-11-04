# AGENTS.md

pr-brief is a Go CLI (standard library only) shipped with a Claude Code plugin and a GitHub Action.

Rules an agent cannot infer from the code:

- Never commit a binary. `./build.sh` writes them to `plugin/bin/` (ignored by git) for local use, and CI fails if one is tracked. The release workflow builds the ones users get. Only the launcher script `plugin/bin/pr-brief` is tracked.
- Add no third-party Go modules. CI fails if `go.mod` has a `require`.
- Go is pinned in `.go-version`. Builds use `-buildvcs=false`, so the same source gives the same bytes.
- After a change to a theme or the renderer, run `./build.sh gallery` and commit `docs/gallery/`. After a change to `docs/assets/*.html`, run `./build.sh assets`.
- The README's first `mermaid` block must equal the output of `pr-brief diagram` for `internal/diagram/testdata/doc.flow.json` in `github-dark`. Do not edit it by hand.
- The limits live in `internal/convention`. `docs/convention.html` and `plugin/skills/pr-brief/references/convention.md` must show the same numbers. A test checks this.
- The footer of every page in `docs/` must show the version in `plugin/.claude-plugin/plugin.json`.
- Do not add a setting. There are five on purpose: style, improve.previous, diagram.theme, diagram.format and tickets.pattern. A new rule is a release, not an option.
- A release is a version bump in `plugin/.claude-plugin/plugin.json` merged to main. Never tag by hand: `.github/workflows/release.yml` tests, builds the binaries, tags, publishes and moves `v0`. Check it with `actionlint`.
- The ASCII diagram form is parsed back by the gate. After a change to `internal/diagram/ascii.go`, run `UPDATE_GOLDEN=1 go test ./internal/diagram -run ASCII`, read the diff, and commit `testdata/golden/ascii-*.txt`.
- Use `--base origin/main` with `pr-brief shape`. Local `main` can be old.
- Check: `go test ./... && gofmt -l . && go vet ./... && claude plugin validate --strict .`
