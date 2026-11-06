---
name: pr-brief
description: >
  Write or improve a pull request description so a reviewer knows what to read: a short conceptual
  brief, Input -> Functions -> Output diagrams (Mermaid or plain text, one per flow), and a ranked list of
  the files that carry logic and risk. Also gates PR descriptions: a hook blocks `gh pr create`,
  `gh pr edit`, `az repos pr create|update`, `glab mr create|update` and the GitHub, GitLab and Azure DevOps
  MCP PR tools when the description does not meet the convention.

  Use when the user opens, updates or asks to improve a PR description ("write the PR description",
  "make this PR easier to review", "improve PR 123", "open a PR"), or when the gate blocks a PR
  command. Commands: /pr-brief, /pr-brief improve <PR# | URL>, /pr-brief config, /pr-brief check <file>,
  /pr-brief ci (set up the CI check for GitHub or Azure DevOps). Add `sign` to any write command
  (`/pr-brief sign`, `/pr-brief improve 12 sign`) to end the description with a visible
  "Written with pr-brief" line. Without `sign`, the description has no such line.

  The PR title is never changed. Only the description is.
argument-hint: "[improve <PR# | URL> | config [show | style <s> | previous <fold|comment|drop> | theme <name|file> | format <auto|mermaid|ascii>] | check <file> | ci] [sign]"
allowed-tools: Read, Agent, AskUserQuestion
---

# /pr-brief

You write the description of a pull request. Output order: **Brief**, then **Change map**, then
**Review guide**. The PR title is never touched.

Load only what the run needs:

- `references/body-template.md`: the exact description shape. Always read it.
- `references/diagram-convention.md`: how a flow is drawn. Read it after step 2, and only when `shape` says
  there is a flow to draw (`smallPR` and `configOnly` are both false).
- `references/diagram-ascii.md`: read it only when the diagram format is `ascii`.
- `references/convention.md`: every fixed rule. Read it only when a gate finding is not clear from its
  message, or when the user asks about a rule.
- `references/improve.md`: read it for `/pr-brief improve`.
- `references/config.md`: read it on the first run (all settings `default`) and for `/pr-brief config`.
- `references/ci.md`: read it for `/pr-brief ci` and for CI mode.

## 0. Set up

Find the binary. Use the first that exists:

```bash
PRB="${CLAUDE_PLUGIN_ROOT}/bin/pr-brief"
[ -x "$PRB" ] || PRB="$(ls ~/.claude/plugins/cache/*/pr-brief/*/bin/pr-brief 2>/dev/null | tail -1)"
[ -x "$PRB" ] || PRB="$(command -v pr-brief)"
"$PRB" version
```

If none works, stop. Tell the user the skill needs the `pr-brief` binary, and how to get it:
install the plugin (`/plugin marketplace add gagoar/gago-plugins`, then
`/plugin install pr-brief@gago-plugins`), or download the binary for their platform from
https://github.com/gagoar/pr-brief/releases, check it against `SHA256SUMS`, and put it on their `PATH`
as `pr-brief`. Do not go on without it: every later step calls it.

Find the host: `git remote get-url origin`. `github.com` means `gh`. `dev.azure.com` or
`visualstudio.com` means `az repos`. A remote with `gitlab` in its name means `glab mr`: a GitLab "merge
request" is this skill's "pull request". A self-managed GitLab on a name without `gitlab` in it is not
detected: ask the user, and pass `--host gitlab` to `pr-brief gate` and `pr-brief diagram`. If `gh` answers 404 for a repo you can see in `gh auth status`,
another logged-in account probably owns the repo. Check `gh auth status`, then run the command as the account that owns the repo
(`gh auth status` lists the accounts; `gh auth --help` shows how to run one command as another account).

The host sets the length limit. A GitHub description holds 65,536 characters at most. An Azure DevOps
description holds 4,000. GitLab uses the GitHub limit until its own is checked. Count characters, not bytes. The gate picks the limit from the host on its own.

