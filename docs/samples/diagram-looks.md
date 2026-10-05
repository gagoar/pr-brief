# Sample: diagram looks (do not merge)

Eight looks for the pr-brief Change map, drawn from the **same** diagram. It has all six node states (added, modified, removed, context, risk, risk + added), all three edge kinds (new `==>`, existing `-->`, removed `-.->`), and the three Input / Functions / Output boxes.

**How to compare.** Open this page in GitHub's **Light**, **Dark** and **Dark dimmed** appearance (Settings → Appearance). For each look, check:

- Can you read every label?
- Can you see the thick red border on the two risk nodes, and the dashed border on the removed node?
- Do the three boxes and the edges look calm, not noisy?
- Is the look the same in the other modes?

The block below prints the Mermaid version GitHub is running:

```mermaid
info
```

## A. Today (baseline)

The current look: Mermaid's default theme, the six hand-written `classDef` lines, and Mermaid's default pastel colours on the three subgraph boxes. No init line.

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

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## B. Primer Solid, init + classDef

GitHub-native saturated fills with white text. One `%%{init}%%` line (base theme, grey edges, spacing) plus `classDef`, neutral subgraph boxes and grey edges.

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"basis","nodeSpacing":40,"rankSpacing":60},"themeVariables":{"darkMode":false,"primaryColor":"#59636e","primaryTextColor":"#ffffff","primaryBorderColor":"#afb8c1","nodeTextColor":"#ffffff","lineColor":"#6e7781","clusterBorder":"#8c959f","titleColor":"#6e7781","edgeLabelBackground":"#59636e"}}}%%
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
  classDef added fill:#1a7f37,stroke:#4ac26b,stroke-width:2px,color:#ffffff
  classDef modified fill:#9a6700,stroke:#d4a72c,stroke-width:2px,color:#ffffff
  classDef removed fill:#cf222e,stroke:#ff8182,stroke-width:2px,stroke-dasharray:6 4,color:#ffffff
  classDef context fill:#59636e,stroke:#afb8c1,stroke-width:1px,color:#ffffff
  classDef risk fill:#59636e,stroke:#da3633,stroke-width:4px,color:#ffffff
  classDef riskadded fill:#1a7f37,stroke:#da3633,stroke-width:4px,color:#ffffff
  classDef zone fill:#8c959f14,stroke:#8c959f,stroke-width:1px,color:#6e7781
  class IN,FN,OUT zone
  linkStyle default stroke:#6e7781,stroke-width:1.6px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## C. Primer Solid, classDef only

The same node colours with no init line. Only `classDef`, a neutral `zone` class for the three boxes, and `linkStyle default` for edges. The most portable version.

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
  classDef added fill:#1a7f37,stroke:#4ac26b,stroke-width:2px,color:#ffffff
  classDef modified fill:#9a6700,stroke:#d4a72c,stroke-width:2px,color:#ffffff
  classDef removed fill:#cf222e,stroke:#ff8182,stroke-width:2px,stroke-dasharray:6 4,color:#ffffff
  classDef context fill:#59636e,stroke:#afb8c1,stroke-width:1px,color:#ffffff
  classDef risk fill:#59636e,stroke:#da3633,stroke-width:4px,color:#ffffff
  classDef riskadded fill:#1a7f37,stroke:#da3633,stroke-width:4px,color:#ffffff
  classDef zone fill:#8c959f14,stroke:#8c959f,stroke-width:1px,color:#6e7781
  class IN,FN,OUT zone
  linkStyle default stroke:#6e7781,stroke-width:1.6px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## D. Catppuccin pastel, init + classDef

Soft pastels with dark labels. The closest to the Craft / Beautiful Mermaid feel.

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"basis","nodeSpacing":40,"rankSpacing":60},"themeVariables":{"darkMode":false,"primaryColor":"#bac2de","primaryTextColor":"#1e1e2e","primaryBorderColor":"#6c7086","nodeTextColor":"#1e1e2e","lineColor":"#7f849c","clusterBorder":"#7f849c","titleColor":"#7f849c","edgeLabelBackground":"#bac2de"}}}%%
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
  classDef added fill:#a6e3a1,stroke:#40a02b,stroke-width:2px,color:#1e1e2e
  classDef modified fill:#f9e2af,stroke:#df8e1d,stroke-width:2px,color:#1e1e2e
  classDef removed fill:#f38ba8,stroke:#d20f39,stroke-width:2px,stroke-dasharray:6 4,color:#1e1e2e
  classDef context fill:#bac2de,stroke:#6c7086,stroke-width:1px,color:#1e1e2e
  classDef risk fill:#bac2de,stroke:#d20f39,stroke-width:4px,color:#1e1e2e
  classDef riskadded fill:#a6e3a1,stroke:#d20f39,stroke-width:4px,color:#1e1e2e
  classDef zone fill:#8c959f14,stroke:#7f849c,stroke-width:1px,color:#7f849c
  class IN,FN,OUT zone
  linkStyle default stroke:#7f849c,stroke-width:1.6px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## E. Tokyo Night, init + classDef

