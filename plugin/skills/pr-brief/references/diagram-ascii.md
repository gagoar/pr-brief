# Plain-text diagrams

Read this only when the diagram format is `ascii` (the setting, or `auto` on Azure DevOps).

`diagram.format` picks Mermaid or plain text (`pr-brief diagram --format ascii`). `auto` draws plain text on Azure DevOps and Mermaid on GitHub and GitLab. A theme styles Mermaid only. Plain text has no colour.

**You draw it.** In plain text the tool does not draw. `pr-brief diagram --format ascii` prints two things:

- On **stdout**, a skeleton to paste: the hidden flow comment `<!-- pr-brief:flow ... -->`, a `text` fence with one placeholder line, and the Legend line. Keep the comment and the Legend as printed. Replace the placeholder line with your drawing.
- On **stderr**, the spec: the exact text of every box, and every call with its style. `--raw` prints the spec on stdout.

The gate reads the flow comment and checks that your drawing has one box for each node, with exactly that text, and no other box. It checks the width and the Legend. **It cannot check your lines**, so draw them with care: an arrow that points the wrong way passes the gate and misleads the reader.

### How to draw it

1. Put inputs in the top row and outputs in the bottom row. A call runs down the page. Put each function one row below the lowest function that calls it.
2. Draw a box for every node, with exactly the text in the spec between two `|`, one space of padding on each side:

   - A function: corners `+`, top and bottom `-`. Its text starts with the status marker: `+` added, `~` modified, `-` removed, `.` unchanged context (`| + InviteCommand.Handle |`).
   - A risky function (its text starts with `!`): top and bottom border of `=`.
   - A removed function: a dashed border (`+- - - -+`).
   - An input or an output: rounded corners (`.` and `'`), no marker (`| Input 1 |`).

3. Draw an arrow for every call, from the bottom of the caller down to the top of the callee, ending in a head just above the callee. Use `V` for a new call and `v` for an existing one. Draw a new call with `#` and `=`, an existing call with `|` and `-`, and a removed call with `:` and `.` and a head of `x`.
4. Leave the corners of a bend as `+`. When two lines cross, a `+` is enough. Give each call its own line. Do not merge two calls into one line.
5. Keep every line at 78 characters or fewer, with spaces and no tabs. When a row of boxes is too wide, move a box to a row below it.
6. Put nothing in a box except its text, and draw no box that the spec does not list. Do not add a title or a caption inside the fence.

Check your drawing against the spec before the gate does: count the boxes, then trace each call from its first box to its second.

```text
                   .---------.
                   | Input 1 |
                   '---------'
                        #
                        V
            +------------------------+
            | + InviteCommand.Handle |
            +------------------------+
              #               |   # #
              #               |   +=+==============+
              #               |     +==============+============+
              V               v                    V            V
+=======================+ +-----------------+ .----------. .----------.
| + !CodeGenerator.Next | | . UserQuery.Get | | Output 1 | | Output 2 |
+=======================+ +-----------------+ '----------' '----------'
```

**Legend:** [+] added · [.] unchanged context · ! risk (= border) · = # new call · - | existing call

In that drawing, `InviteCommand.Handle` calls `CodeGenerator.Next` (new, `#`) and `UserQuery.Get` (existing, `|`), and writes `Output 1` and `Output 2` (new). The two output calls bend right and cross each other at a `+`.

The begin marker adds `format=ascii`. The References table, the heading and the limits are the same as for Mermaid. Extra diagrams (`stateDiagram-v2`, `erDiagram`, `sequenceDiagram`) are Mermaid, so leave them out in plain text.