**Never lose a Jira or Linear ticket.** Jira and Linear link a PR to a ticket by the key in its description
(`ABC-123`, `ENG-45`, a `/browse/ABC-123` or `linear.app/.../issue/ENG-45` link, or "Closes ENG-45"). A rewrite
that drops the key unlinks the PR. `improve` and `tickets` keep every ticket found in the old description and in
the branch name, in a visible `**Tickets:**` line above the begin marker. Never delete that line, and never run
a command that writes the description without them. If a command says it would lose a ticket, stop and tell
the user.

Never add attribution lines or "generated with" text to a description, a commit or a title. The one
exception is the opt-in signature in step 5, and only when the user typed `sign` or asked for it.

## 1. Settings

```bash
"$PRB" config show --json
```

If `styleSource`, `improve.previousSource`, `diagram.themeSource`, `diagram.formatSource` and `tickets.patternSource` are all `default`,
no one has chosen yet. Read `references/config.md`, ask its questions once, and save the answers. Otherwise use
the settings as they are.

Check that the chosen style's skills exist, and that `greenhorn` is installed for step 5b:

- `ste` and `ste+iceberg` need the `unmistakable:asd-ste100` skill from the `unmistakable` plugin. If it is
  missing, stop and print: `/plugin marketplace add gagoar/gago-plugins` then
  `/plugin install unmistakable@gago-plugins`.
- `iceberg` and `ste+iceberg` need the `iceberg` plugin (`iceberg:edit`). If it is missing, stop and
  print: `/plugin marketplace add gagoar/gago-plugins` then `/plugin install iceberg@gago-plugins`.

- `greenhorn` provides `/greenhorn:test`. If it is missing, stop and print: `/plugin marketplace add gagoar/gago-plugins`
  then `/plugin install greenhorn@gago-plugins`. Never skip step 5b without telling the user.

Never switch to another style without asking.

## 2. Shape

```bash
"$PRB" shape [--base <ref>] [--title "<title>"] [--working-tree]
```

It prints JSON. The tool is deterministic and regex-based, so treat it as a first draft that you
verify. Fields you use: `flows`, `refs`, `extras`, `unplacedFunctions`, `removedFunctions`,
`droppedFlows`, `filesRanked` (each with a `line`), `noiseCounts`, `size`, `smallPR`, `configOnly`, `needsAgents`, `warnings`.

- `configOnly` or `smallPR`: no diagram. Write `<!-- pr-brief:no-diagram: <reason> -->` in the Change
  map, then one visible sentence under it, such as `No diagram: this PR changes 2 code files, 3 or fewer.`
  The marker is a hidden comment, so the sentence tells the reader why the map is empty.
  Skip steps 3 and 4. Do not read `diagram-convention.md`. Still read the changed files to write What changed
  and Read these first.
- `flows` empty but code changed: read `unplacedFunctions`. Group them as one flow named "internal
  changes" only if they share a purpose. Otherwise use the no-diagram marker with the reason.

Otherwise read `references/diagram-convention.md` now (and `diagram-ascii.md` if the format is `ascii`).

## 3. Understand each flow

The cluster agent checks a flow's Input, edges and Outputs against the code. It costs a model run per flow, so
use it where it earns its cost:

The report says which case you are in. Read `needsAgents` and follow it. Do not decide by yourself.

- **`needsAgents` is false** (one flow that `shape` placed cleanly): no agent. Read the flow's diff
  yourself (`git diff <base>...HEAD -- <the flow's files>`) and the code one hop away, and apply the same
  checks (the agent file lists them). The shape output is a regex draft, so do check it.
- **`needsAgents` is true** (two or more flows, or unplaced functions, or any warning): start one
  `pr-brief:pr-brief-cluster` agent per flow (at most 3), plus one for `unplacedFunctions` if there are any.
  Start them in one message so they run in parallel. Give each:
  - the repo path and `base` from the JSON;
  - its flow, and the `refs` entries for its ids;
  - the diff of its own files, pasted in the prompt (`git diff <base>...HEAD -- <the flow's files>`), so it
    does not run `git diff` again;
  - the contract: return the JSON described in the agent file.

Merge their answers into the flows:

- apply `edges` and `outputs` corrections (an agent can fix a wrong Input, drop a false edge or add
  a missing Output);
