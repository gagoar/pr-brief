# /pr-brief improve

Read this only for `/pr-brief improve` (and the CI rewrite job). Steps 2 to 7 are in `SKILL.md`.

For a PR that exists already: opened in the web UI, opened before this plugin, or flagged by CI.

1. Fetch the PR (pin the account first, see step 0):
   `gh pr view <n> --json number,title,body,baseRefName,headRefName,url`, `az repos pr show --id <n>`, or
   `glab mr view <n> -F json` (the MR object: look for `description`, `source_branch` and `target_branch`;
   check the keys, `glab` does not list them). For the code, `glab mr diff <n>`.
   Save the current description to `cur.md`. Get the code: check out the head branch, or use
   `gh pr diff <n>`. The title stays as it is.
   Then ask whether pr-brief already wrote it: `"$PRB" gate --probe --file cur.md`. It prints
   `written=true` when the description has valid markers and passes every check. If it does, say so and
   ask the user whether to write it again. Stop if the user says no. Skip the question when
   the user said to regenerate or to force it. A description with a `> pr-brief skipped:` line is not
   "written": the author opted out, so ask before you replace it.
2. Find the past writing: `"$PRB" body past --current cur.md`. It prints the description a human wrote
   (empty if there is none). Use it, with the linked issue, as input for the Brief.
3. Run steps 2 to 7 and save the new block (`pr-brief:begin` to `pr-brief:end`) as `managed.md`. For `quiet`, save the output of `body wrap` instead (the block with the fold). The PR
   exists, so in step 7 run `assemble --pr <n> --no-tickets`: the files link to their diff in the PR. `body improve` keeps the tickets (step 4 below).
4. Build the final description. This also writes a local backup first:

   ```bash
   "$PRB" body improve --managed managed.md --current cur.md \
     --host <github.com | dev.azure.com> --owner <owner> --repo <repo> --pr <n> \
     --branch <the PR's head branch> > final.md
   ```

   If the user named a ticket in the request, add `--ticket "Closes ENG-45"`.
   `improve` keeps every Jira and Linear ticket from the old description (also from the hidden earlier-description
   block) and from the branch. It writes them in a `**Tickets:**` line above the begin marker, and it refuses to
   write if one would be lost. Get the head branch with `gh pr view <n> --json headRefName`, or from
   `az repos pr show --id <n>` (`sourceRefName`).

   The mode comes from `improve.previous`. Pass `--mode fold|comment|drop` to override it for this run.
   `fold` (the default) keeps text that pr-brief did not write in a collapsed `Previous description` section
   after the new description. A description pr-brief wrote is never kept.
5. Show the user a before/after diff of the description. Ask the user before writing.
   This changes a shared PR.
6. On approval: `gh pr edit <n> --body-file final.md` (or `az repos pr update --id <n> --description "$(cat final.md)"`,
   or `glab mr update <n> --description-file final.md`).
   Do not pass `--title`.
7. Report the backup path. To undo: `"$PRB" body restore --owner <owner> --repo <repo> --pr <n>` prints
   the oldest saved description (`--at <timestamp prefix>` picks another).
   A `fold` block is already readable: expand it in the PR. To make a hidden `comment` block visible: `"$PRB" body uncomment --current final.md`.

Running `improve` again is safe. The original human text is carried forward unchanged, and blocks
never nest.
