# AGENTS.md

pr-brief is a Go CLI (standard library only) shipped with a Claude Code plugin and a GitHub Action.

Rules an agent cannot infer from the code:

- After any change to Go code, run `./build.sh` and commit `bin/`. CI rebuilds every binary and fails if one differs. Never edit `bin/` by hand.
- Add no third-party Go modules. CI fails if `go.mod` has a `require`.
- Go is pinned in `.go-version`. Builds use `-buildvcs=false`, so the same source gives the same bytes.
- After a change to a theme or the renderer, run `./build.sh gallery` and commit `docs/gallery/`. After a change to `docs/assets/*.html`, run `./build.sh assets`.
- The README's first `mermaid` block must equal the output of `pr-brief diagram` for `internal/diagram/testdata/doc.flow.json` in `github-dark`. Do not edit it by hand.
- The limits live in `internal/convention`. `docs/convention.html` and `skills/pr-brief/references/convention.md` must show the same numbers. A test checks this.
- The footer of every page in `docs/` must show the version in `.claude-plugin/plugin.json`.
- Do not add a setting. There are three on purpose. A new rule is a release, not an option.
- Use `--base origin/main` with `pr-brief shape`. Local `main` can be old.
- Check: `go test ./... && gofmt -l . && go vet ./... && claude plugin validate --strict .`