- keep `sentence` for the What-changed bullets, `detail` for the References table;
- collect `delicateFiles` for the Review guide;
- a point in `unsure` that matters becomes a question to the user, or is left out.

If an agent changes an id or adds a node, re-check the limits in `diagram-convention.md`.

## 4. Draw

You do not write the diagram's style. The tool does, and the gate rejects anything else.

For each flow:

1. Save the flow, with the corrections, as a flow JSON file (`schema/pr-brief-flow.schema.json`; `shape`
   output has this shape). Or skip the file: `"$PRB" diagram --report shape.json --flow-number N`.
2. **Mermaid** (the format is `mermaid`, or `auto` on GitHub and GitLab): do not run `diagram` and do not copy
   its output. Write one line under the heading, `{{diagram flow1.json}}`, and step 7 puts the chart there.
   Run `"$PRB" diagram --flow flow.json` yourself only to see an error: if it refuses the flow (too many
   nodes, a backwards edge), fix the flow, not the output.
   **ASCII**: `"$PRB" diagram --flow flow.json --format ascii` prints a skeleton on stdout and the spec on stderr.
   You draw it: follow `diagram-ascii.md`, paste the skeleton and replace its placeholder line with the drawing.
   Count your boxes against the spec and trace each call from its first box to its second. The gate checks the
   boxes and the width. It cannot check the lines, so a wrong arrow is your mistake to catch.
   Add `--host github|gitlab|azure-devops` to see what `auto` picks, or `--theme <name>` to override the theme.
3. Under `### Flow N: Input 1 -> <what the flow does>`, put the placeholder (or the drawing), then the
   References table as `diagram-convention.md` says.

The Legend line is part of what `diagram` prints, so a placeholder brings it too. For ASCII, paste it
directly under the closing fence, then a blank line, then the References table. Do not write or edit it.

Then add any diagram from `extras`. Those are drawn by hand, are not themed, and are Mermaid. With `ascii`
format, leave the extras out: the host does not draw Mermaid, and the gate refuses a Mermaid fence there.

Parse-check each Mermaid diagram when `npx` exists (plain text needs no check):

```bash
"$PRB" diagram --flow flow.json --raw > /tmp/pr-brief-flow1.mmd
npx -y @mermaid-js/mermaid-cli -i /tmp/pr-brief-flow1.mmd -o /tmp/pr-brief-flow1.svg
```

## 5. Write the description

Follow `body-template.md`. Write it to a temp file outside the repo, for example
`${TMPDIR:-/tmp}/pr-brief-body-<branch>.md`.

- **Brief**: one paragraph, conceptual. What the system now does differently, and the effect. Use the
  past writing and the linked issue for the intent. No file names.
- **Review guide**:
  - Size: one line before What changed, from `size`: `**Size**: <files> files changed, <testFiles> of them
    tests (<testPercent>%)`. A reviewer who sees many files sees at once how many are tests. Add the share of
    changed lines when it differs a lot from the share of files: `, <testLinePercent>% of the changed lines`.
    Leave the line out when `testFiles` is 0.
  - What changed: one bullet per Functions node, grouped by flow, in diagram order.
  - Read these first: at most 7 rows from `filesRanked`. Open each file. Write why it is delicate and
    one thing to check. Drop a row you cannot justify. Write each file as `path/to/file:line` in code font,
    with the row's `line` from `filesRanked`: the link then opens the changed block, not the top of the file.
    Do not write the link yourself: step 7 adds it.
  - Review order: one line.
  - Other changed files and the noise counts in a `<details>` block.
- **Signature** (opt-in): only when the user typed `sign` or asked for the signature. Never by default.
  Put one line just before `<!-- pr-brief:end -->`:
  `<sub>Written with [pr-brief](https://github.com/gagoar/pr-brief) v<version></sub>`. Take `<version>`
  from `"$PRB" version`. A build that ends in `-dev` has no release: write the line without `v<version>`.
  The gate does not require this line. It is not prose: do not edit it in step 6.
