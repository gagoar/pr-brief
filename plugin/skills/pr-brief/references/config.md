# /pr-brief config and the first-run questions

Read this when the settings are all `default` (the first run), or for `/pr-brief config`.

## First-run questions

Ask once with AskUserQuestion:

1. **Style** for the prose:
   - `ste+iceberg` (default, for developers): ASD-STE100 rewrite, then iceberg, then lint.
   - `ste`: ASD-STE100 rewrite, then lint.
   - `iceberg`: iceberg only.
2. **Scope**: `user` (only you) or `repo` (committed `.pr-brief.json`, shared by the team).
3. **Earlier description** when improving a PR: `fold` (the default: keep text that pr-brief did not write in
   a collapsed section), `comment` (keep it as a hidden HTML comment) or `drop` (replace it).
4. **Diagram theme**: `github-dark` (default), `github-light`, `dracula`, `alucard` (Dracula's light
   theme), or a path to a custom theme JSON file (see `examples/theme-custom.json` in https://github.com/gagoar/pr-brief). Check a custom
   file with `"$PRB" theme validate <file>` before saving it.

5. **Diagram format**: how a flow is drawn.
   - `auto` (default): the host decides. Azure DevOps gets plain text (`ascii`), GitHub and GitLab get `mermaid`.
   - `mermaid`: a themed Mermaid chart.
   - `ascii`: plain text in a code fence. It has no colours, so a theme does not apply: a status is a marker
     (`+ ~ - .`) and a risky function keeps its `!`.
   Ask this first when the host is Azure DevOps or unknown, because Mermaid support there depends on the
   version. Offer a self-test: print the smallest Mermaid block (`graph LR` with `A --> B`), ask the user to
   paste it in a draft PR description and say whether it drew a chart or showed source, and save `mermaid`
   or `ascii` from the answer. Settings live per repo (`--scope repo`), so a person on GitHub in one repo and
   Azure DevOps in another gets the right format in each.

6. **Ticket keys**: how your Jira or Linear tickets are written, so that `improve` keeps them and the gate can
   warn. The default is a guess: 2 to 10 letters, a dash and digits (`ABC-123`). It can match a word such as
   `lodash-4`. Better to name the projects. Ask for the project keys (for example "PAY, OPS, ENG") and build
   `(PAY|OPS|ENG)-[0-9]+`. Add `|AB#[0-9]+` for Azure DevOps work items, or `|#[0-9]+` for GitHub issues, when the
   team writes those. The pattern is a regular expression, at most 200 characters, and it must not match an
   empty string. Jira and Linear links are always read, whatever the pattern. Show the user what the pattern
   finds before you save it: `"$PRB" tickets --list --current cur.md --branch <branch>` (it reads the old
   description and the branch).

Save with `"$PRB" config set style <value> --scope <scope>`,
`"$PRB" config set improve.previous <value> --scope <scope>`,
`"$PRB" config set diagram.theme <name-or-path> --scope <scope>`,
`"$PRB" config set diagram.format <auto|mermaid|ascii> --scope <scope>` and
`"$PRB" config set tickets.pattern '<regex>' --scope <scope>`. Skip the last one when the user keeps the default. For the repo scope, show the file and
leave the commit to the user. A repo's theme file must be inside the repo, so CI can read it.

## /pr-brief config

- No argument, or `show`: run `"$PRB" config show` and print it. Then offer to change a setting.
- `style <s>`, `previous <p>`, `theme <t>`, `format <f>` and `tickets <regex>`: `"$PRB" config set ... --scope user`. Add
  `--scope repo` when asked. `theme` sets `diagram.theme`: a built-in name or a path to a theme JSON file. `format` sets `diagram.format`:
  `auto`, `mermaid` or `ascii`.
  `tickets` sets `tickets.pattern`. An empty value, `config set tickets.pattern ''`, goes back to the default.
- To change the settings interactively, ask with AskUserQuestion for the scope, style,
  `improve.previous`, theme, diagram format and ticket keys, then call `config set` for each answer.
- `"$PRB" theme list` shows the built-in themes. `"$PRB" theme show [name|file]` shows the colours and
  the contrast ratios. `"$PRB" theme validate <file>` checks a custom file.

There are only five settings. If the user asks for another (a different node limit, a section name, a
softer gate), say that it is fixed on purpose: `references/convention.md` lists the rules, and
changing one is a plugin release. The one opt-out is per PR: `> pr-brief skipped: <reason>`.
