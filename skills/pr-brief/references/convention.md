# The convention

pr-brief has two settings: `style` and `improve.previous`. Everything else on this page is fixed.
A rule changes only in a plugin release. The constants live in `internal/convention`.

| Rule | Value |
|---|---|
| PR title | Never changed. Not by `/pr-brief`, not by `improve`. |
| Description parts, in order | `## Brief`, `## Change map`, `## Review guide` |
| Brief | One paragraph, 5 sentences at most, conceptual. No file names. |
| Diagrams per PR | 3 at most. One per flow. |
| Nodes per diagram | 9 at most, context nodes included |
| Edges per diagram | 14 at most |
| Label length | 28 characters at most |
| Context (grey) nodes per diagram | 2 at most |
| Diagram skeleton | `graph LR`, subgraphs `Input`, `Functions`, `Output`, in that order |
| Input and Output nodes | Show only `I<n>` or `O<n>`. A References table explains each. |
| Mermaid keywords | `graph` only. Never `flowchart`, never `click`, never links. |
| Small PR | 3 code files or fewer: no diagram. Write `<!-- pr-brief:no-diagram: <reason> -->`. |
| Config-only PR | No diagram. Same marker. |
| Read these first | 1 to 7 rows, ranked by risk |
| Description length | 65,536 characters at most (the GitHub limit) |
| Prose with style `ste` or `ste+iceberg` | Zero hard STE100 violations |
| iceberg flags | `--no-em-dash --no-weakeners --strip-ai-commentary` |
| Gate | Always on. It always blocks. |
| Writing to a PR | Always ask first. Show a before/after diff. |
| Opt out for one PR | A visible line: `> pr-brief skipped: <reason>` |

## Markers

```
<!-- pr-brief:begin v1 style=ste+iceberg -->
...the description...
<!-- pr-brief:end -->

<!-- pr-brief:previous v1 saved=2026-10-05T14:02Z
...the earlier human description, with every "--" written as "-&#45;"...
pr-brief:previous:end -->
```

The `previous` block exists only when `improve.previous` is `comment`. It follows the end marker.

## What counts as noise

Tests, docs, generated files, lockfiles and binaries never become diagram nodes. The Review guide
reports them as counts: `tests: N · docs: N · generated: N`.

## Risk

A file or function is risky when its path matches auth, secret, credential, password, IAM, RBAC,
key vault, network, firewall, migration, `.proto`, OpenAPI or controllers, or when it changes more
than 300 lines, or when the added code touches locks, transactions, retries, timeouts, crypto or
permissions. Words inside strings and comments do not count.
