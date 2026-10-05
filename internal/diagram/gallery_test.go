package diagram

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gagoar/pr-brief/internal/theme"
)

func sum(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// The docs site shows one picture per built-in theme. This test fails when a theme has no
// picture, when the flow or the renderer changed since the pictures were drawn, or when a
// picture is missing. Fix it with `./build.sh gallery`.
func TestGalleryIsFreshAndComplete(t *testing.T) {
	dir := filepath.Join("..", "..", "docs", "gallery")
	flowData, err := os.ReadFile(filepath.Join(dir, "flow.json"))
	if err != nil {
		t.Fatal(err)
	}
	flow, err := Parse(flowData)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Flow   string            `json:"flow"`
		Themes map[string]string `json:"themes"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Flow != sum(flowData) {
		t.Error("docs/gallery/flow.json changed after the pictures were drawn; run ./build.sh gallery")
	}

	var want []string
	for _, name := range theme.Names() {
		want = append(want, name)
		th, _ := theme.Get(name)
		src, err := Render(flow, th)
		if err != nil {
			t.Fatal(err)
		}
		mmd, err := os.ReadFile(filepath.Join(dir, name+".mmd"))
		if err != nil {
			t.Errorf("theme %s has no gallery entry; run ./build.sh gallery", name)
			continue
		}
		if string(mmd) != src+"\n" {
			t.Errorf("docs/gallery/%s.mmd is not what the renderer prints now; run ./build.sh gallery", name)
		}
		if manifest.Themes[name] != sum(mmd) {
			t.Errorf("manifest.json has a stale hash for %s; run ./build.sh gallery", name)
		}
		for _, ext := range []string{".svg", ".png"} {
			if st, err := os.Stat(filepath.Join(dir, name+ext)); err != nil || st.Size() < 5000 {
				t.Errorf("docs/gallery/%s%s is missing or empty; run ./build.sh gallery", name, ext)
			}
		}
	}
	var have []string
	for name := range manifest.Themes {
		have = append(have, name)
	}
	sort.Strings(want)
	sort.Strings(have)
	if len(want) != len(have) {
		t.Errorf("manifest lists %v but the built-in themes are %v", have, want)
	}
}