Neon pastels with deep-ink labels.

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"basis","nodeSpacing":40,"rankSpacing":60},"themeVariables":{"darkMode":false,"primaryColor":"#a9b1d6","primaryTextColor":"#1a1b26","primaryBorderColor":"#565f89","nodeTextColor":"#1a1b26","lineColor":"#737aa2","clusterBorder":"#737aa2","titleColor":"#737aa2","edgeLabelBackground":"#a9b1d6"}}}%%
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
  classDef added fill:#9ece6a,stroke:#587539,stroke-width:2px,color:#1a1b26
  classDef modified fill:#e0af68,stroke:#8c6c3e,stroke-width:2px,color:#1a1b26
  classDef removed fill:#f7768e,stroke:#f52a65,stroke-width:2px,stroke-dasharray:6 4,color:#1a1b26
  classDef context fill:#a9b1d6,stroke:#565f89,stroke-width:1px,color:#1a1b26
  classDef risk fill:#a9b1d6,stroke:#f52a65,stroke-width:4px,color:#1a1b26
  classDef riskadded fill:#9ece6a,stroke:#f52a65,stroke-width:4px,color:#1a1b26
  classDef zone fill:#8c959f14,stroke:#737aa2,stroke-width:1px,color:#737aa2
  class IN,FN,OUT zone
  linkStyle default stroke:#737aa2,stroke-width:1.6px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## F. Okabe-Ito, init + classDef

Colour-blind-safe colours with black text. Removed nodes are pale dashed ghosts.

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"basis","nodeSpacing":40,"rankSpacing":60},"themeVariables":{"darkMode":false,"primaryColor":"#bbbbbb","primaryTextColor":"#000000","primaryBorderColor":"#777777","nodeTextColor":"#000000","lineColor":"#7c818c","clusterBorder":"#8c959f","titleColor":"#6e7781","edgeLabelBackground":"#bbbbbb"}}}%%
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
  classDef added fill:#009e73,stroke:#00704f,stroke-width:2px,color:#000000
  classDef modified fill:#e69f00,stroke:#a36f00,stroke-width:2px,color:#000000
  classDef removed fill:#f7dfcc,stroke:#d55e00,stroke-width:2px,stroke-dasharray:6 4,color:#8a3d00
  classDef context fill:#bbbbbb,stroke:#777777,stroke-width:1px,color:#000000
  classDef risk fill:#bbbbbb,stroke:#d55e00,stroke-width:4px,color:#000000
  classDef riskadded fill:#009e73,stroke:#d55e00,stroke-width:4px,color:#000000
  classDef zone fill:#8c959f14,stroke:#8c959f,stroke-width:1px,color:#6e7781
  class IN,FN,OUT zone
  linkStyle default stroke:#7c818c,stroke-width:1.6px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## G. Primer Solid, YAML frontmatter + neo look

A `---` config block with `look: neo`, curve and spacing instead of an init line. Tests what GitHub accepts. Likely to fail on Azure DevOps.

```mermaid
---
config:
  look: neo
  theme: base
  flowchart:
    curve: basis
    nodeSpacing: 40
    rankSpacing: 60
  themeVariables:
    primaryColor: "#59636e"
    primaryTextColor: "#ffffff"
    primaryBorderColor: "#afb8c1"
    nodeTextColor: "#ffffff"
    lineColor: "#6e7781"
    clusterBorder: "#8c959f"
    titleColor: "#6e7781"
    edgeLabelBackground: "#59636e"
---
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
  classDef added fill:#1a7f37,stroke:#4ac26b,stroke-width:2px,color:#ffffff
  classDef modified fill:#9a6700,stroke:#d4a72c,stroke-width:2px,color:#ffffff
  classDef removed fill:#cf222e,stroke:#ff8182,stroke-width:2px,stroke-dasharray:6 4,color:#ffffff
  classDef context fill:#59636e,stroke:#afb8c1,stroke-width:1px,color:#ffffff
  classDef risk fill:#59636e,stroke:#da3633,stroke-width:4px,color:#ffffff
  classDef riskadded fill:#1a7f37,stroke:#da3633,stroke-width:4px,color:#ffffff
  classDef zone fill:#8c959f14,stroke:#8c959f,stroke-width:1px,color:#6e7781
  class IN,FN,OUT zone
  linkStyle default stroke:#6e7781,stroke-width:1.6px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## H. Primer Solid, init + fonts

Look B plus `fontFamily` and `fontSize` in the init line. Tests whether GitHub honours fonts, and whether the font setting breaks the rest of the init line.

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"basis","nodeSpacing":40,"rankSpacing":60},"themeVariables":{"darkMode":false,"fontFamily":"-apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif","fontSize":"14px","primaryColor":"#59636e","primaryTextColor":"#ffffff","primaryBorderColor":"#afb8c1","nodeTextColor":"#ffffff","lineColor":"#6e7781","clusterBorder":"#8c959f","titleColor":"#6e7781","edgeLabelBackground":"#59636e"}}}%%
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
  classDef added fill:#1a7f37,stroke:#4ac26b,stroke-width:2px,color:#ffffff
  classDef modified fill:#9a6700,stroke:#d4a72c,stroke-width:2px,color:#ffffff
  classDef removed fill:#cf222e,stroke:#ff8182,stroke-width:2px,stroke-dasharray:6 4,color:#ffffff
  classDef context fill:#59636e,stroke:#afb8c1,stroke-width:1px,color:#ffffff
  classDef risk fill:#59636e,stroke:#da3633,stroke-width:4px,color:#ffffff
  classDef riskadded fill:#1a7f37,stroke:#da3633,stroke-width:4px,color:#ffffff
  classDef zone fill:#8c959f14,stroke:#8c959f,stroke-width:1px,color:#6e7781
  class IN,FN,OUT zone
  linkStyle default stroke:#6e7781,stroke-width:1.6px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

---
This page is a throwaway test branch. It is not part of the plugin.
