# /pr-brief quiet

Read this only when the user typed `quiet` (`/pr-brief quiet`, `/pr-brief improve 12 quiet`). It can go with `sign`.

A quiet description opens with one visible lead sentence. The block from `<!-- pr-brief:begin ... -->` to
`<!-- pr-brief:end -->` sits in a collapsed `<details>` element that is outside the markers:

```
**Tickets:** ...            (the tool adds it, above the lead)

<lead>

<details>
<summary>pr-brief, <model> <effort></summary>

<!-- pr-brief:begin ... -->
...
<!-- pr-brief:end -->

</details>

<the fold, when there is one>
```

Never write the wrapper by hand. The gate reads the same block as for any description.

1. Write the block as in step 5. Save it as `draft.md`.
2. Write the lead: one sentence on what the PR is for, and whether it needs attention. Edit it in step 6 with the
   Brief. On `improve`, run `"$PRB" body lead --current cur.md`. If it prints a lead, a person may have written it:
   keep it and do not rewrite it.
3. Name the writer in the summary: `pr-brief, <model> <effort>`, for example `pr-brief, Claude Opus 5.5 medium`. Write
   only what you know. With nothing known, write `pr-brief`.
4. Wrap the block, then use `draft-quiet.md` as the draft in step 7. For `improve`, save the result as `managed.md`:

   ```bash
   "$PRB" body wrap --managed draft.md --lead "<one sentence>" --summary "pr-brief, <model> <effort>" \
     --host <github.com | dev.azure.com> > draft-quiet.md
   ```

The gate adds one rule, `quiet`: the lead is not empty, the wrapper closes right after the end marker, and a blank
line follows `</summary>` (without it the host shows the Markdown inside as plain text).

Azure DevOps may not collapse `<details>` in a PR description. `body wrap` says so. Look at the PR after you create it.
