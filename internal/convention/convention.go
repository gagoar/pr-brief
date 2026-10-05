// Package convention holds every fixed rule of pr-brief. Nothing here is
// configurable: a change to a rule is a plugin release.
package convention

const (
	// Diagram limits (Input -> Functions -> Output).
	MaxNodes    = 9
	MaxDiagrams = 3
	MaxEdges    = 14
	MaxLabel    = 28
	MaxContext  = 2

	// Review guide limits.
	MinReadRows = 1
	MaxReadRows = 7

	// A PR with this many code files or fewer gets no diagram.
	SmallPRFiles = 3

	// Brief: one paragraph, at most this many sentences.
	MaxBriefSentences = 5

	// GitHub rejects PR bodies longer than this (HTTP 422).
	MaxBodyChars = 65536
)

// Markers that delimit the generated description.
const (
	BeginPrefix  = "<!-- pr-brief:begin"
	End          = "<!-- pr-brief:end -->"
	PreviousOpen = "<!-- pr-brief:previous"
	PreviousEnd  = "pr-brief:previous:end -->"
	SkipPrefix   = "<!-- pr-brief:skip:"
	NoDiagram    = "<!-- pr-brief:no-diagram:"
)

// Section headings, in required order.
var Sections = []string{"## Brief", "## Change map", "## Review guide"}

// Prose styles.
const (
	StyleSTEIceberg = "ste+iceberg"
	StyleSTE        = "ste"
	StyleIceberg    = "iceberg"
	DefaultStyle    = StyleSTEIceberg
)

var Styles = []string{StyleSTEIceberg, StyleSTE, StyleIceberg}

// Previous-description handling for improve.
const (
	PreviousDrop    = "drop"
	PreviousComment = "comment"
	DefaultPrevious = PreviousDrop
)

var PreviousModes = []string{PreviousDrop, PreviousComment}

// IcebergFlags is the fixed flag set passed to iceberg:edit.
var IcebergFlags = []string{"--no-em-dash", "--no-weakeners", "--strip-ai-commentary"}

// ConfigVersion is the only accepted "version" in config files.
const ConfigVersion = 1
