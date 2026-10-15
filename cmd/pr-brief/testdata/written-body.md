<!-- pr-brief:begin v1 style=ste+iceberg theme=github-dark -->
## Brief
Reviewers had to look for the files in the Read these first table. Each file is now a link, so they open it from the description. The link follows the PR to its latest commit, never to one commit. A new command writes the links, and the gate rejects a file with no link or a stale one. The skill adds the links after it creates the PR.

## Change map
### Flow 1: Input 1 -> a description gets linked files, and the gate checks them
```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#0d1117","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#14181e","primaryTextColor":"#e6edf3","primaryBorderColor":"#383d43","nodeTextColor":"#e6edf3","textColor":"#e6edf3","mainBkg":"#14181e","nodeBorder":"#383d43","lineColor":"#3d444d","clusterBkg":"#0d1117","clusterBorder":"#383d43","titleColor":"#9198a1","edgeLabelBackground":"#0d1117"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["Input 1"])
  end
  subgraph FN["Functions"]
    F1["runLinks"]:::added
    F2["RewriteReadFirst"]:::added
    F3["Repo.DiffURL"]:::added
    F4["links.Valid"]:::added
    F5["!gate.Check"]:::risk
  end
  subgraph OUT["Output"]
    O1["Output 1"]
    O2["Output 2"]
  end
  I1 ==> F1
  I1 --> F5
  F1 ==> F2
  F2 ==> F3
  F5 ==> F4
  F2 ==> O1
  F5 --> O2
  F3 ~~~ O1
  F3 ~~~ O2
  classDef default fill:#14181e,stroke:#383d43,stroke-width:0.75px,color:#e6edf3,font-weight:500
  classDef added fill:#12221d,stroke:#3fb950,stroke-width:1px,color:#e6edf3,font-weight:500
  classDef modified fill:#211f18,stroke:#d29922,stroke-width:1px,color:#e6edf3,font-weight:500
  classDef removed fill:#24171c,stroke:#f85149,stroke-width:1px,stroke-dasharray:4 4,color:#e6edf3,font-weight:500
  classDef context fill:#14181e,stroke:#383d43,stroke-width:0.75px,color:#9198a1,font-weight:500
  classDef risk fill:#14181e,stroke:#f85149,stroke-width:2px,color:#e6edf3,font-weight:500
  classDef riskadded fill:#12221d,stroke:#f85149,stroke-width:2px,color:#e6edf3,font-weight:500
  classDef zone fill:#0d1117,stroke:#383d43,stroke-width:1px,color:#9198a1,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,6 stroke:#3d444d,stroke-width:1px
  linkStyle 0,2,3,4,5 stroke:#3d444d,stroke-width:2px
```
**Legend:** 🟩 added · 🔴 risk (! and a thick red border)

| Ref | What | Detail |
|---|---|---|
| Input 1 | `pr-brief links` and `pr-brief gate` | The skill runs `links` on a description. The gate checks the result. |
| Output 1 | The description with linked files | Before the PR exists, a file links to the branch. After it, a file links to its diff in the PR. |
| Output 2 | The gate findings | A row fails when it has no link or links to another file. It also fails when the link shows an older state: one commit, a range of commits or an older iteration. |

## Review guide
**What changed**:
- `runLinks`: reads a description and the remote, and prints the description with linked files.
- `RewriteReadFirst`: finds the Read-these-first table and links each row.
- `Repo.DiffURL`: builds the link to a file's diff in a PR, for GitHub and Azure DevOps.
- `links.Valid`: checks that a link points at its file and shows the whole PR, not one commit or a range.
- `gate.Check`: fails a row with no link, a wrong link, or a link to an older state.

**Read these first**
| File | Why it is delicate | What to check |
|---|---|---|
| [`internal/links/links.go`](https://github.com/gagoar/pr-brief/pull/14/files#diff-af6d98b39a58930341c268de21116a1df94ea9975c38c62ec43f0f40e7751198) | It builds both host formats and decides which links are valid. | Check the Azure DevOps URLs. No live organisation has tested them. |
| [`internal/gate/check.go`](https://github.com/gagoar/pr-brief/pull/14/files#diff-8e8ae94c60bdf527c9112cbfbd3678c9ea2e9ded559a521610498ba6dffe34b5) | A wrong rule here blocks every PR. | Check the three messages, and that a link to the branch still passes. |
| [`cmd/pr-brief/links.go`](https://github.com/gagoar/pr-brief/pull/14/files#diff-de4075d4f9041056ae98204bb015801f2e4d71d7976f794f1c141d3e89857f00) | It reads the remote and picks the form of the link. | Check the order: `--pr` first, then `--branch`, then the current branch. |
| [`skills/pr-brief/SKILL.md`](https://github.com/gagoar/pr-brief/pull/14/files#diff-409dc0398e959a0fcf5b9fc22de1837cf33cd68ec753b1b252a4a99395c29bc5) | The agent follows it to link, to create the PR and to upgrade the links. | Check step 7 and step 8. |

**Review order**: Start at `runLinks`, then follow the arrows to `Repo.DiffURL`.
<details><summary>Other changed files (23) · tests: 3 · docs: 12 · generated: 0</summary>

- `.claude-plugin/` (1 file)
- `.github/` (1 file)
- `./` (1 file)
- `bin/` (5 files)
- `cmd/pr-brief/` (2 files)
- `docs/` (7 files)
- `internal/gate/` (1 file)
- `internal/host/` (1 file)
- `internal/initfiles/templates/` (1 file)
- `internal/links/` (1 file)
- `skills/pr-brief/references/` (2 files)

</details>
<!-- pr-brief:end -->

