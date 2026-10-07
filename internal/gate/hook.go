package gate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gagoar/pr-brief/internal/host"
	"regexp"
	"strings"
)

// HookInput is the PreToolUse payload Claude Code sends on stdin.
type HookInput struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
	Cwd       string         `json:"cwd"`
}

// HookOutput is the JSON a PreToolUse hook prints to deny a call.
type HookOutput struct {
	HookSpecificOutput struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
	} `json:"hookSpecificOutput"`
}

// DenyJSON renders a denial.
func DenyJSON(reason string) []byte {
	var o HookOutput
	o.HookSpecificOutput.HookEventName = "PreToolUse"
	o.HookSpecificOutput.PermissionDecision = "deny"
	o.HookSpecificOutput.PermissionDecisionReason = reason
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(o)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// Extraction is what the hook found out about a tool call.
type Extraction struct {
	Relevant bool   // the call writes a PR description
	Body     string // the description, when Known
	Known    bool   // the description could be read
	Problem  string // why it could not be read, when Relevant && !Known
	Host     string // the host the tool call targets, when the call says so
}

var (
	ghRe = regexp.MustCompile(`(?:^|[\s;&|(])gh\s+pr\s+(create|edit)\b`)
	azRe = regexp.MustCompile(`(?:^|[\s;&|(])az\s+repos\s+pr\s+(create|update)\b`)
)

// Extract inspects one tool call. Calls that do not write a PR description are
// not Relevant and always pass.
func Extract(in HookInput) []Extraction {
	name := in.ToolName
	switch {
	case name == "Bash":
		cmd, _ := in.ToolInput["command"].(string)
		return extractBash(cmd, newFileReader(in.Cwd))
	case strings.HasPrefix(name, "mcp__") && (strings.HasSuffix(name, "create_pull_request") || strings.HasSuffix(name, "update_pull_request")):
		ex := extractMCP(name, in.ToolInput)
		ex.Host = mcpHost(name)
		return []Extraction{ex}
	}
	return nil
}

func extractMCP(name string, input map[string]any) Extraction {
	creating := strings.HasSuffix(name, "create_pull_request")
	for _, key := range []string{"body", "description"} {
		if v, ok := input[key]; ok {
			if s, ok := v.(string); ok {
				if strings.TrimSpace(s) == "" && !creating {
					return Extraction{Relevant: true, Known: true, Body: ""}
				}
				return Extraction{Relevant: true, Known: true, Body: s}
			}
		}
	}
	if creating {
		return Extraction{Relevant: true, Problem: "the call has no body"}
	}
	return Extraction{} // an update that leaves the description alone
}

// mcpHost reads the host from an MCP tool name such as mcp__azure-devops__repo_create_pull_request.
func mcpHost(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "azure") || strings.Contains(n, "devops"):
		return host.AzureDevOps
	case strings.Contains(n, "github"):
		return host.GitHub
	}
	return ""
}

func extractBash(cmd string, read fileReader) []Extraction {
	var out []Extraction
	type spec struct {
		host     string
		re       *regexp.Regexp
		creating string
		bodyLong []string
		bodyShrt []string
		fileLong []string
		fileShrt []string
		fill     []string
	}
	specs := []spec{
		{host.GitHub, ghRe, "create", []string{"--body"}, []string{"-b"}, []string{"--body-file"}, []string{"-F"}, []string{"--fill", "--fill-first", "--fill-verbose", "--web", "-f", "-w"}},
		{host.AzureDevOps, azRe, "create", []string{"--description"}, []string{"-d"}, nil, nil, nil},
	}
	for _, sp := range specs {
		for _, loc := range sp.re.FindAllStringSubmatchIndex(cmd, -1) {
			// Position of "gh"/"az" itself, after the optional leading separator.
			start := loc[0]
			for start < loc[1] && (cmd[start] == ' ' || cmd[start] == '\t' || cmd[start] == '\n' || cmd[start] == ';' || cmd[start] == '&' || cmd[start] == '|' || cmd[start] == '(') {
				start++
			}
			if !atCommandPosition(cmd, start) {
				continue
			}
			verb := cmd[loc[2]:loc[3]]
			creating := verb == sp.creating
			words := parseWords(cmd[loc[1]:], read)
			ex := bodyFromWords(words, creating, sp.bodyLong, sp.bodyShrt, sp.fileLong, sp.fileShrt, sp.fill, read)
			ex.Host = sp.host
			out = append(out, ex)
		}
	}
	return out
}

