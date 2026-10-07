# The description template

The PR title is never touched. Only the description changes.

````
<!-- pr-brief:begin v1 style=ste+iceberg theme=github-dark -->
## Brief
One paragraph, 3 to 5 sentences. Conceptual: the problem or capability, what the system does
differently now, and the effect. No file names. No implementation detail.

## Change map
### Flow 1: Input 1 -> <what the flow does>
<the output of `pr-brief diagram`, with its ```mermaid fence>
| Ref | What | Detail |
|---|---|---|
| Input 1 | ... | ... |
| Output 1 | ... | ... |

### Flow 2: Input 2 -> <what the flow does>
<the output of `pr-brief diagram`, with its ```mermaid fence>
| Ref | What | Detail |
|---|---|---|
| Input 2 | ... | ... |
| Output 1 | see Flow 1 | |

## Review guide
**What changed**:
- `InviteCommand.Handle`: what it does now.
- `CodeGenerator.Next`: what it does now.

**Read these first**
| File | Why it is delicate | What to check |
|---|---|---|
| `src/Invite/CodeGenerator.cs` | It makes the secret code. | Check the random source. |

**Review order**: Start at `InviteCommand.Handle`, then follow the arrows in Flow 1.
<details><summary>Other changed files (14) · tests: 6 · docs: 1 · generated: 0</summary>

- `src/Invite/InviteRequest.cs`
- `src/Invite/InviteResponse.cs`

</details>
<!-- pr-brief:end -->
````

## Rules

- **Brief**: no code spans with paths, no bullets, one paragraph.
- **What changed**: group by flow, in diagram order. One bullet per Functions node, using the node's
  label. Flows with no diagram are listed by name. Folded nodes (`+N more`) list their members.
- **Read these first**: take the top rows of `filesRanked` (at most 7). Read the file, then write why
  it is delicate and one concrete thing to check. A row with no reason you can state is not delicate:
  drop it. The table must keep at least 1 row.
- **Review order**: one line. Name the first thing to read.
- **Other changed files**: everything else, grouped by folder, plus the counts of tests, docs and
  generated files (`noiseCounts`).
- **No diagram**: replace the whole Change map body with
  `<!-- pr-brief:no-diagram: <reason> -->`. Use it for 3 code files or fewer (`smallPR`) and for
  config-only PRs (`configOnly`). Still write What changed and Read these first. For a dependency
  bump, put a table of old and new versions under What changed.
- **Repo PR template**: put its headings after `<!-- pr-brief:end -->`. Fill them from the past
  writing where it answers them. Leave the rest empty. The gate ignores them.
- **Past writing**: with `improve.previous: comment`, `pr-brief body improve` appends the hidden
  `pr-brief:previous` block. Never write that block by hand.
- **Skip**: a line `> pr-brief skipped: <reason>` replaces the whole pr-brief description.
