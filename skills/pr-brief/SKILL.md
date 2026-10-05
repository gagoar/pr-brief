---
name: pr-brief
description: >
  Write or improve a pull request description so a reviewer knows what to read: a short conceptual
  brief, Input -> Functions -> Output Mermaid diagrams (one per flow), and a ranked list of the
  files that carry logic and risk. Also gates PR descriptions: a hook blocks `gh pr create`,
  `gh pr edit`, `az repos pr create|update` and the GitHub/Azure DevOps MCP PR tools when the
  description does not meet the convention.

  Use when the user opens, updates or asks to improve a PR description ("write the PR description",
  "make this PR easier to review", "improve PR 123", "open a PR"), or when the gate blocks a PR
  command. Commands: /pr-brief, /pr-brief improve <PR# | URL>, /pr-brief config, /pr-brief check <file>.

  The PR title is never changed. Only the description is.
argument-hint: "[improve <PR# | URL> | config [show | style <s> | previous <drop|comment>] | check <file>]"
allowed-tools: Bash, Read, Write, Agent, AskUserQuestion
---

# /pr-brief

You write the description of a pull request. Output order: **Brief**, then **Change map**, then
**Review guide**. The PR title is never touched.

Read these before you write anything:

- `references/convention.md`: every fixed rule.
- `references/diagram-convention.md`: how to draw a flow.
- `references/body-template.md`: the exact description shape.

## 0. Set up

Find the binary. Use the first that exists:

```bash
PRB="${CLAUDE_PLUGIN_ROOT}/bin/pr-brief"
[ -x "$PRB" ] || PRB="$(ls ~/.claude/plugins/cache/*/pr-brief/*/bin/pr-brief 2>/dev/null | tail -1)"
"$PRB" version
```

Find the host: `git remote get-url origin`. `github.com` means `gh`. `dev.azure.com` or
`visualstudio.com` means `az repos`. If `gh` answers 404 for a repo you can see in `gh auth status`,
another logged-in account probably owns the repo. Check `gh auth status`, then run the command with
`GH_TOKEN=$(gh auth token --user <account>)`.

Never add attribution lines or "generated with" text to a description, a commit or a title.

## 1. Settings

```bash
"$PRB" config show --json
```

If `styleSource` and `improve.previousSource` are both `default`, no one has chosen yet. Ask once with
AskUserQuestion:

1. **Style** for the prose:
   - `ste+iceberg` (default, for developers): ASD-STE100 rewrite, then iceberg, then lint.
   - `ste`: ASD-STE100 rewrite, then lint.
   - `iceberg`: iceberg only.
2. **Scope**: `user` (only you) or `repo` (committed `.pr-brief.json`, shared by the team).
3. **Earlier description** when improving a PR: `drop` (replace it) or `comment` (keep it as a hidden
   HTML comment).

Save with `"$PRB" config set style <value> --scope <scope>` and
`"$PRB" config set improve.previous <value> --scope <scope>`. For the repo scope, show the file and
leave the commit to the user.

Check that the chosen style's skills exist:

- `ste` and `ste+iceberg` need the `asd-ste100` skill. If it is missing, stop and print:
  `git clone https://github.com/danyuchn/asd-ste100-skill ~/.claude/skills/asd-ste100`
- `iceberg` and `ste+iceberg` need the `iceberg` plugin (`iceberg:edit`). If it is missing, stop and
  print: `/plugin marketplace add gagoar/gago-plugins` then `/plugin install iceberg@gago-plugins`.

Never switch to another style without asking.

## 2. Shape

```bash
"$PRB" shape [--base <ref>] [--title "<title>"] [--working-tree]
```

It prints JSON. The tool is deterministic and regex-based, so treat it as a first draft that you
verify. Fields you use: `flows`, `refs`, `extras`, `unplacedFunctions`, `removedFunctions`,
`droppedFlows`, `filesRanked`, `noiseCounts`, `smallPR`, `configOnly`, `warnings`.

- `configOnly` or `smallPR`: no diagram. Write `<!-- pr-brief:no-diagram: <reason> -->` in the Change
  map. Skip step 3 for flows, but still read the changed files to write What changed and
  Read these first.
- `flows` empty but code changed: read `unplacedFunctions`. Group them as one flow named "internal
  changes" only if they share a purpose. Otherwise use the no-diagram marker with the reason.

## 3. Understand each flow (fan out)

Start one `pr-brief:pr-brief-cluster` agent per flow (at most 3), plus one for `unplacedFunctions` if
there are any. Start them in one message so they run in parallel. Give each:

- the repo path and `base` from the JSON;
- its flow, and the `refs` entries for its ids;
- the contract: return the JSON described in the agent file.

Skip this step for small PRs. Read the files yourself instead.

Merge their answers into the flows:

- apply `edges` and `outputs` corrections (an agent can fix a wrong Input, drop a false edge or add
  a missing Output);
- keep `sentence` for the What-changed bullets, `detail` for the References table;
- collect `delicateFiles` for the Review guide;
- a point in `unsure` that matters becomes a question to the user, or is left out.

If an agent changes an id or adds a node, re-check the limits in `diagram-convention.md`.

## 4. Draw

For each flow, write the heading, the diagram and the References table exactly as
`diagram-convention.md` says. Then add any diagram from `extras`.

Parse-check each diagram when `npx` exists:

```bash
npx -y @mermaid-js/mermaid-cli -i /tmp/pr-brief-flow1.mmd -o /tmp/pr-brief-flow1.svg
```

