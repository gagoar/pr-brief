# Upstream

The colour model, the mixing rule and the style constants of pr-brief's diagram themes come from
**lukilabs/beautiful-mermaid**, the library behind https://agents.craft.do/mermaid.

| | |
|---|---|
| Repo | https://github.com/lukilabs/beautiful-mermaid |
| Licence | MIT, Copyright (c) 2026 Craft Docs |
| Read at | `2ac8bbbb060ca0a65a6a21f3200bd99b1587b488` (main, 2026-05-06) |
| Files read | `src/theme.ts` (last changed in `02daec0`), `src/styles.ts`, `src/renderer.ts`, `src/layout-engine.ts` |

## What is copied

- **Colours** of `github-light`, `github-dark` and `dracula`: `bg`, `fg`, `line`, `accent`, `muted`, from `THEMES` in `src/theme.ts`.
- **The mixing rule** `MIX`: node fill 3%, node stroke 20%, line 50%, arrow 85%, secondary text 60%, group header 5%, inner stroke 12%, key badge 10%. pr-brief computes each mix to hex, as `color-mix(in srgb, fg N%, bg)` does, because Azure DevOps may not support `color-mix()`. Halves round to even.
- **Style constants:** sharp corners, node stroke 0.75px, group and connector stroke 1px, thick edges 2px, dotted `4 4`, node text 13px at weight 500, group header 12px at weight 600, Inter, orthogonal routing, node spacing 28, layer spacing 48, padding 40.

## What is not from beautiful-mermaid

- **`alucard`** is Dracula's official light theme, from https://draculatheme.com/spec (Alucard Classic, MIT, Copyright (c) 2023 Dracula Theme). It is mapped the way Craft maps `dracula`: line and muted from Comment, accent from Purple.
- **The status colours** (added, modified, removed). beautiful-mermaid's themes have none. Each built-in theme uses its own palette's green, orange or yellow, and red. A custom theme gets Primer's emphasis colours by background lightness unless its file has a `status` block.
- **A 10% tint** for status fills (the `keyBadge` mix), a 1px status stroke, and a 2px red stroke for risk.

## What Mermaid cannot copy

- Accent-coloured arrowheads. Mermaid colours an arrowhead like its edge, so `accent` is accepted for compatibility and has no effect.
- The header band on each group.
- ELK's edge routing. `curve: step` is the closest.
- Web fonts. GitHub blocks them, so Inter shows only if the viewer has it installed.

## Layout

On GitHub's Mermaid 11.17.2 the Output box drops below the Functions box unless the deepest function has invisible `~~~` links to each output. `internal/diagram` adds them and the gate checks them.
