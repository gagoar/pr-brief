# Sample: pick a diagram look (do not merge)

Four candidates, each drawn twice: the **all-states** test diagram (every node state and edge kind), and a **realistic** one (the hook-gate flow from PR #1). Compare them in GitHub's **Light**, **Dark** and **Dark dimmed** appearance (Settings → Appearance), then tell me which to build in.

| Look | Where it comes from |
|---|---|
| github-light, github-dark | [`lukilabs/beautiful-mermaid`](https://github.com/lukilabs/beautiful-mermaid) `src/theme.ts` (MIT, Copyright (c) 2026 Craft Docs) |
| dracula | the same file |
| alucard | Dracula's official light theme ([spec](https://draculatheme.com/spec), Alucard Classic), mapped the way Craft maps `dracula`: line and muted = Comment, accent = Purple. Not in beautiful-mermaid. |

**How the style is built.** Craft's own mixing rule (node fill 3%, node stroke 20%, line 50%, secondary text 60%, group header 5%) and style constants (sharp corners, 0.75px node strokes, 1px group strokes, 2px thick edges, dotted `4 4`, 13px text at weight 500, 12px group headers at weight 600, orthogonal edges, node spacing 28, layer spacing 48, padding 40). Status colours (added, modified, removed) are not in Craft's themes. They come from each theme's own palette, drawn as a 10% tint with a 1px stroke. Not copyable in Mermaid: accent arrowheads, the group header band, ELK's exact routing.

**Layout fix.** On GitHub's Mermaid 11.17.2, Output drops below Functions unless the deepest function has invisible links to each output (`F5 ~~~ O1`). Edges are styled by index so those links stay invisible.

## 1. github-light

bg `#ffffff` · fg `#1f2328` · line `#d1d9e0` · node fill `#f8f8f9` · node stroke `#d2d3d4` · secondary text `#59636e`

### All states

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

### Realistic: the hook-gate flow

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#ffffff","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#f8f8f9","primaryTextColor":"#1f2328","primaryBorderColor":"#d2d3d4","nodeTextColor":"#1f2328","textColor":"#1f2328","mainBkg":"#f8f8f9","nodeBorder":"#d2d3d4","lineColor":"#d1d9e0","clusterBkg":"#ffffff","clusterBorder":"#d2d3d4","titleColor":"#59636e","edgeLabelBackground":"#ffffff"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["gateHook()"]:::added
    F2["styleFor()"]:::added
    F3["Evaluate()"]:::added
    F4["!extractBash()"]:::riskadded
    F5["Check()"]:::added
    F6["steHard()"]:::added
  end
  subgraph OUT["Output"]
    O1["O1"]
  end
  I1 ==> F1
  F1 ==> F2
  F1 ==> F3
  F3 ==> F4
  F3 ==> F5
  F5 ==> F6
  F1 ==> O1
  F6 ~~~ O1
  classDef default fill:#f8f8f9,stroke:#d2d3d4,stroke-width:0.75px,color:#1f2328,font-weight:500
  classDef added fill:#e8f2eb,stroke:#1a7f37,stroke-width:1px,color:#1f2328,font-weight:500
  classDef modified fill:#f5f0e6,stroke:#9a6700,stroke-width:1px,color:#1f2328,font-weight:500
  classDef removed fill:#fae9ea,stroke:#cf222e,stroke-width:1px,stroke-dasharray:4 4,color:#1f2328,font-weight:500
  classDef context fill:#f8f8f9,stroke:#d2d3d4,stroke-width:0.75px,color:#59636e,font-weight:500
  classDef risk fill:#f8f8f9,stroke:#cf222e,stroke-width:2px,color:#1f2328,font-weight:500
  classDef riskadded fill:#e8f2eb,stroke:#cf222e,stroke-width:2px,color:#1f2328,font-weight:500
  classDef zone fill:#ffffff,stroke:#d2d3d4,stroke-width:1px,color:#59636e,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 0,1,2,3,4,5,6 stroke:#d1d9e0,stroke-width:2px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 2. github-dark

bg `#0d1117` · fg `#e6edf3` · line `#3d444d` · node fill `#14181e` · node stroke `#383d43` · secondary text `#9198a1`

### All states

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

### Realistic: the hook-gate flow

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#0d1117","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#14181e","primaryTextColor":"#e6edf3","primaryBorderColor":"#383d43","nodeTextColor":"#e6edf3","textColor":"#e6edf3","mainBkg":"#14181e","nodeBorder":"#383d43","lineColor":"#3d444d","clusterBkg":"#0d1117","clusterBorder":"#383d43","titleColor":"#9198a1","edgeLabelBackground":"#0d1117"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["gateHook()"]:::added
    F2["styleFor()"]:::added
    F3["Evaluate()"]:::added
    F4["!extractBash()"]:::riskadded
    F5["Check()"]:::added
    F6["steHard()"]:::added
  end
  subgraph OUT["Output"]
    O1["O1"]
  end
  I1 ==> F1
  F1 ==> F2
  F1 ==> F3
  F3 ==> F4
  F3 ==> F5
  F5 ==> F6
  F1 ==> O1
  F6 ~~~ O1
  classDef default fill:#14181e,stroke:#383d43,stroke-width:0.75px,color:#e6edf3,font-weight:500
  classDef added fill:#12221d,stroke:#3fb950,stroke-width:1px,color:#e6edf3,font-weight:500
  classDef modified fill:#211f18,stroke:#d29922,stroke-width:1px,color:#e6edf3,font-weight:500
  classDef removed fill:#24171c,stroke:#f85149,stroke-width:1px,stroke-dasharray:4 4,color:#e6edf3,font-weight:500
  classDef context fill:#14181e,stroke:#383d43,stroke-width:0.75px,color:#9198a1,font-weight:500
  classDef risk fill:#14181e,stroke:#f85149,stroke-width:2px,color:#e6edf3,font-weight:500
  classDef riskadded fill:#12221d,stroke:#f85149,stroke-width:2px,color:#e6edf3,font-weight:500
  classDef zone fill:#0d1117,stroke:#383d43,stroke-width:1px,color:#9198a1,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 0,1,2,3,4,5,6 stroke:#3d444d,stroke-width:2px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 3. dracula

bg `#282a36` · fg `#f8f8f2` · line `#6272a4` · node fill `#2e303c` · node stroke `#52535c` · secondary text `#6272a4`

### All states

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

### Realistic: the hook-gate flow

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#282a36","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#2e303c","primaryTextColor":"#f8f8f2","primaryBorderColor":"#52535c","nodeTextColor":"#f8f8f2","textColor":"#f8f8f2","mainBkg":"#2e303c","nodeBorder":"#52535c","lineColor":"#6272a4","clusterBkg":"#282a36","clusterBorder":"#52535c","titleColor":"#6272a4","edgeLabelBackground":"#282a36"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["gateHook()"]:::added
    F2["styleFor()"]:::added
    F3["Evaluate()"]:::added
    F4["!extractBash()"]:::riskadded
    F5["Check()"]:::added
    F6["steHard()"]:::added
  end
  subgraph OUT["Output"]
    O1["O1"]
  end
  I1 ==> F1
  F1 ==> F2
  F1 ==> F3
  F3 ==> F4
  F3 ==> F5
  F5 ==> F6
  F1 ==> O1
  F6 ~~~ O1
  classDef default fill:#2e303c,stroke:#52535c,stroke-width:0.75px,color:#f8f8f2,font-weight:500
  classDef added fill:#2c3f3d,stroke:#50fa7b,stroke-width:1px,color:#f8f8f2,font-weight:500
  classDef modified fill:#3e383b,stroke:#ffb86c,stroke-width:1px,color:#f8f8f2,font-weight:500
  classDef removed fill:#3e2e39,stroke:#ff5555,stroke-width:1px,stroke-dasharray:4 4,color:#f8f8f2,font-weight:500
  classDef context fill:#2e303c,stroke:#52535c,stroke-width:0.75px,color:#6272a4,font-weight:500
  classDef risk fill:#2e303c,stroke:#ff5555,stroke-width:2px,color:#f8f8f2,font-weight:500
  classDef riskadded fill:#2c3f3d,stroke:#ff5555,stroke-width:2px,color:#f8f8f2,font-weight:500
  classDef zone fill:#282a36,stroke:#52535c,stroke-width:1px,color:#6272a4,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 0,1,2,3,4,5,6 stroke:#6272a4,stroke-width:2px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 4. alucard

bg `#FFFBEB` · fg `#1F1F1F` · line `#6C664B` · node fill `#f8f4e5` · node stroke `#d2cfc2` · secondary text `#6C664B`

### All states

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

### Realistic: the hook-gate flow

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#FFFBEB","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#f8f4e5","primaryTextColor":"#1F1F1F","primaryBorderColor":"#d2cfc2","nodeTextColor":"#1F1F1F","textColor":"#1F1F1F","mainBkg":"#f8f4e5","nodeBorder":"#d2cfc2","lineColor":"#6C664B","clusterBkg":"#FFFBEB","clusterBorder":"#d2cfc2","titleColor":"#6C664B","edgeLabelBackground":"#FFFBEB"}}}%%
graph LR
  subgraph IN["Input"]
    I1(["I1"])
  end
  subgraph FN["Functions"]
    F1["gateHook()"]:::added
    F2["styleFor()"]:::added
    F3["Evaluate()"]:::added
    F4["!extractBash()"]:::riskadded
    F5["Check()"]:::added
    F6["steHard()"]:::added
  end
  subgraph OUT["Output"]
    O1["O1"]
  end
  I1 ==> F1
  F1 ==> F2
  F1 ==> F3
  F3 ==> F4
  F3 ==> F5
  F5 ==> F6
  F1 ==> O1
  F6 ~~~ O1
  classDef default fill:#f8f4e5,stroke:#d2cfc2,stroke-width:0.75px,color:#1F1F1F,font-weight:500
  classDef added fill:#e8edd4,stroke:#14710A,stroke-width:1px,color:#1F1F1F,font-weight:500
  classDef modified fill:#f6ead6,stroke:#A34D14,stroke-width:1px,color:#1F1F1F,font-weight:500
  classDef removed fill:#fae8d8,stroke:#CB3A2A,stroke-width:1px,stroke-dasharray:4 4,color:#1F1F1F,font-weight:500
  classDef context fill:#f8f4e5,stroke:#d2cfc2,stroke-width:0.75px,color:#6C664B,font-weight:500
  classDef risk fill:#f8f4e5,stroke:#CB3A2A,stroke-width:2px,color:#1F1F1F,font-weight:500
  classDef riskadded fill:#e8edd4,stroke:#CB3A2A,stroke-width:2px,color:#1F1F1F,font-weight:500
  classDef zone fill:#FFFBEB,stroke:#d2cfc2,stroke-width:1px,color:#6C664B,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 0,1,2,3,4,5,6 stroke:#6C664B,stroke-width:2px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

---
Throwaway test branch. Theme values: MIT License, Copyright (c) 2026 Craft Docs, https://github.com/lukilabs/beautiful-mermaid/blob/main/LICENSE