- Begin with `<!-- pr-brief:begin v1 style=<style> theme=<theme id> -->` and end with
  `<!-- pr-brief:end -->`. For plain-text diagrams, add `format=ascii` after the theme
  (`<!-- pr-brief:begin v1 style=<style> theme=<theme id> format=ascii -->`). Mermaid needs no `format=`. The theme id is the `id` from `"$PRB" theme show --json`: a built-in name, or
  `custom:<8 hex>` for a theme file.

## 5b. Cold-reader pass (greenhorn)

The Brief and the What-changed bullets are written by someone who knows the code. A newcomer does not. Before the
style pipeline, find where a stranger would get lost, and fix it in the prose.

1. Put the prose items in a scratch file, as in step 6: the Brief, the What-changed bullets, the References "Detail"
   cells and the Review-order line. Never put diagrams, tables or file paths in it.
2. Run `/greenhorn:test` on the scratch file with `--profile adjacent-engineer`. It lists each word, step or jump that
   a reader who writes code but does not know this product cannot follow, and says what the reader thinks the text says.
   It rewrites nothing.
3. Answer each stumble from a source, in this order: the code, the linked issue, the docs, the user. Never invent
   a definition. A stumble with no source goes to a list of open points. Ask the user about it in one question, or
   leave out the sentence that depends on it. In CI mode, never ask: leave the sentence out.
4. Fix the prose, not the shape. Define a term where it first appears, in bold, in the form "X is a Y that does Z", at
   25 words or fewer. Keep the Brief a single paragraph of 5 sentences at most, with no file names: when a definition
   does not fit there, put it in the first What-changed bullet that uses the term. Use one word for one thing.
5. Skip this step when the reader finds nothing to stumble on and its summary matches your intent. Text that already
   reads well gets worse with scaffolding.

`/greenhorn:explain` is a different tool: it rewrites a whole text into a gist, an example and a mechanism. That shape
does not fit this description, so do not use it here.

A model reader is not a human newcomer. This step finds likely stumbles. It does not prove that the text is clear. Do
not write in a description that it does.

## 6. Edit the prose (the style pipeline)

Edit prose only. Never change fences, diagram text, table structure, node ids, file paths, code spans,
HTML comments or markers.

Prose items: the Brief paragraph; the References "Detail" cells; the What-changed bullets; the
"Why it is delicate" and "What to check" cells; the Review-order line.

Put the prose items in a scratch file, one per paragraph. Edit that file, then put each item back.

| Style | Steps |
|---|---|
| `ste+iceberg` | 1. `unmistakable:asd-ste100` skill, STE-flavored mode. 2. `iceberg:edit` with `--no-em-dash --no-weakeners --strip-ai-commentary`. 3. `"$PRB" lint`. |
| `ste` | 1. `unmistakable:asd-ste100` skill, STE-flavored mode. 2. `"$PRB" lint`. |
| `iceberg` | `iceberg:edit` with the same flags. No lint. |

Lint the scratch file: `"$PRB" lint --file <scratch>`. Fix every hard violation and lint again.
Iceberg can bring back constructs that STE bans, such as semicolons. When that happens, fix the
violation and keep iceberg's other cuts.

## 7. Assemble, then gate

One command expands each `{{diagram FILE}}` line into the chart and its Legend, links the **Read these first**
files, and keeps the tickets. It runs the same code as `diagram`, `links` and `tickets`, in that order.

Each file in **Read these first** must be a link, so a reviewer opens it from the description and does not look for
it in the Files changed tab. A row written as `path:line` links to that line: GitHub scrolls the diff to it, and a
branch link opens the file there. Azure DevOps has no link to a line of a pull request diff, so a PR link opens the
file, and the line stays in the label for the reader. The tool writes the links after the prose edit, so the editor
never touches a URL. A link follows the PR to its **latest commit**. The tool never links to a commit, and the gate
rejects a link to a commit, a range of commits or an older iteration.

```bash
"$PRB" assemble --body <draft.md> --branch <branch> > ticketed.md            # no PR yet: links to each file on the branch
"$PRB" assemble --body <draft.md> --pr <n> --no-tickets > final.md           # the PR exists (improve): links to each file's diff
```

