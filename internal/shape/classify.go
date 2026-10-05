package shape

import (
	"path"
	"regexp"
	"strings"
)

// Class is the role of a changed file.
type Class string

const (
	ClassCode      Class = "code"
	ClassTest      Class = "tests"
	ClassDoc       Class = "docs"
	ClassGenerated Class = "generated"
	ClassOther     Class = "other" // config, data, assets: counted, never drawn
)

var (
	testRe  = regexp.MustCompile(`(?i)(^|/)(tests?|__tests__|spec|specs|testdata|fixtures)/|_test\.go$|\.(test|spec)\.[jt]sx?$|tests?\.(cs|java)$|(^|/)test_[^/]*\.py$|_test\.py$`)
	docRe   = regexp.MustCompile(`(?i)\.(md|mdx|rst|txt|adoc)$|(^|/)docs?/`)
	genRe   = regexp.MustCompile(`(?i)\.generated\.|\.g\.cs$|\.designer\.cs$|\.pb\.go$|_pb2\.py$|(^|/)generated/|(^|/)(dist|bin|obj|node_modules|vendor)/|\.min\.(js|css)$|\.snap$|package-lock\.json$|yarn\.lock$|pnpm-lock\.yaml$|go\.sum$|\.lock$`)
	codeExt = map[string]bool{
		".go": true, ".cs": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true,
		".py": true, ".java": true, ".kt": true, ".tf": true, ".bicep": true, ".sql": true, ".proto": true,
	}
	manifestRe = regexp.MustCompile(`(?i)(^|/)(package\.json|go\.mod|requirements[^/]*\.txt|pyproject\.toml|pom\.xml|build\.gradle|Directory\.Packages\.props|[^/]*\.csproj)$`)
	workflowRe = regexp.MustCompile(`(^|/)(\.github/workflows|azure-pipelines|\.azuredevops|pipelines)/|azure-pipelines[^/]*\.ya?ml$`)
)

// Classify decides what a changed file is.
func Classify(p string) Class {
	switch {
	case genRe.MatchString(p):
		return ClassGenerated
	case testRe.MatchString(p):
		return ClassTest
	case docRe.MatchString(p):
		return ClassDoc
	}
	ext := strings.ToLower(path.Ext(p))
	if codeExt[ext] || (workflowRe.MatchString(p) && (ext == ".yml" || ext == ".yaml")) {
		return ClassCode
	}
	return ClassOther
}

// IsManifest reports package and project manifests.
func IsManifest(p string) bool { return manifestRe.MatchString(p) }

var (
	riskPathRe = regexp.MustCompile(`(?i)auth|secret|credential|password|iam\b|rbac|keyvault|key-vault|network|nsg|firewall|migration|\.proto$|openapi|swagger|(^|/)controllers?/|prod(uction)?[._/-]`)
	riskDiffRe = regexp.MustCompile(`(?i)\b(lock|mutex|semaphore|interlocked|synchronized|transaction|rollback|retry|retries|backoff|concurrent|deadlock|timeout|password|secret|crypto|encrypt|decrypt|permission|authorize)\b`)
)

// Risk is why a file or function deserves a careful read.
type Risk struct {
	Score   int      `json:"score"`
	Reasons []string `json:"reasons,omitempty"`
}

func (r *Risk) add(score int, reason string) {
	r.Score += score
	for _, x := range r.Reasons {
		if x == reason {
			return
		}
	}
	r.Reasons = append(r.Reasons, reason)
}

// pathRisk scores a path alone.
func pathRisk(p string) Risk {
	var r Risk
	if m := riskPathRe.FindString(p); m != "" {
		r.add(3, "path touches "+strings.Trim(strings.ToLower(m), "/._-"))
	}
	return r
}

// maxKeywordReasons caps the keyword evidence for one file or function, so a file
// that merely talks about locks cannot outrank one that takes them.
const maxKeywordReasons = 3

// diffRisk scores changed lines (added lines are what the PR introduces).
// Strings and comments are ignored: a word inside a regex or a message is not logic.
func diffRisk(added []string) Risk {
	var r Risk
	seen := map[string]bool{}
	for _, l := range added {
		for _, m := range riskDiffRe.FindAllString(stripLine(l, false), -1) {
			k := strings.ToLower(m)
			if !seen[k] && len(seen) < maxKeywordReasons {
				seen[k] = true
				r.add(1, "changes "+k+" logic")
			}
		}
	}
	return r
}

func sizeRisk(lines int) Risk {
	var r Risk
	if lines > 300 {
		r.add(2, "large change (>300 lines)")
	}
	return r
}

func merge(rs ...Risk) Risk {
	var out Risk
	for _, r := range rs {
		for _, reason := range r.Reasons {
			out.add(0, reason)
		}
		out.Score += r.Score
	}
	return out
}
