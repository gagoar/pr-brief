# Security

## Report a vulnerability

Use GitHub's private reporting: **Security → Report a vulnerability** in this repository. Do not open a public issue. You will get a first answer within 7 days.

Only the latest release gets fixes.

## What pr-brief does and does not protect

The gate is a **quality control, not a security boundary.** It stops a badly formed PR description. It does not stop a person who wants to bypass it.

- **Hook (Claude Code, opt-in).** The plugin does not declare it. You install it yourself with `pr-brief hook install`, which writes one entry into your own `settings.json` and keeps a copy of the file as it was. The hook receives each Bash command and each PR tool call. It runs locally. It matches patterns in the command text and never runs that text. It reads only the files that a PR command names, such as `--body-file`. It makes no network calls and keeps no log. If its own input is broken, it lets the call through. The entry that `pr-brief hook install` writes is a shell command. It runs the newest `pr-brief` plugin under `~/.claude/plugins/cache/gago-plugins/` or `~/.claude/plugins/cache/pr-brief/`, or else the newest binary that the launcher downloaded to `~/.cache/pr-brief/`. It looks in no other folder, so a plugin of the same name from another marketplace is never run. `--bin PATH` writes that one path instead.
- **GitHub Action.** The Action reads the PR description from `$GITHUB_EVENT_PATH` and the settings from `.pr-brief.json`. It never runs code from the PR. The example workflow checks out the **base commit** (`github.event.pull_request.base.sha`), so a PR cannot loosen its own check by editing `.pr-brief.json`. A PR that adds the first `.pr-brief.json` is checked with the defaults. Anyone who can edit the workflow file in a PR can still remove the check: protect the branch and require the `description` check.
- **Annotations.** Text in `::error` messages is escaped, so a description cannot inject workflow commands.

## Supply chain

The repository holds no compiled binary. The release workflow builds five of them from the tagged source, and the launcher `plugin/skills/pr-brief/scripts/pr-brief` (and the Action, through the shim `plugin/bin/pr-brief`) downloads the one for the machine, for the plugin's own version, and refuses to run it unless it matches the release's `SHA256SUMS`. That check catches a damaged or swapped file, not a swapped release: `SHA256SUMS` comes from the same place. The build attestation is the independent check. When the GitHub CLI (`gh`) is installed and logged in, the launcher runs `gh attestation verify` on the download and refuses a binary that fails. With no `gh`, or no login, it checks the checksum only and prints a notice. To check a binary:

1. The release workflow, not a person, builds every binary. CI fails if a binary is tracked by git.
2. Each release has a `SHA256SUMS` file and a build-provenance attestation. Check one with `gh attestation verify <the binary> --repo gagoar/pr-brief`. The launcher keeps its download in `~/.cache/pr-brief/v<version>/`.
3. Build it yourself: install the Go version in `.go-version` and run `./build.sh`. The output is byte for byte the same.

The Go code uses only the standard library. There are no third-party modules.