func bodyFromWords(words []word, creating bool, bodyLong, bodyShort, fileLong, fileShort, fill []string, read fileReader) Extraction {
	var (
		haveBody, haveFile bool
		bodyW              word
		filePath           string
		fileLiteral        = true
		filled             bool
	)
	in := func(list []string, s string) bool {
		for _, v := range list {
			if v == s {
				return true
			}
		}
		return false
	}
	for i := 0; i < len(words); i++ {
		w := words[i].text
		next := func() (word, bool) {
			if i+1 < len(words) {
				i++
				return words[i], true
			}
			return word{}, false
		}
		switch {
		case in(bodyLong, w) || in(bodyShort, w):
			if v, ok := next(); ok {
				haveBody, bodyW = true, v
			}
		case in(fileLong, w) || in(fileShort, w):
			if v, ok := next(); ok {
				haveFile, filePath, fileLiteral = true, v.text, v.literal
			}
		case in(fill, w):
			filled = true
		default:
			for _, p := range bodyLong {
				if strings.HasPrefix(w, p+"=") {
					haveBody, bodyW = true, word{strings.TrimPrefix(w, p+"="), words[i].literal}
				}
			}
			for _, p := range fileLong {
				if strings.HasPrefix(w, p+"=") {
					haveFile, filePath, fileLiteral = true, strings.TrimPrefix(w, p+"="), words[i].literal
				}
			}
		}
	}

	switch {
	case haveFile:
		if filePath == "-" {
			return Extraction{Relevant: true, Problem: "the description comes from stdin, which the hook cannot read; write it to a file and pass --body-file <path>"}
		}
		if !fileLiteral {
			return Extraction{Relevant: true, Problem: fmt.Sprintf("the --body-file path %q contains a variable; use a plain path", filePath)}
		}
		data, err := read(filePath)
		if err != nil {
			return Extraction{Relevant: true, Problem: fmt.Sprintf("cannot read --body-file %s (%v); write the file first, in a separate step", filePath, err)}
		}
		return Extraction{Relevant: true, Known: true, Body: data}
	case haveBody:
		if !bodyW.literal {
			return Extraction{Relevant: true, Problem: "the description depends on a shell variable or command the hook cannot evaluate; write it to a file and pass --body-file <path>"}
		}
		return Extraction{Relevant: true, Known: true, Body: bodyW.text}
	case creating && filled:
		return Extraction{Relevant: true, Problem: "the description would come from commits or a browser form (--fill / --web); write it with /pr-brief and pass --body-file <path>"}
	case creating:
		return Extraction{Relevant: true, Problem: "no description given; run /pr-brief and pass --body-file <path>"}
	}
	return Extraction{} // an edit/update that leaves the description alone
}

// Evaluate runs the gate on a tool call. It returns a denial reason, or "" to allow.
// options is called with the call's working directory so the caller can resolve the repo config.
func Evaluate(in HookInput, options func(cwd string) Options) string {
	var reasons []string
	for _, ex := range Extract(in) {
		if !ex.Relevant {
			continue
		}
		if !ex.Known {
			reasons = append(reasons, "pr-brief: "+ex.Problem)
			continue
		}
		opts := options(in.Cwd)
		if ex.Host != "" {
			opts.Host = ex.Host
		}
		res := Check(ex.Body, opts)
		if res.OK() {
			continue
		}
		var b strings.Builder
		b.WriteString("pr-brief: this PR description does not meet the convention.\n")
		for i, f := range res.Findings {
			if i == 12 {
				fmt.Fprintf(&b, "  ... and %d more\n", len(res.Findings)-12)
				break
			}
			fmt.Fprintf(&b, "  - [%s] %s\n", f.Rule, f.Message)
		}
		b.WriteString("Run /pr-brief to write it. To skip this one PR, put `> pr-brief skipped: <reason>` in the description.")
		reasons = append(reasons, b.String())
	}
	return strings.Join(reasons, "\n")
}
