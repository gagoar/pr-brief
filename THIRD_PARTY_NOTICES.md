# Third-party notices

pr-brief is MIT-licensed (see `LICENSE`). It contains work from these projects.

## beautiful-mermaid (diagram themes)

Theme colours, the colour-mixing rule and the style constants in `internal/theme`. See `internal/theme/UPSTREAM.md`.
https://github.com/lukilabs/beautiful-mermaid

```
MIT License

Copyright (c) 2026 Craft Docs

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

## Dracula theme (the Dracula and Alucard colours)

The `dracula` colours come through beautiful-mermaid. The `alucard` colours are Dracula's official light theme (Alucard Classic) from https://draculatheme.com/spec.
https://github.com/dracula/dracula-theme

```
The MIT License (MIT)

Copyright (c) 2023 Dracula Theme

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

## ste-lint.py (the prose lint)

`internal/stelint` is a Go port of `scripts/ste-lint.py` by Dustin Yuchen Teng, MIT. See `internal/stelint/UPSTREAM.md` and `internal/stelint/testdata/LICENSE-UPSTREAM`.
https://github.com/danyuchn/asd-ste100-skill

The `asd-ste100` skill that the `ste` and `ste+iceberg` styles use is the same work. It is packaged as the plugin
[unmistakable](https://github.com/gagoar/unmistakable) (MIT, with the upstream notice kept), which pr-brief declares as a dependency.

## Icon

`docs/assets/icon.svg` uses the "code review" glyph from The Noun Project (icon 5458418), supplied by the maintainer. Check the licence you hold for that icon: the free licence (CC BY 3.0) requires credit to its author on the Noun Project page.

## Space Grotesk and DM Mono (site fonts)

The web fonts in `docs/assets/fonts/` (Space Grotesk 600 and 700, DM Mono 400) come from the `@fontsource` packages. Both are licensed under the SIL Open Font License 1.1 (https://openfontlicense.org). Space Grotesk: Copyright 2020 The Space Grotesk Project Authors (https://github.com/floriankarsten/space-grotesk). DM Mono: Copyright 2020 The DM Mono Project Authors (https://github.com/googlefonts/dm-mono).
