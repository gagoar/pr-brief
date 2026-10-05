# Sample: the diagram themes, drawn by the tool (do not merge)

Every diagram below is the **unedited output of `pr-brief diagram`** from v0.2.0. Each theme is drawn twice: the **all-states** test flow (every node state and edge kind) and a **realistic** flow (the hook gate from PR #1). Compare them in GitHub's **Light**, **Dark** and **Dark dimmed** appearance (Settings → Appearance).

The fifth theme is a custom file (`examples/theme-custom.json`), to show that your own colours work.

## 1. github-dark

the default.

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

## 2. github-light

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

## 3. dracula

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

Dracula's official light theme.

### All states

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#fffbeb","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#f8f4e5","primaryTextColor":"#1f1f1f","primaryBorderColor":"#d2cfc2","nodeTextColor":"#1f1f1f","textColor":"#1f1f1f","mainBkg":"#f8f4e5","nodeBorder":"#d2cfc2","lineColor":"#6c664b","clusterBkg":"#fffbeb","clusterBorder":"#d2cfc2","titleColor":"#6c664b","edgeLabelBackground":"#fffbeb"}}}%%
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
  classDef default fill:#f8f4e5,stroke:#d2cfc2,stroke-width:0.75px,color:#1f1f1f,font-weight:500
  classDef added fill:#e8edd4,stroke:#14710a,stroke-width:1px,color:#1f1f1f,font-weight:500
  classDef modified fill:#f6ead6,stroke:#a34d14,stroke-width:1px,color:#1f1f1f,font-weight:500
  classDef removed fill:#fae8d8,stroke:#cb3a2a,stroke-width:1px,stroke-dasharray:4 4,color:#1f1f1f,font-weight:500
  classDef context fill:#f8f4e5,stroke:#d2cfc2,stroke-width:0.75px,color:#6c664b,font-weight:500
  classDef risk fill:#f8f4e5,stroke:#cb3a2a,stroke-width:2px,color:#1f1f1f,font-weight:500
  classDef riskadded fill:#e8edd4,stroke:#cb3a2a,stroke-width:2px,color:#1f1f1f,font-weight:500
  classDef zone fill:#fffbeb,stroke:#d2cfc2,stroke-width:1px,color:#6c664b,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#6c664b,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#6c664b,stroke-width:2px
  linkStyle 5 stroke:#6c664b,stroke-width:1px,stroke-dasharray:4 4
```

### Realistic: the hook-gate flow

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":false,"background":"#fffbeb","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#f8f4e5","primaryTextColor":"#1f1f1f","primaryBorderColor":"#d2cfc2","nodeTextColor":"#1f1f1f","textColor":"#1f1f1f","mainBkg":"#f8f4e5","nodeBorder":"#d2cfc2","lineColor":"#6c664b","clusterBkg":"#fffbeb","clusterBorder":"#d2cfc2","titleColor":"#6c664b","edgeLabelBackground":"#fffbeb"}}}%%
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
  classDef default fill:#f8f4e5,stroke:#d2cfc2,stroke-width:0.75px,color:#1f1f1f,font-weight:500
  classDef added fill:#e8edd4,stroke:#14710a,stroke-width:1px,color:#1f1f1f,font-weight:500
  classDef modified fill:#f6ead6,stroke:#a34d14,stroke-width:1px,color:#1f1f1f,font-weight:500
  classDef removed fill:#fae8d8,stroke:#cb3a2a,stroke-width:1px,stroke-dasharray:4 4,color:#1f1f1f,font-weight:500
  classDef context fill:#f8f4e5,stroke:#d2cfc2,stroke-width:0.75px,color:#6c664b,font-weight:500
  classDef risk fill:#f8f4e5,stroke:#cb3a2a,stroke-width:2px,color:#1f1f1f,font-weight:500
  classDef riskadded fill:#e8edd4,stroke:#cb3a2a,stroke-width:2px,color:#1f1f1f,font-weight:500
  classDef zone fill:#fffbeb,stroke:#d2cfc2,stroke-width:1px,color:#6c664b,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 0,1,2,3,4,5,6 stroke:#6c664b,stroke-width:2px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

## 5. custom: examples/theme-custom.json

a custom theme file, in beautiful-mermaid's format.

### All states

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#10141c","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#161a23","primaryTextColor":"#e4ecf7","primaryBorderColor":"#3a3f48","nodeTextColor":"#e4ecf7","textColor":"#e4ecf7","mainBkg":"#161a23","nodeBorder":"#3a3f48","lineColor":"#4a5a78","clusterBkg":"#10141c","clusterBorder":"#3a3f48","titleColor":"#8b9bb4","edgeLabelBackground":"#10141c"}}}%%
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
  classDef default fill:#161a23,stroke:#3a3f48,stroke-width:0.75px,color:#e4ecf7,font-weight:500
  classDef added fill:#132624,stroke:#2ecc71,stroke-width:1px,color:#e4ecf7,font-weight:500
  classDef modified fill:#26261b,stroke:#f1c40f,stroke-width:1px,color:#e4ecf7,font-weight:500
  classDef removed fill:#261a1f,stroke:#e74c3c,stroke-width:1px,stroke-dasharray:4 4,color:#e4ecf7,font-weight:500
  classDef context fill:#161a23,stroke:#3a3f48,stroke-width:0.75px,color:#8b9bb4,font-weight:500
  classDef risk fill:#161a23,stroke:#e74c3c,stroke-width:2px,color:#e4ecf7,font-weight:500
  classDef riskadded fill:#132624,stroke:#e74c3c,stroke-width:2px,color:#e4ecf7,font-weight:500
  classDef zone fill:#10141c,stroke:#3a3f48,stroke-width:1px,color:#8b9bb4,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 1,2,3 stroke:#4a5a78,stroke-width:1px
  linkStyle 0,4,6,7 stroke:#4a5a78,stroke-width:2px
  linkStyle 5 stroke:#4a5a78,stroke-width:1px,stroke-dasharray:4 4
```

### Realistic: the hook-gate flow

```mermaid
%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":28,"rankSpacing":48,"diagramPadding":40},"themeVariables":{"darkMode":true,"background":"#10141c","fontFamily":"Inter, Helvetica, Arial","fontSize":"13px","dropShadow":"none","primaryColor":"#161a23","primaryTextColor":"#e4ecf7","primaryBorderColor":"#3a3f48","nodeTextColor":"#e4ecf7","textColor":"#e4ecf7","mainBkg":"#161a23","nodeBorder":"#3a3f48","lineColor":"#4a5a78","clusterBkg":"#10141c","clusterBorder":"#3a3f48","titleColor":"#8b9bb4","edgeLabelBackground":"#10141c"}}}%%
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
  classDef default fill:#161a23,stroke:#3a3f48,stroke-width:0.75px,color:#e4ecf7,font-weight:500
  classDef added fill:#132624,stroke:#2ecc71,stroke-width:1px,color:#e4ecf7,font-weight:500
  classDef modified fill:#26261b,stroke:#f1c40f,stroke-width:1px,color:#e4ecf7,font-weight:500
  classDef removed fill:#261a1f,stroke:#e74c3c,stroke-width:1px,stroke-dasharray:4 4,color:#e4ecf7,font-weight:500
  classDef context fill:#161a23,stroke:#3a3f48,stroke-width:0.75px,color:#8b9bb4,font-weight:500
  classDef risk fill:#161a23,stroke:#e74c3c,stroke-width:2px,color:#e4ecf7,font-weight:500
  classDef riskadded fill:#132624,stroke:#e74c3c,stroke-width:2px,color:#e4ecf7,font-weight:500
  classDef zone fill:#10141c,stroke:#3a3f48,stroke-width:1px,color:#8b9bb4,font-weight:600,font-size:12px
  class IN,FN,OUT zone
  linkStyle 0,1,2,3,4,5,6 stroke:#4a5a78,stroke-width:2px
```

- [ ] Light  - [ ] Dark  - [ ] Dark dimmed

---
Throwaway test branch. Theme colours: beautiful-mermaid (MIT, Copyright (c) 2026 Craft Docs) and the Dracula spec (MIT, Copyright (c) 2023 Dracula Theme). See THIRD_PARTY_NOTICES.md.
