# /pr-brief quiet

Read this only when the user typed `quiet` (`/pr-brief quiet`, `/pr-brief improve 12 quiet`). It can go with `sign`.

A quiet description keeps the Brief visible and folds the rest. The Change map and the Review guide sit in a
collapsed `<details>` element, inside the markers, right after the Brief:

```
<!-- pr-brief:begin ... -->
## Brief
<one paragraph>

<details>
<summary>Change map and review guide · pr-brief, <model> <effort></summary>

## Change map
...
## Review guide
...

</details>
<!-- pr-brief:end -->
```

Never write the fold by hand. The gate reads the same sections as for any description.

1. Write the block as in step 5. Save it as `draft.md`. Write the Brief so it can stand alone: it is the only part a
   reader sees without a click.
2. Fold it. Name the writer in the summary, after the label: `Change map and review guide · pr-brief, <model> <effort>`,
   for example `... · pr-brief, Claude Opus 5.5 medium`. Write only what you know. With nothing known, leave
   `--summary` out.

   ```bash
   "$PRB" body wrap --managed draft.md --summary "Change map and review guide · pr-brief, <model> <effort>" \
     --host <github.com | dev.azure.com> > draft-quiet.md
   ```

3. Use `draft-quiet.md` as the draft in step 7. For `improve`, save it as `managed.md`.

The gate adds one rule, `quiet`: the fold closes with `</details>` on the last line before the end marker, and a
blank line follows `</summary>` (without it the host shows the Markdown inside as plain text). The Brief rules do
not change: the Brief ends where the fold starts.

Azure DevOps may not collapse `<details>` in a PR description. `body wrap` says so. Look at the PR after you create it.
