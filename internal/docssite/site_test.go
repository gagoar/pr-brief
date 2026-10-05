// Package docssite tests the static website in docs/: it must agree with the code,
// and every link on it must resolve.
package docssite

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/convention"
)

const docs = "../../docs"

func pages(t *testing.T) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(docs, "*.html"))
	if err != nil || len(m) == 0 {
		t.Fatalf("no pages in docs/: %v", err)
	}
	return m
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func version(t *testing.T) string {
	t.Helper()
	var m struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(read(t, "../../.claude-plugin/plugin.json")), &m); err != nil {
		t.Fatal(err)
	}
	return m.Version
}

// The footer of every page names the version. iceberg's site shows a stale one.
func TestFooterVersionMatchesThePlugin(t *testing.T) {
	want := "pr-brief v" + version(t)
	for _, p := range pages(t) {
		if !strings.Contains(read(t, p), want) {
			t.Errorf("%s: footer must say %q; edit the footer when you bump the version", filepath.Base(p), want)
		}
	}
}

// The convention page states the limits. It must say the numbers the code enforces.
func TestConventionPageStatesTheRealLimits(t *testing.T) {
	page := read(t, filepath.Join(docs, "convention.html"))
	want := []string{
		fmt.Sprintf("%d nodes", convention.MaxNodes),
		fmt.Sprintf("%d diagrams", convention.MaxDiagrams),
		fmt.Sprintf("%d edges", convention.MaxEdges),
		fmt.Sprintf("%d characters", convention.MaxLabel),
		fmt.Sprintf("%d context nodes", convention.MaxContext),
		fmt.Sprintf("%d to %d rows", convention.MinReadRows, convention.MaxReadRows),
		fmt.Sprintf("%d code files", convention.SmallPRFiles),
		fmt.Sprintf("%d sentences", convention.MaxBriefSentences),
		"65,536 characters",
	}
	if convention.MaxBodyChars != 65536 {
		t.Fatal("the page says 65,536; update it with the constant")
	}
	for _, w := range want {
		if !strings.Contains(page, w) {
			t.Errorf("convention.html must say %q (it is enforced by internal/convention)", w)
		}
	}
	for _, s := range convention.Sections {
		if !strings.Contains(page, strings.TrimPrefix(s, "## ")) {
			t.Errorf("convention.html must name the part %q", s)
		}
	}
}

var (
	attrRe = regexp.MustCompile(`(?:href|src|srcset)="([^"]+)"`)
	idRe   = regexp.MustCompile(`id="([^"]+)"`)
)

func ids(t *testing.T, p string) map[string]bool {
	out := map[string]bool{}
	for _, m := range idRe.FindAllStringSubmatch(read(t, p), -1) {
		out[m[1]] = true
	}
	return out
}

// Every local link and image resolves, and every anchor exists on its target page.
func TestLocalLinksAndAnchorsResolve(t *testing.T) {
	for _, p := range pages(t) {
		for _, m := range attrRe.FindAllStringSubmatch(read(t, p), -1) {
			ref := m[1]
			if strings.HasPrefix(ref, "http") || strings.HasPrefix(ref, "mailto:") || strings.HasPrefix(ref, "data:") {
				continue
			}
			file, frag, _ := strings.Cut(ref, "#")
			target := p
			if file != "" {
				target = filepath.Join(docs, file)
				if st, err := os.Stat(target); err != nil || st.IsDir() {
					t.Errorf("%s links to %q, which does not exist in docs/", filepath.Base(p), ref)
					continue
				}
			}
			if frag != "" && strings.HasSuffix(target, ".html") && !ids(t, target)[frag] {
				t.Errorf("%s links to %q, but %s has no id %q", filepath.Base(p), ref, filepath.Base(target), frag)
			}
		}
	}
}

func TestEveryPageHasItsMetadataAndLoadsNothingExternal(t *testing.T) {
	for _, p := range pages(t) {
		page, name := read(t, p), filepath.Base(p)
		for _, want := range []string{"<title>", `name="description"`, `rel="canonical"`, `property="og:image"`, `name="twitter:card"`, `lang="en"`, "<h1>"} {
			if !strings.Contains(page, want) {
				t.Errorf("%s lacks %s", name, want)
			}
		}
		if regexp.MustCompile(`<script[^>]+src=`).MatchString(page) || regexp.MustCompile(`<link[^>]+rel="stylesheet"`).MatchString(page) {
			t.Errorf("%s loads an external script or stylesheet; the site is self-contained", name)
		}
		if strings.Contains(page, "fonts.googleapis") {
			t.Errorf("%s loads web fonts; use the system stacks of the design system", name)
		}
	}
}

func TestSitemapAndLlmsTxtListEveryPage(t *testing.T) {
	sitemap, llms := read(t, filepath.Join(docs, "sitemap.xml")), read(t, filepath.Join(docs, "llms.txt"))
	for _, p := range pages(t) {
		name := filepath.Base(p)
		if name == "404.html" {
			continue
		}
		loc := "https://gagoar.github.io/pr-brief/" + name
		if name == "index.html" {
			loc = "https://gagoar.github.io/pr-brief/"
		}
		if !strings.Contains(sitemap, "<loc>"+loc+"</loc>") {
			t.Errorf("sitemap.xml lacks %s", loc)
		}
		if !strings.Contains(llms, loc) {
			t.Errorf("llms.txt lacks %s", loc)
		}
	}
	if _, err := os.Stat(filepath.Join(docs, ".nojekyll")); err != nil {
		t.Error("docs/.nojekyll is missing; Pages would run Jekyll over the site")
	}
}

func TestImagesAreSmallEnoughForAGitRepo(t *testing.T) {
	const limit = 400 * 1024
	err := filepath.Walk(docs, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".png") && info.Size() > limit {
			t.Errorf("%s is %d KB; keep images under %d KB", p, info.Size()/1024, limit/1024)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
