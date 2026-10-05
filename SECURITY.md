# Security

## Report a vulnerability

Use GitHub's private reporting: **Security → Report a vulnerability** in this repository. Do not open a public issue. You will get a first answer within 7 days.

Only the latest release gets fixes.

## What pr-brief does and does not protect

The gate is a **quality control, not a security boundary.** It stops a badly formed PR description. It does not stop a person who wants to bypass it.

- **Hook (Claude Code).** The hook receives each Bash command and each PR tool call. It runs locally. It matches patterns in the command text and never runs that text. It reads only the files that a PR command names, such as `--body-file`. It makes no network calls and keeps no log. If its own input is broken, it lets the call through.
- **GitHub Action.** The Action reads the PR description from `$GITHUB_EVENT_PATH` and the settings from `.pr-brief.json`. It never runs code from the PR. The example workflow checks out the **base commit** (`github.event.pull_request.base.sha`), so a PR cannot loosen its own check by editing `.pr-brief.json`. A PR that adds the first `.pr-brief.json` is checked with the defaults. Anyone who can edit the workflow file in a PR can still remove the check: protect the branch and require the `description` check.
- **Annotations.** Text in `::error` messages is escaped, so a description cannot inject workflow commands.

## Supply chain

The plugin ships five compiled binaries in `bin/`. To check them:

1. CI rebuilds every binary from source and fails if the committed file differs.
2. Each release has a `SHA256SUMS` file and a build-provenance attestation. Check one with `gh attestation verify bin/pr-brief-<os>-<arch> --repo gagoar/pr-brief`.
3. Build it yourself: install the Go version in `.go-version` and run `./build.sh`. The output is byte for byte the same.

The Go code uses only the standard library. There are no third-party modules.