For a **new PR** the branch name is the ticket source (`feature/ABC-123-x`, `gago/eng-45-fix`). If the user named a
ticket in the request, add `--ticket "Closes ENG-45"` (repeat for more). The step reads the old description
(`--current cur.md`, when there is one), the branch name and each `--ticket`. It does not read the new text, which
only summarises the diff. For an **existing PR** `body improve` keeps the tickets itself (see `improve.md`), so pass
`--no-tickets` there.

It fails when a flow is in plain text (you draw those by hand and paste them), when a flow file is missing or breaks a
limit, or when a ticket would be lost. Fix the cause and run it again. Use `ticketed.md` from here on.

```bash
"$PRB" gate --file ticketed.md
```

Fix each `FAIL` and run it again. A `warn:` about a ticket in the branch name is not a failure: run `assemble` with
`--branch`. After 3 rounds with failures left, show them to the user and stop. If a finding is not clear from its
message, read `references/convention.md`. Do not edit the gate's rules or weaken the description to pass.

## 8. Create or update the PR

New PR (GitHub): `gh pr create --title "<title>" --body-file <body.md>`. Use the title the user gave.
If none exists, propose one from the commits and ask. Add `--draft` when the user wants a draft.
Azure DevOps: `az repos pr create --title "<title>" --description "$(cat <body.md>)"`.
GitLab: `glab mr create --title "<title>" --description-file <body.md>`.

The hook checks the same body. A hook denial lists the findings; fix them and try again.

**Then upgrade the links.** A new PR had no number when you wrote the body, so its files link to the branch.
Now the PR exists. Take its number (from the URL `gh pr create` prints, or `pullRequestId` in the output of
`az repos pr create`), assemble the same draft again with `--pr <n>`, and update the description. The diff always
shows the latest commit, so the links stay right after more pushes:

```bash
"$PRB" assemble --body <draft.md> --branch <branch> --pr <n> > final.md
gh pr edit <n> --body-file final.md        # or: az repos pr update --id <n> --description "$(cat final.md)", or: glab mr update <n> --description-file final.md
```

This changes only the links in the description you just wrote, so it needs no confirmation. The hook
checks it. Do not pass `--title`.

Existing PR: read `references/improve.md`.

## Other commands

- `/pr-brief improve <PR# | URL>`: read `references/improve.md`.
- `/pr-brief config`: read `references/config.md`. There are only five settings. If the user asks for another (a
  different node limit, a section name, a softer gate), say that it is fixed on purpose: `references/convention.md`
  lists the rules, and changing one is a plugin release. The one opt-out is per PR: `> pr-brief skipped: <reason>`.
- `/pr-brief ci`, and CI mode (a prompt that says "CI mode"): read `references/ci.md`. In CI mode never ask, and
  never skip a missing dependency.

## /pr-brief check <file>

`"$PRB" gate --file <file>`. Print the findings. Do not fix anything unless asked.

## Skipping a PR

For a PR that should not carry a brief (a release PR, an automated bump), put one visible line in the
description: `> pr-brief skipped: <reason>`. The reason is required and stays readable to reviewers.
Do not add it without telling the user.

## The gate, in short

The gate is fixed code, not an AI. It checks the text between the markers (and the total length) and
lists every finding it can. Each finding names a rule: `length`, `skip`, `markers`, `previous`, `sections`,
`brief`, `diagram`, `style`, `references`, `review`, `ste`. The length limit follows the host: 65,536
characters on GitHub, 4,000 on Azure DevOps. The same gate runs as this hook, as a GitHub Action, in an
Azure pipeline, and from `"$PRB" gate --file`. Fix a `diagram` or `style` finding by running `diagram`
again, never by editing the style. The full rule table is `docs/gate.html` in the pr-brief repository.

## When the hook blocks you

The hook denies a PR command whose description fails the gate, or whose description it cannot read.
It reads `--body`, `--body-file`, `--description`, a `$(cat <<'EOF' ... EOF)` heredoc and
`$(cat file)`. It cannot read a shell variable or stdin. Write the description to a file and pass
`--body-file <path>`. Then run `"$PRB" gate --file <path>` to see all findings at once.

