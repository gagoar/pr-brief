package theme

// DefaultName is the theme used when none is configured.
const DefaultName = "github-dark"

// The colours of github-light, github-dark and dracula are copied from
// lukilabs/beautiful-mermaid src/theme.ts (MIT, Copyright (c) 2026 Craft Docs).
// alucard is NOT in that file. Its colours are Dracula's official light theme
// (draculatheme.com/spec, Alucard Classic), mapped the way Craft maps dracula:
// line and muted from Comment, accent from Purple.
//
// The status colours are not in beautiful-mermaid either. They come from each
// palette's own green, orange or yellow, and red.
var builtin = map[string]Theme{
	"github-dark": {
		Colors: Colors{BG: "#0d1117", FG: "#e6edf3", Line: "#3d444d", Accent: "#4493f8", Muted: "#9198a1"},
		Status: &Status{Added: "#3fb950", Modified: "#d29922", Removed: "#f85149"},
	},
	"github-light": {
		Colors: Colors{BG: "#ffffff", FG: "#1f2328", Line: "#d1d9e0", Accent: "#0969da", Muted: "#59636e"},
		Status: &Status{Added: "#1a7f37", Modified: "#9a6700", Removed: "#cf222e"},
	},
	"dracula": {
		Colors: Colors{BG: "#282a36", FG: "#f8f8f2", Line: "#6272a4", Accent: "#bd93f9", Muted: "#6272a4"},
		Status: &Status{Added: "#50fa7b", Modified: "#ffb86c", Removed: "#ff5555"},
	},
	"alucard": {
		Colors: Colors{BG: "#fffbeb", FG: "#1f1f1f", Line: "#6c664b", Accent: "#644ac9", Muted: "#6c664b"},
		Status: &Status{Added: "#14710a", Modified: "#a34d14", Removed: "#cb3a2a"},
	},
}

// Default status colours for a custom theme with no status block: Primer's
// emphasis colours, chosen by whether the background is dark or light.
var (
	statusOnDark  = Status{Added: "#3fb950", Modified: "#d29922", Removed: "#f85149"}
	statusOnLight = Status{Added: "#1a7f37", Modified: "#9a6700", Removed: "#cf222e"}
)
