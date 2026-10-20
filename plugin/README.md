# pr-brief

Gives every pull request a readable description: a short brief, Input to Functions to Output diagrams, and a
ranked list of the files to review first. A hook blocks `gh pr create`, `gh pr edit` and `az repos pr`
commands whose description does not meet the convention.

Run `/pr-brief` on a branch, or `/pr-brief improve <PR#>` on an open PR.

## Privacy

The `pr-brief` binary reads local files and git history. It makes no network calls, keeps no log and sends
nothing. The hook matches patterns in the command text and never runs that text.

## Support

For help or to report a problem, open an issue at https://github.com/gagoar/pr-brief/issues.

## Source, docs and license

Source, docs and the full README: https://github.com/gagoar/pr-brief. MIT license, as is, without warranty.