## 5. Write the description

Follow `body-template.md`. Write it to a temp file outside the repo, for example
`${TMPDIR:-/tmp}/pr-brief-body-<branch>.md`.

- **Brief**: one paragraph, conceptual. What the system now does differently, and the effect. Use the
  past writing and the linked issue for the intent. No file names.
- **Review guide**:
  - What changed: one bullet per Functions node, grouped by flow, in diagram order.
  - Read these first: at most 7 rows from `filesRanked`. Open each file. Write why it is delicate and
    one thing to check. Drop a row you cannot justify.
  - Review order: one line.
  - Other changed files and the noise counts in a `<details>` block.
- Begin with `<!-- pr-brief:begin v1 style=<style> -->` and end with `<!-- pr-brief:end -->`.

## 6. Edit the prose (the style pipeline)

Edit prose only. Never change fences, diagram text, table structure, node ids, file paths, code spans,
HTML comments or markers.

Prose items: the Brief paragraph; the References "Detail" cells; the What-changed bullets; the
"Why it is delicate" and "What to check" cells; the Review-order line.

Put the prose items in a scratch file, one per paragraph. Edit that file, then put each item back.

| Style | Steps |
|---|---|
| `ste+iceberg` | 1. `asd-ste100` skill, STE-flavored mode. 2. `iceberg:edit` with `--no-em-dash --no-weakeners --strip-ai-commentary`. 3. `"$PRB" lint`. |
| `ste` | 1. `asd-ste100` skill, STE-flavored mode. 2. `"$PRB" lint`. |
| `iceberg` | `iceberg:edit` with the same flags. No lint. |

Lint the scratch file: `"$PRB" lint --file <scratch>`. Fix every hard violation and lint again.
Iceberg can bring back constructs that STE bans, such as semicolons. When that happens, fix the
violation and keep iceberg's other cuts.

## 7. Gate

```bash
"$PRB" gate --file <body.md>
```

Fix each `FAIL` and run it again. After 3 rounds with failures left, show them to the user and stop.
Do not edit the gate's rules or weaken the description to pass.

## 8. Create or update the PR

New PR (GitHub): `gh pr create --title "<title>" --body-file <body.md>`. Use the title the user gave.
If none exists, propose one from the commits and ask. Add `--draft` when the user wants a draft.
Azure DevOps: `az repos pr create --title "<title>" --description "$(cat <body.md>)"`.

The hook checks the same body. A hook denial lists the findings; fix them and try again.

Existing PR: see **improve**.

## /pr-brief improve <PR# | URL>

For a PR that exists already: opened in the web UI, opened before this plugin, or flagged by CI.

1. Fetch the PR (pin the account first, see step 0):
   `gh pr view <n> --json number,title,body,baseRefName,headRefName,url`, or `az repos pr show --id <n>`.
   Save the current description to `cur.md`. Get the code: check out the head branch, or use
   `gh pr diff <n>`. The title stays as it is.
2. Find the past writing: `"$PRB" body past --current cur.md`. It prints the description a human wrote
   (empty if there is none). Use it, with the linked issue, as input for the Brief.
3. Run steps 2 to 7 and save the new block (`pr-brief:begin` to `pr-brief:end`) as `managed.md`.
4. Build the final description. This also writes a local backup first:

   ```bash
   "$PRB" body improve --managed managed.md --current cur.md \
     --host github.com --owner <owner> --repo <repo> --pr <n> > final.md
   ```

   The mode comes from `improve.previous`. Pass `--mode drop|comment` to override it for this run.
5. Show the user a before/after diff of the description. Ask with AskUserQuestion before writing.
   This changes a shared PR.
6. On approval: `gh pr edit <n> --body-file final.md` (or `az repos pr update --id <n> --description "$(cat final.md)"`).
   Do not pass `--title`.
7. Report the backup path. To undo: `"$PRB" body restore --owner <owner> --repo <repo> --pr <n>` prints
   the oldest saved description (`--at <timestamp prefix>` picks another).
   To make a hidden `comment` block visible: `"$PRB" body uncomment --current final.md`.

Running `improve` again is safe. The original human text is carried forward unchanged, and blocks
never nest.

## /pr-brief config

- No argument, or `show`: run `"$PRB" config show` and print it. Then offer to change a setting.
- `style <s>` and `previous <p>`: `"$PRB" config set ... --scope user`. Add `--scope repo` when asked.
- To change the settings interactively, ask with AskUserQuestion for the scope, style and
  `improve.previous`, then call `config set` for each answer.

There are only two settings. If the user asks for another (a different node limit, a section name, a
softer gate), say that it is fixed on purpose: `references/convention.md` lists the rules, and
changing one is a plugin release. The one opt-out is per PR: `> pr-brief skipped: <reason>`.

## /pr-brief check <file>

`"$PRB" gate --file <file>`. Print the findings. Do not fix anything unless asked.

## Skipping a PR

For a PR that should not carry a brief (a release PR, an automated bump), put one visible line in the
description: `> pr-brief skipped: <reason>`. The reason is required and stays readable to reviewers.
Do not add it without telling the user.

## When the hook blocks you

The hook denies a PR command whose description fails the gate, or whose description it cannot read.
It reads `--body`, `--body-file`, `--description`, a `$(cat <<'EOF' ... EOF)` heredoc and
`$(cat file)`. It cannot read a shell variable or stdin. Write the description to a file and pass
`--body-file <path>`. Then run `"$PRB" gate --file <path>` to see all findings at once.
