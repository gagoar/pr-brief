# Upstream

`internal/stelint` is a Go port of `scripts/ste-lint.py`.

- Repository: https://github.com/danyuchn/asd-ste100-skill
- Pinned commit: `32511c6992ecb5f1971e46a2943f2e6adceedafe`
- Files ported: `scripts/ste-lint.py` (486 lines). Test inputs copied verbatim:
  `examples/linter-edge-cases.md` (as `testdata/linter-edge-cases.md`) and
  `examples/before-after.md` (as `testdata/before-after.md`).
- Retrieved: 2026-10-05
- License: MIT. Copyright (c) 2026 Dustin Yuchen Teng. The full text is in
  `testdata/LICENSE-UPSTREAM` and reproduced below.

```
MIT License

Copyright (c) 2026 Dustin Yuchen Teng

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

(The upstream LICENSE file is the source of the text above; compare with
`testdata/LICENSE-UPSTREAM`.)

## Parity

Output is meant to be byte-identical to `ste-lint.py --json`. The optional test
`STE_PARITY=1 go test ./internal/stelint -run TestParity` checks that against
the pinned upstream script (see `parity/README.md`). At the pinned commit it
passed on every testdata file, 28 hand-written snippets, an options case, and
4000 random inputs. `testdata/golden/*.json` holds upstream's own output for
the testdata files, so `go test` checks the same thing without Python.

## Deliberate deviations

1. **API shape.** Upstream is a CLI script. The port exposes `Lint`,
   `LintNamed`, `LintProse`, `Merge` and `Report.JSON()`. There is no CLI,
   no plain-text report and no `--selftest` flag. The text report and the
   exit code logic are not ported (`Report.Failed()` gives the exit-1 test).
2. **Field names.** `Violation` keeps upstream's JSON keys as Go fields: `File`,
   `Line`, `Col`, `Rule`, `Level`, `Match`, `Message`. The hard class is the
   string `"advisory-free"`, the soft class `"advisory"`.
3. **Selftests.** The `--selftest` assertions became `TestSelftest` in
   `stelint_test.go`. No selftest case was dropped.
4. **Modal present-perfect lookbehinds removed.** Go's RE2 has no lookbehind.
   Upstream already re-checks `MODAL_PERFECT_PREFIX` on the text before each
   match, which rejects everything the lookbehinds rejected. The parity runs
   found no difference.
5. **Passive-voice lookahead emulated.** `(?!\s+(?:to|for|by)\s+\w+ing)` is applied
   after each match with `^\s+(?:to|for|by)\s+\w+ing` on the rest of the line.
   A rejected match resumes scanning at match start + 1, as Python does.
6. **Table split and sentence split** use manual loops instead of
   `(?<!\\)\|` and `(?<=[.!?])\s+`.
7. **Unicode.** Python's `\w`, `\b` and `\s` are Unicode-aware, Go's are ASCII.
   Each line is mapped to a same-length string before matching: non-ASCII word
   characters become `0`, non-ASCII or control whitespace that Python's `\s`
   accepts becomes a tab (so it still fails to match the literal spaces in
   patterns like `spin up`), and U+017F, U+212A, U+0131, U+0130 become `s`, `k`,
   `i`, `i` to follow Python's IGNORECASE folding. Matched text and columns are
   taken from the original line. Columns count code points, as in Python.
8. **Line breaks.** `str.splitlines()` is reproduced (`\n`, `\r\n`, `\r`, `\v`, `\f`,
   `\x1c`-`\x1e`, `\x85`, U+2028, U+2029). `str.strip()`/`split()` use
   Python's `isspace` set.
9. **JSON.** `Report.JSON()` is a hand-written encoder that matches Python's
   `json.dumps(indent=2)` (ASCII-only escapes with lowercase hex, surrogate
   pairs, `[]` for no findings). It omits the trailing newline `print()` adds.
   `per_100_words` is rounded with `strconv` on the exact binary value, which
   matches Python's `round(x, 1)`.
10. **Several files.** Upstream's CLI sums words over files and concatenates
    findings. `Merge` does the same; the Python sort is stable and
    per file, so merging keeps that order.
11. **Invalid UTF-8.** Python raises on undecodable input. Go replaces the bad
    bytes with U+FFFD and carries on. Behaviour on such input is not
    compared with upstream.
12. **`LintProse` (new, not in upstream).** Upstream already skips fenced code
    blocks and inline code spans, so `LintProse` leaves those alone. Upstream does
    not skip URLs (`http://`, `https://`, `ftp://`, `www.`) or HTML comments
    (`<!-- ... -->`, which may span lines). `LintProse` blanks them with spaces
    before linting, which keeps line and column numbers. Trailing sentence
    punctuation is not part of a URL. A `<!--` inside a backtick span or a
    fenced block does not open a comment. Words inside blanked regions are
    not counted.
