# Sample: Craft / beautiful-mermaid themes (do not merge)

The styles here are taken from [`lukilabs/beautiful-mermaid`](https://github.com/lukilabs/beautiful-mermaid) (MIT, Copyright (c) 2026 Craft Docs), the library behind [agents.craft.do/mermaid](https://agents.craft.do/mermaid). Each look uses that project's **own theme colours** (`src/theme.ts`), its **own mixing rule** (node fill 3%, node stroke 20%, line 50%, secondary text 60%, group header 5%), and its **own style constants** (`src/styles.ts`, `src/renderer.ts`):

| Craft constant | Value | Mermaid setting used here |
|---|---|---|
| Corners | sharp (`rx=0`) | Mermaid's default rectangle |
| Node stroke | 0.75px | `classDef … stroke-width:0.75px` |
| Group box stroke | 1px, fill = background | `classDef zone`, `clusterBkg`, `clusterBorder` |
| Edge stroke | 1px, thick 2px, dotted `4 4` | `linkStyle` by edge index |
| Node text | 13px, weight 500 | `fontSize`, `font-weight:500` |
| Group header text | 12px, weight 600, secondary colour | `classDef zone … font-size:12px,font-weight:600` |
| Font | Inter | `fontFamily: Inter, Helvetica, Arial` (no hyphens: GitHub blanks a theme value that has one; the viewer needs Inter installed because GitHub blocks web fonts) |
| Arrowheads | accent colour (or fg 85%) | **not possible**: Mermaid colours arrowheads like the edge, so they use the line colour |
| Shadows | none | `dropShadow: none` |
| Edge routing | orthogonal | `curve: step` |
| Spacing | node 28, layer 48, padding 40 | `nodeSpacing`, `rankSpacing`, `diagramPadding` |

**What is not from Craft.** Their themes have no "added / modified / removed" colours, so those come from each theme's own editor palette (Tokyo Night, Catppuccin, Nord, and so on), drawn in Craft's style: a 10% tint fill (their `keyBadge` mix) with a 1px coloured stroke. **What Mermaid cannot copy:** the accent-coloured arrowheads, the header band on each group, and ELK's exact edge routing (`curve: step` is the closest).

Every look draws the **same** diagram, so only the style changes. Compare in GitHub's Light, Dark and Dark dimmed appearance (Settings → Appearance). Dark themes paint their own group boxes, but the page behind the diagram is GitHub's.

**Layout fix (found by testing on GitHub's own Mermaid 11.17.2).** Without help, GitHub drops the Output box *below* the Functions box and routes an edge behind an unrelated node. Two invisible links (`F5 ~~~ O1`, `F5 ~~~ O2`, from the deepest function to each output) keep Output to the right. Edges are styled by index, because `linkStyle default` would also paint the invisible links. The baseline (0) is left as it is today, to show the problem.

**Alucard** (look 16) is Dracula's official light theme. It is *not* in beautiful-mermaid. Its colours come from the [Dracula spec](https://draculatheme.com/spec) (Alucard Classic), mapped the way Craft maps `dracula`: line and muted = Comment, accent = Purple.

## 0. Today (baseline)

```mermaid
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  classDef added fill:#d4f7d4,stroke:#2e7d32,color:#1a1a1a
  classDef modified fill:#ffe9b3,stroke:#b26a00,color:#1a1a1a
  classDef removed fill:#fde0e0,stroke:#c62828,stroke-dasharray:4 3,color:#555
  classDef context fill:#f2f2f2,stroke:#bbb,color:#666
  classDef risk fill:#ffe9b3,stroke:#c62828,stroke-width:3px,color:#1a1a1a
  classDef riskadded fill:#d4f7d4,stroke:#c62828,stroke-width:3px,color:#1a1a1a
```

## 1. zinc-light

bg `#FFFFFF` · fg `#27272A` · line `#939394` · node fill `#f9f9f9` · node stroke `#d4d4d4` · secondary text `#7d7d7f`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#FFFFFF","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#f9f9f9","primaryTextColor":"#27272A","primaryBorderColor":"#d4d4d4","nodeTextColor":"#27272A","textColor":"#27272A","mainBkg":"#f9f9f9","nodeBorder":"#d4d4d4","lineColor":"#939394","clusterBkg":"#FFFFFF","clusterBorder":"#d4d4d4","titleColor":"#7d7d7f","edgeLabelBackground":"#FFFFFF"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#f9f9f9,stroke:#d4d4d4,stroke-width:0.75px,color:#27272A,font-weight:500
  classDef added fill:#e8f6ed,stroke:#16a34a,stroke-width:1px,color:#27272A,font-weight:500
  classDef modified fill:#fbf1e6,stroke:#d97706,stroke-width:1px,color:#27272A,font-weight:500
  classDef removed fill:#fce9e9,stroke:#dc2626,stroke-width:1px,stroke-dasharray:4 4,color:#27272A,font-weight:500
  classDef context fill:#f9f9f9,stroke:#d4d4d4,stroke-width:0.75px,color:#7d7d7f,font-weight:500
  classDef risk fill:#f9f9f9,stroke:#dc2626,stroke-width:2px,color:#27272A,font-weight:500
  classDef riskadded fill:#e8f6ed,stroke:#dc2626,stroke-width:2px,color:#27272A,font-weight:500
  classDef zone fill:#FFFFFF,stroke:#d4d4d4,stroke-width:1px,color:#7d7d7f,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#939394,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#939394,stroke-width:2px
  linkStyle 5 stroke:#939394,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 2. zinc-dark

bg `#18181B` · fg `#FAFAFA` · line `#89898a` · node fill `#1f1f22` · node stroke `#454548` · secondary text `#a0a0a1`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#18181B","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#1f1f22","primaryTextColor":"#FAFAFA","primaryBorderColor":"#454548","nodeTextColor":"#FAFAFA","textColor":"#FAFAFA","mainBkg":"#1f1f22","nodeBorder":"#454548","lineColor":"#89898a","clusterBkg":"#18181B","clusterBorder":"#454548","titleColor":"#a0a0a1","edgeLabelBackground":"#18181B"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#1f1f22,stroke:#454548,stroke-width:0.75px,color:#FAFAFA,font-weight:500
  classDef added fill:#1d2c25,stroke:#4ade80,stroke-width:1px,color:#FAFAFA,font-weight:500
  classDef modified fill:#2f291c,stroke:#fbbf24,stroke-width:1px,color:#FAFAFA,font-weight:500
  classDef removed fill:#2e2124,stroke:#f87171,stroke-width:1px,stroke-dasharray:4 4,color:#FAFAFA,font-weight:500
  classDef context fill:#1f1f22,stroke:#454548,stroke-width:0.75px,color:#a0a0a1,font-weight:500
  classDef risk fill:#1f1f22,stroke:#f87171,stroke-width:2px,color:#FAFAFA,font-weight:500
  classDef riskadded fill:#1d2c25,stroke:#f87171,stroke-width:2px,color:#FAFAFA,font-weight:500
  classDef zone fill:#18181B,stroke:#454548,stroke-width:1px,color:#a0a0a1,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#89898a,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#89898a,stroke-width:2px
  linkStyle 5 stroke:#89898a,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 3. tokyo-night

bg `#1a1b26` · fg `#a9b1d6` · line `#3d59a1` · node fill `#1e202b` · node stroke `#373949` · secondary text `#565f89`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#1a1b26","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#1e202b","primaryTextColor":"#a9b1d6","primaryBorderColor":"#373949","nodeTextColor":"#a9b1d6","textColor":"#a9b1d6","mainBkg":"#1e202b","nodeBorder":"#373949","lineColor":"#3d59a1","clusterBkg":"#1a1b26","clusterBorder":"#373949","titleColor":"#565f89","edgeLabelBackground":"#1a1b26"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#1e202b,stroke:#373949,stroke-width:0.75px,color:#a9b1d6,font-weight:500
  classDef added fill:#272d2d,stroke:#9ece6a,stroke-width:1px,color:#a9b1d6,font-weight:500
  classDef modified fill:#2e2a2d,stroke:#e0af68,stroke-width:1px,color:#a9b1d6,font-weight:500
  classDef removed fill:#302430,stroke:#f7768e,stroke-width:1px,stroke-dasharray:4 4,color:#a9b1d6,font-weight:500
  classDef context fill:#1e202b,stroke:#373949,stroke-width:0.75px,color:#565f89,font-weight:500
  classDef risk fill:#1e202b,stroke:#f7768e,stroke-width:2px,color:#a9b1d6,font-weight:500
  classDef riskadded fill:#272d2d,stroke:#f7768e,stroke-width:2px,color:#a9b1d6,font-weight:500
  classDef zone fill:#1a1b26,stroke:#373949,stroke-width:1px,color:#565f89,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#3d59a1,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#3d59a1,stroke-width:2px
  linkStyle 5 stroke:#3d59a1,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 4. tokyo-night-storm

bg `#24283b` · fg `#a9b1d6` · line `#3d59a1` · node fill `#282c40` · node stroke `#3f435a` · secondary text `#565f89`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#24283b","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#282c40","primaryTextColor":"#a9b1d6","primaryBorderColor":"#3f435a","nodeTextColor":"#a9b1d6","textColor":"#a9b1d6","mainBkg":"#282c40","nodeBorder":"#3f435a","lineColor":"#3d59a1","clusterBkg":"#24283b","clusterBorder":"#3f435a","titleColor":"#565f89","edgeLabelBackground":"#24283b"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#282c40,stroke:#3f435a,stroke-width:0.75px,color:#a9b1d6,font-weight:500
  classDef added fill:#303940,stroke:#9ece6a,stroke-width:1px,color:#a9b1d6,font-weight:500
  classDef modified fill:#373640,stroke:#e0af68,stroke-width:1px,color:#a9b1d6,font-weight:500
  classDef removed fill:#393043,stroke:#f7768e,stroke-width:1px,stroke-dasharray:4 4,color:#a9b1d6,font-weight:500
  classDef context fill:#282c40,stroke:#3f435a,stroke-width:0.75px,color:#565f89,font-weight:500
  classDef risk fill:#282c40,stroke:#f7768e,stroke-width:2px,color:#a9b1d6,font-weight:500
  classDef riskadded fill:#303940,stroke:#f7768e,stroke-width:2px,color:#a9b1d6,font-weight:500
  classDef zone fill:#24283b,stroke:#3f435a,stroke-width:1px,color:#565f89,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#3d59a1,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#3d59a1,stroke-width:2px
  linkStyle 5 stroke:#3d59a1,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 5. tokyo-night-light

bg `#d5d6db` · fg `#343b58` · line `#34548a` · node fill `#d0d1d7` · node stroke `#b5b7c1` · secondary text `#9699a3`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#d5d6db","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#d0d1d7","primaryTextColor":"#343b58","primaryBorderColor":"#b5b7c1","nodeTextColor":"#343b58","textColor":"#343b58","mainBkg":"#d0d1d7","nodeBorder":"#b5b7c1","lineColor":"#34548a","clusterBkg":"#d5d6db","clusterBorder":"#b5b7c1","titleColor":"#9699a3","edgeLabelBackground":"#d5d6db"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#d0d1d7,stroke:#b5b7c1,stroke-width:0.75px,color:#343b58,font-weight:500
  classDef added fill:#c8cccb,stroke:#587539,stroke-width:1px,color:#343b58,font-weight:500
  classDef modified fill:#cecbcb,stroke:#8c6c3e,stroke-width:1px,color:#343b58,font-weight:500
  classDef removed fill:#d8c5cf,stroke:#f52a65,stroke-width:1px,stroke-dasharray:4 4,color:#343b58,font-weight:500
  classDef context fill:#d0d1d7,stroke:#b5b7c1,stroke-width:0.75px,color:#9699a3,font-weight:500
  classDef risk fill:#d0d1d7,stroke:#f52a65,stroke-width:2px,color:#343b58,font-weight:500
  classDef riskadded fill:#c8cccb,stroke:#f52a65,stroke-width:2px,color:#343b58,font-weight:500
  classDef zone fill:#d5d6db,stroke:#b5b7c1,stroke-width:1px,color:#9699a3,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#34548a,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#34548a,stroke-width:2px
  linkStyle 5 stroke:#34548a,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 6. catppuccin-mocha

bg `#1e1e2e` · fg `#cdd6f4` · line `#585b70` · node fill `#232434` · node stroke `#414356` · secondary text `#6c7086`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#1e1e2e","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#232434","primaryTextColor":"#cdd6f4","primaryBorderColor":"#414356","nodeTextColor":"#cdd6f4","textColor":"#cdd6f4","mainBkg":"#232434","nodeBorder":"#414356","lineColor":"#585b70","clusterBkg":"#1e1e2e","clusterBorder":"#414356","titleColor":"#6c7086","edgeLabelBackground":"#1e1e2e"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#232434,stroke:#414356,stroke-width:0.75px,color:#cdd6f4,font-weight:500
  classDef added fill:#2c323a,stroke:#a6e3a1,stroke-width:1px,color:#cdd6f4,font-weight:500
  classDef modified fill:#34323b,stroke:#f9e2af,stroke-width:1px,color:#cdd6f4,font-weight:500
  classDef removed fill:#33293a,stroke:#f38ba8,stroke-width:1px,stroke-dasharray:4 4,color:#cdd6f4,font-weight:500
  classDef context fill:#232434,stroke:#414356,stroke-width:0.75px,color:#6c7086,font-weight:500
  classDef risk fill:#232434,stroke:#f38ba8,stroke-width:2px,color:#cdd6f4,font-weight:500
  classDef riskadded fill:#2c323a,stroke:#f38ba8,stroke-width:2px,color:#cdd6f4,font-weight:500
  classDef zone fill:#1e1e2e,stroke:#414356,stroke-width:1px,color:#6c7086,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#585b70,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#585b70,stroke-width:2px
  linkStyle 5 stroke:#585b70,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 7. catppuccin-latte

bg `#eff1f5` · fg `#4c4f69` · line `#9ca0b0` · node fill `#eaecf1` · node stroke `#ced1d9` · secondary text `#9ca0b0`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#eff1f5","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#eaecf1","primaryTextColor":"#4c4f69","primaryBorderColor":"#ced1d9","nodeTextColor":"#4c4f69","textColor":"#4c4f69","mainBkg":"#eaecf1","nodeBorder":"#ced1d9","lineColor":"#9ca0b0","clusterBkg":"#eff1f5","clusterBorder":"#ced1d9","titleColor":"#9ca0b0","edgeLabelBackground":"#eff1f5"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#eaecf1,stroke:#ced1d9,stroke-width:0.75px,color:#4c4f69,font-weight:500
  classDef added fill:#dee9e1,stroke:#40a02b,stroke-width:1px,color:#4c4f69,font-weight:500
  classDef modified fill:#ede7df,stroke:#df8e1d,stroke-width:1px,color:#4c4f69,font-weight:500
  classDef removed fill:#ecdae2,stroke:#d20f39,stroke-width:1px,stroke-dasharray:4 4,color:#4c4f69,font-weight:500
  classDef context fill:#eaecf1,stroke:#ced1d9,stroke-width:0.75px,color:#9ca0b0,font-weight:500
  classDef risk fill:#eaecf1,stroke:#d20f39,stroke-width:2px,color:#4c4f69,font-weight:500
  classDef riskadded fill:#dee9e1,stroke:#d20f39,stroke-width:2px,color:#4c4f69,font-weight:500
  classDef zone fill:#eff1f5,stroke:#ced1d9,stroke-width:1px,color:#9ca0b0,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#9ca0b0,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#9ca0b0,stroke-width:2px
  linkStyle 5 stroke:#9ca0b0,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 8. nord

bg `#2e3440` · fg `#d8dee9` · line `#4c566a` · node fill `#333945` · node stroke `#505662` · secondary text `#616e88`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#2e3440","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#333945","primaryTextColor":"#d8dee9","primaryBorderColor":"#505662","nodeTextColor":"#d8dee9","textColor":"#d8dee9","mainBkg":"#333945","nodeBorder":"#505662","lineColor":"#4c566a","clusterBkg":"#2e3440","clusterBorder":"#505662","titleColor":"#616e88","edgeLabelBackground":"#2e3440"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#333945,stroke:#505662,stroke-width:0.75px,color:#d8dee9,font-weight:500
  classDef added fill:#3a4248,stroke:#a3be8c,stroke-width:1px,color:#d8dee9,font-weight:500
  classDef modified fill:#414348,stroke:#ebcb8b,stroke-width:1px,color:#d8dee9,font-weight:500
  classDef removed fill:#3c3844,stroke:#bf616a,stroke-width:1px,stroke-dasharray:4 4,color:#d8dee9,font-weight:500
  classDef context fill:#333945,stroke:#505662,stroke-width:0.75px,color:#616e88,font-weight:500
  classDef risk fill:#333945,stroke:#bf616a,stroke-width:2px,color:#d8dee9,font-weight:500
  classDef riskadded fill:#3a4248,stroke:#bf616a,stroke-width:2px,color:#d8dee9,font-weight:500
  classDef zone fill:#2e3440,stroke:#505662,stroke-width:1px,color:#616e88,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#4c566a,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#4c566a,stroke-width:2px
  linkStyle 5 stroke:#4c566a,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 9. nord-light

bg `#eceff4` · fg `#2e3440` · line `#aab1c0` · node fill `#e6e9ef` · node stroke `#c6cad0` · secondary text `#7b88a1`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#eceff4","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#e6e9ef","primaryTextColor":"#2e3440","primaryBorderColor":"#c6cad0","nodeTextColor":"#2e3440","textColor":"#2e3440","mainBkg":"#e6e9ef","nodeBorder":"#c6cad0","lineColor":"#aab1c0","clusterBkg":"#eceff4","clusterBorder":"#c6cad0","titleColor":"#7b88a1","edgeLabelBackground":"#eceff4"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#e6e9ef,stroke:#c6cad0,stroke-width:0.75px,color:#2e3440,font-weight:500
  classDef added fill:#e5eaea,stroke:#a3be8c,stroke-width:1px,color:#2e3440,font-weight:500
  classDef modified fill:#ecebea,stroke:#ebcb8b,stroke-width:1px,color:#2e3440,font-weight:500
  classDef removed fill:#e8e1e6,stroke:#bf616a,stroke-width:1px,stroke-dasharray:4 4,color:#2e3440,font-weight:500
  classDef context fill:#e6e9ef,stroke:#c6cad0,stroke-width:0.75px,color:#7b88a1,font-weight:500
  classDef risk fill:#e6e9ef,stroke:#bf616a,stroke-width:2px,color:#2e3440,font-weight:500
  classDef riskadded fill:#e5eaea,stroke:#bf616a,stroke-width:2px,color:#2e3440,font-weight:500
  classDef zone fill:#eceff4,stroke:#c6cad0,stroke-width:1px,color:#7b88a1,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#aab1c0,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#aab1c0,stroke-width:2px
  linkStyle 5 stroke:#aab1c0,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 10. dracula

bg `#282a36` · fg `#f8f8f2` · line `#6272a4` · node fill `#2e303c` · node stroke `#52535c` · secondary text `#6272a4`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#282a36","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#2e303c","primaryTextColor":"#f8f8f2","primaryBorderColor":"#52535c","nodeTextColor":"#f8f8f2","textColor":"#f8f8f2","mainBkg":"#2e303c","nodeBorder":"#52535c","lineColor":"#6272a4","clusterBkg":"#282a36","clusterBorder":"#52535c","titleColor":"#6272a4","edgeLabelBackground":"#282a36"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#2e303c,stroke:#52535c,stroke-width:0.75px,color:#f8f8f2,font-weight:500
  classDef added fill:#2c3f3d,stroke:#50fa7b,stroke-width:1px,color:#f8f8f2,font-weight:500
  classDef modified fill:#3e383b,stroke:#ffb86c,stroke-width:1px,color:#f8f8f2,font-weight:500
  classDef removed fill:#3e2e39,stroke:#ff5555,stroke-width:1px,stroke-dasharray:4 4,color:#f8f8f2,font-weight:500
  classDef context fill:#2e303c,stroke:#52535c,stroke-width:0.75px,color:#6272a4,font-weight:500
  classDef risk fill:#2e303c,stroke:#ff5555,stroke-width:2px,color:#f8f8f2,font-weight:500
  classDef riskadded fill:#2c3f3d,stroke:#ff5555,stroke-width:2px,color:#f8f8f2,font-weight:500
  classDef zone fill:#282a36,stroke:#52535c,stroke-width:1px,color:#6272a4,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#6272a4,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#6272a4,stroke-width:2px
  linkStyle 5 stroke:#6272a4,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 11. github-light

bg `#ffffff` · fg `#1f2328` · line `#d1d9e0` · node fill `#f8f8f9` · node stroke `#d2d3d4` · secondary text `#59636e`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#ffffff","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#f8f8f9","primaryTextColor":"#1f2328","primaryBorderColor":"#d2d3d4","nodeTextColor":"#1f2328","textColor":"#1f2328","mainBkg":"#f8f8f9","nodeBorder":"#d2d3d4","lineColor":"#d1d9e0","clusterBkg":"#ffffff","clusterBorder":"#d2d3d4","titleColor":"#59636e","edgeLabelBackground":"#ffffff"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#f8f8f9,stroke:#d2d3d4,stroke-width:0.75px,color:#1f2328,font-weight:500
  classDef added fill:#e8f2eb,stroke:#1a7f37,stroke-width:1px,color:#1f2328,font-weight:500
  classDef modified fill:#f5f0e6,stroke:#9a6700,stroke-width:1px,color:#1f2328,font-weight:500
  classDef removed fill:#fae9ea,stroke:#cf222e,stroke-width:1px,stroke-dasharray:4 4,color:#1f2328,font-weight:500
  classDef context fill:#f8f8f9,stroke:#d2d3d4,stroke-width:0.75px,color:#59636e,font-weight:500
  classDef risk fill:#f8f8f9,stroke:#cf222e,stroke-width:2px,color:#1f2328,font-weight:500
  classDef riskadded fill:#e8f2eb,stroke:#cf222e,stroke-width:2px,color:#1f2328,font-weight:500
  classDef zone fill:#ffffff,stroke:#d2d3d4,stroke-width:1px,color:#59636e,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#d1d9e0,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#d1d9e0,stroke-width:2px
  linkStyle 5 stroke:#d1d9e0,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 12. github-dark

bg `#0d1117` · fg `#e6edf3` · line `#3d444d` · node fill `#14181e` · node stroke `#383d43` · secondary text `#9198a1`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#0d1117","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#14181e","primaryTextColor":"#e6edf3","primaryBorderColor":"#383d43","nodeTextColor":"#e6edf3","textColor":"#e6edf3","mainBkg":"#14181e","nodeBorder":"#383d43","lineColor":"#3d444d","clusterBkg":"#0d1117","clusterBorder":"#383d43","titleColor":"#9198a1","edgeLabelBackground":"#0d1117"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#14181e,stroke:#383d43,stroke-width:0.75px,color:#e6edf3,font-weight:500
  classDef added fill:#12221d,stroke:#3fb950,stroke-width:1px,color:#e6edf3,font-weight:500
  classDef modified fill:#211f18,stroke:#d29922,stroke-width:1px,color:#e6edf3,font-weight:500
  classDef removed fill:#24171c,stroke:#f85149,stroke-width:1px,stroke-dasharray:4 4,color:#e6edf3,font-weight:500
  classDef context fill:#14181e,stroke:#383d43,stroke-width:0.75px,color:#9198a1,font-weight:500
  classDef risk fill:#14181e,stroke:#f85149,stroke-width:2px,color:#e6edf3,font-weight:500
  classDef riskadded fill:#12221d,stroke:#f85149,stroke-width:2px,color:#e6edf3,font-weight:500
  classDef zone fill:#0d1117,stroke:#383d43,stroke-width:1px,color:#9198a1,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#3d444d,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#3d444d,stroke-width:2px
  linkStyle 5 stroke:#3d444d,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 13. solarized-light

bg `#fdf6e3` · fg `#657b83` · line `#93a1a1` · node fill `#f8f2e0` · node stroke `#dfddd0` · secondary text `#93a1a1`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#fdf6e3","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#f8f2e0","primaryTextColor":"#657b83","primaryBorderColor":"#dfddd0","nodeTextColor":"#657b83","textColor":"#657b83","mainBkg":"#f8f2e0","nodeBorder":"#dfddd0","lineColor":"#93a1a1","clusterBkg":"#fdf6e3","clusterBorder":"#dfddd0","titleColor":"#93a1a1","edgeLabelBackground":"#fdf6e3"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#f8f2e0,stroke:#dfddd0,stroke-width:0.75px,color:#657b83,font-weight:500
  classDef added fill:#f1edcc,stroke:#859900,stroke-width:1px,color:#657b83,font-weight:500
  classDef modified fill:#f6ebcc,stroke:#b58900,stroke-width:1px,color:#657b83,font-weight:500
  classDef removed fill:#fae2d1,stroke:#dc322f,stroke-width:1px,stroke-dasharray:4 4,color:#657b83,font-weight:500
  classDef context fill:#f8f2e0,stroke:#dfddd0,stroke-width:0.75px,color:#93a1a1,font-weight:500
  classDef risk fill:#f8f2e0,stroke:#dc322f,stroke-width:2px,color:#657b83,font-weight:500
  classDef riskadded fill:#f1edcc,stroke:#dc322f,stroke-width:2px,color:#657b83,font-weight:500
  classDef zone fill:#fdf6e3,stroke:#dfddd0,stroke-width:1px,color:#93a1a1,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#93a1a1,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#93a1a1,stroke-width:2px
  linkStyle 5 stroke:#93a1a1,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 14. solarized-dark

bg `#002b36` · fg `#839496` · line `#586e75` · node fill `#042e39` · node stroke `#1a4049` · secondary text `#586e75`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#002b36","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#042e39","primaryTextColor":"#839496","primaryBorderColor":"#1a4049","nodeTextColor":"#839496","textColor":"#839496","mainBkg":"#042e39","nodeBorder":"#1a4049","lineColor":"#586e75","clusterBkg":"#002b36","clusterBorder":"#1a4049","titleColor":"#586e75","edgeLabelBackground":"#002b36"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#042e39,stroke:#1a4049,stroke-width:0.75px,color:#839496,font-weight:500
  classDef added fill:#0d3631,stroke:#859900,stroke-width:1px,color:#839496,font-weight:500
  classDef modified fill:#123431,stroke:#b58900,stroke-width:1px,color:#839496,font-weight:500
  classDef removed fill:#162c35,stroke:#dc322f,stroke-width:1px,stroke-dasharray:4 4,color:#839496,font-weight:500
  classDef context fill:#042e39,stroke:#1a4049,stroke-width:0.75px,color:#586e75,font-weight:500
  classDef risk fill:#042e39,stroke:#dc322f,stroke-width:2px,color:#839496,font-weight:500
  classDef riskadded fill:#0d3631,stroke:#dc322f,stroke-width:2px,color:#839496,font-weight:500
  classDef zone fill:#002b36,stroke:#1a4049,stroke-width:1px,color:#586e75,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#586e75,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#586e75,stroke-width:2px
  linkStyle 5 stroke:#586e75,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 15. one-dark

bg `#282c34` · fg `#abb2bf` · line `#4b5263` · node fill `#2c3038` · node stroke `#424750` · secondary text `#5c6370`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#282c34","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#2c3038","primaryTextColor":"#abb2bf","primaryBorderColor":"#424750","nodeTextColor":"#abb2bf","textColor":"#abb2bf","mainBkg":"#2c3038","nodeBorder":"#424750","lineColor":"#4b5263","clusterBkg":"#282c34","clusterBorder":"#424750","titleColor":"#5c6370","edgeLabelBackground":"#282c34"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#2c3038,stroke:#424750,stroke-width:0.75px,color:#abb2bf,font-weight:500
  classDef added fill:#333b3b,stroke:#98c379,stroke-width:1px,color:#abb2bf,font-weight:500
  classDef modified fill:#3b3b3b,stroke:#e5c07b,stroke-width:1px,color:#abb2bf,font-weight:500
  classDef removed fill:#3a323a,stroke:#e06c75,stroke-width:1px,stroke-dasharray:4 4,color:#abb2bf,font-weight:500
  classDef context fill:#2c3038,stroke:#424750,stroke-width:0.75px,color:#5c6370,font-weight:500
  classDef risk fill:#2c3038,stroke:#e06c75,stroke-width:2px,color:#abb2bf,font-weight:500
  classDef riskadded fill:#333b3b,stroke:#e06c75,stroke-width:2px,color:#abb2bf,font-weight:500
  classDef zone fill:#282c34,stroke:#424750,stroke-width:1px,color:#5c6370,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#4b5263,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#4b5263,stroke-width:2px
  linkStyle 5 stroke:#4b5263,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 16. alucard

bg `#FFFBEB` · fg `#1F1F1F` · line `#6C664B` · node fill `#f8f4e5` · node stroke `#d2cfc2` · secondary text `#6C664B`

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#FFFBEB","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#f8f4e5","primaryTextColor":"#1F1F1F","primaryBorderColor":"#d2cfc2","nodeTextColor":"#1F1F1F","textColor":"#1F1F1F","mainBkg":"#f8f4e5","nodeBorder":"#d2cfc2","lineColor":"#6C664B","clusterBkg":"#FFFBEB","clusterBorder":"#d2cfc2","titleColor":"#6C664B","edgeLabelBackground":"#FFFBEB"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["handleInvite()"]:::added
    F2["checkQuota()"]:::modified
    F3["sendLegacyMail()"]:::removed
    F4["UserQuery.Get()"]:::context
    F5["!CodeGenerator.Next()"]:::risk
    F6["!saveInvite()"]:::riskadded
  end
  subgraph OUT["Output"]
    O1[("O1")]
    O2>"O2"]
  end
  I1 ==> F1
  F1 --> F2
  F1 --> F4
  F2 --> F5
  F1 ==> F6
  F1 -.-> F3
  F6 ==> O1
  F6 ==> O2
  F5 ~~~ O1
  F5 ~~~ O2
  classDef default fill:#f8f4e5,stroke:#d2cfc2,stroke-width:0.75px,color:#1F1F1F,font-weight:500
  classDef added fill:#e8edd4,stroke:#14710A,stroke-width:1px,color:#1F1F1F,font-weight:500
  classDef modified fill:#f6ead6,stroke:#A34D14,stroke-width:1px,color:#1F1F1F,font-weight:500
  classDef removed fill:#fae8d8,stroke:#CB3A2A,stroke-width:1px,stroke-dasharray:4 4,color:#1F1F1F,font-weight:500
  classDef context fill:#f8f4e5,stroke:#d2cfc2,stroke-width:0.75px,color:#6C664B,font-weight:500
  classDef risk fill:#f8f4e5,stroke:#CB3A2A,stroke-width:2px,color:#1F1F1F,font-weight:500
  classDef riskadded fill:#e8edd4,stroke:#CB3A2A,stroke-width:2px,color:#1F1F1F,font-weight:500
  classDef zone fill:#FFFBEB,stroke:#d2cfc2,stroke-width:1px,color:#6C664B,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#6C664B,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#6C664B,stroke-width:2px
  linkStyle 5 stroke:#6C664B,stroke-width:1px,stroke-dasharray:4 4
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

---
Throwaway test branch. Theme values: MIT License, Copyright (c) 2026 Craft Docs, https://github.com/lukilabs/beautiful-mermaid/blob/main/LICENSE
