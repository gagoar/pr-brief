// Package schemacheck keeps the published JSON Schemas in step with the Go types.
package schemacheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/theme"
)

func schema(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "schema", name))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("%s is not valid JSON: %v", name, err)
	}
	return m
}

func props(m map[string]any) []string {
	var out []string
	if p, ok := m["properties"].(map[string]any); ok {
		for k := range p {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func tags(v any) []string {
	var out []string
	var walk func(reflect.Type)
	walk = func(rt reflect.Type) {
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")[0]
			if f.Anonymous && tag == "" {
				walk(f.Type)
				continue
			}
			if tag != "" && tag != "-" {
				out = append(out, tag)
			}
		}
	}
	walk(reflect.TypeOf(v))
	sort.Strings(out)
	return out
}

func same(t *testing.T, what string, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("%s: schema has %v, Go type has %v", what, got, want)
	}
}

func sub(m map[string]any, path ...string) map[string]any {
	for _, p := range path {
		switch v := m[p].(type) {
		case map[string]any:
			m = v
		default:
			return nil
		}
	}
	return m
}

func TestFlowSchemaMatchesTheGoTypes(t *testing.T) {
	s := schema(t, "pr-brief-flow.schema.json")
	same(t, "flow", props(s), tags(diagram.Flow{}))
	same(t, "flow.inputs", props(sub(s, "properties", "inputs", "items")), tags(diagram.Port{}))
	same(t, "flow.outputs", props(sub(s, "properties", "outputs", "items")), tags(diagram.Port{}))
	same(t, "flow.nodes", props(sub(s, "properties", "nodes", "items")), tags(diagram.Node{}))
	same(t, "flow.edges", props(sub(s, "properties", "edges", "items")), tags(diagram.Edge{}))
}

func TestThemeSchemaMatchesTheGoTypes(t *testing.T) {
	s := schema(t, "pr-brief-theme.schema.json")
	same(t, "theme", props(s), tags(theme.Theme{}))
	same(t, "theme.status", props(sub(s, "properties", "status")), tags(theme.Status{}))
	if s["additionalProperties"] != false {
		t.Error("the theme schema must reject unknown keys, as the Go parser does")
	}
}

func TestSettingsSchemaMatchesTheConfigType(t *testing.T) {
	s := schema(t, "pr-brief.schema.json")
	same(t, "config", props(s), tags(config.File{}))
	same(t, "config.diagram", props(sub(s, "properties", "diagram")), tags(config.Diagram{}))
	same(t, "config.improve", props(sub(s, "properties", "improve")), tags(config.Improve{}))
	anyOf := sub(s, "properties", "diagram", "properties", "theme")["anyOf"].([]any)
	var names []string
	for _, n := range anyOf[0].(map[string]any)["enum"].([]any) {
		names = append(names, n.(string))
	}
	sort.Strings(names)
	want := append([]string{}, theme.Names()...)
	sort.Strings(want)
	same(t, "built-in theme names", names, want)
}

func TestTheExampleThemeIsValid(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "theme-custom.json"))
	if err != nil {
		t.Fatal(err)
	}
	th, err := theme.Parse(data, "theme-custom")
	if err != nil {
		t.Fatalf("examples/theme-custom.json: %v", err)
	}
	if th.Status == nil || th.Derive().Added != "#2ecc71" {
		t.Errorf("status override lost: %+v", th.Status)
	}
}

func TestFlowFixturesParse(t *testing.T) {
	for _, n := range []string{"states", "real"} {
		data, err := os.ReadFile(filepath.Join("..", "diagram", "testdata", n+".flow.json"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := diagram.Parse(data); err != nil {
			t.Errorf("%s: %v", n, err)
		}
	}
}
