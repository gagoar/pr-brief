# /pr-brief hook

Read this for `/pr-brief hook`, and for the one-time offer in step 8 of SKILL.md.

The hook is opt-in. The plugin cannot declare it. `"$PRB" hook install` writes one entry into the user's own
Claude Code `settings.json` (or the Codex `hooks.json`), keeps a copy of the file as it was, and changes
nothing else. The entry runs the real binary, not the launcher script, so it works on Windows too.

## /pr-brief hook

1. `"$PRB" hook status`. It prints `user=`, `repo=` and `asked=`. If `user=true` or `repo=true`, say the hook is
   already on, say where, and stop. Offer `"$PRB" hook remove --scope <user|repo>` only if the user asked to
   turn it off.
2. Ask with your question tool (with none, ask in plain text and wait):
   - **Where?** `user` covers every repository (`~/.claude/settings.json`). `repo` covers this repository
     only (`.claude/settings.json`; choose it only inside a git repository).
   - **Which tool?** Ask only if the user named Codex or you are running in Codex: `--harness codex`.
     Otherwise use Claude Code.
3. Run `"$PRB" hook install --scope <user|repo> [--harness codex]`. Show its output.
4. Tell the user two things. Restart the session: the harness reads the file at start. Run `/pr-brief hook`
   again after a plugin update, because the entry holds the path of the binary. On Codex also say: open
   `/hooks` and trust the pr-brief entry.
5. Run `"$PRB" hook asked`, so the offer in step 8 does not come back.

## The one-time offer (after step 8)

Make the offer only when all of these hold: you are not in CI mode, the run created or updated a PR, and
`"$PRB" hook status` says `user=false`, `repo=false` and `asked=false`. Run `"$PRB" hook asked` first, whatever
the answer, so the user is asked once. Then ask in one short question: "A hand-typed `gh pr create` skips the
gate on this machine. Install the hook that blocks a bad description? (yes / no, /pr-brief hook later)". On yes,
follow `/pr-brief hook` from step 2. On no, say that `/pr-brief hook` installs it later, and go on.
