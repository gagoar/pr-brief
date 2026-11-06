# pr-brief

Gives every pull request a readable description: a short brief, Input to Functions to Output diagrams, and a
ranked list of the files to review first. A gate checks that the description meets the convention, in the skill and in CI. To also block
`gh pr create`, `gh pr edit` and `az repos pr` commands locally, opt in to the hook:
`pr-brief hook install --scope user`. The plugin declares no hook itself.

Run `/pr-brief` on a branch, or `/pr-brief improve <PR#>` on an open PR.

## Privacy

The `pr-brief` binary reads local files and git history. It makes no network calls, keeps no log and sends
nothing. The optional hook matches patterns in the command text and never runs that text.

## Support

For help or to report a problem, open an issue at https://github.com/gagoar/pr-brief/issues.

## Source, docs and license

Source, docs and the full README: https://github.com/gagoar/pr-brief. MIT license, as is, without warranty.
