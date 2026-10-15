# Contributing to pr-brief

Thank you for helping. Please read the [Code of Conduct](CODE_OF_CONDUCT.md) first.

## Set up

You need the Go version in `.go-version`. There are no third-party modules.

```
go test ./...
```

## Before you open a PR

```
gofmt -l .          # must print nothing
go vet ./...
go test ./...
./build.sh          # rebuild bin/ after any Go change, then commit bin/
claude plugin validate --strict .
```

CI rebuilds every binary and fails if one differs from the committed file. Never edit `bin/` by hand. See [`AGENTS.md`](AGENTS.md) for the other generated files (gallery, images).

## The PR description

This repo uses its own tool. Run `/pr-brief` in Claude Code, or fill in the [PR template](.github/pull_request_template.md) and check it with `pr-brief gate --file body.md`. The `description` check blocks a merge when the description fails.

## What we accept

- Bug fixes, with a test that fails without the fix.
- A new built-in theme, with the [theme request form](https://github.com/gagoar/pr-brief/issues/new?template=theme-request.yml). A theme needs a public source for its colours, a licence that allows the use, and 4.5:1 contrast.
- Support for another language in `pr-brief shape`.

## What we do not accept

- A new setting. There are four on purpose. Read [the convention page](https://gagoar.github.io/pr-brief/convention.html). A change to a fixed rule is a proposal for a new release: open an issue first.
- A third-party Go module.

## Security

Report a vulnerability in private. See [SECURITY.md](SECURITY.md).
