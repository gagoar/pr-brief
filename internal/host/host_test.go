package host

import (
	"testing"

	"github.com/gagoar/pr-brief/internal/convention"
)

func TestDetect(t *testing.T) {
	cases := map[string]string{
		"https://github.com/gagoar/pr-brief.git":                GitHub,
		"git@github.com:gagoar/pr-brief.git":                    GitHub,
		"https://dev.azure.com/org/proj/_git/repo":              AzureDevOps,
		"https://org@dev.azure.com/org/proj/_git/repo":          AzureDevOps,
		"git@ssh.dev.azure.com:v3/org/proj/repo":                AzureDevOps,
		"https://org.visualstudio.com/proj/_git/repo":           AzureDevOps,
		"org@vs-ssh.visualstudio.com:v3/org/proj/repo":          AzureDevOps,
		"https://dev.azure.com/org/proj/_git/github.com-mirror": AzureDevOps,
		"https://gitlab.com/a/b.git":                            "",
		"":                                                      "",
	}
	for url, want := range cases {
		if got := Detect(url); got != want {
			t.Errorf("Detect(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestNormalizeAndLimit(t *testing.T) {
	for in, want := range map[string]string{"": "", "GitHub": GitHub, "github.com": GitHub, "ado": AzureDevOps, "dev.azure.com": AzureDevOps, "azure-devops": AzureDevOps} {
		if got, ok := Normalize(in); !ok || got != want {
			t.Errorf("Normalize(%q) = %q, %v", in, got, ok)
		}
	}
	if _, ok := Normalize("gitlab"); ok {
		t.Error("an unknown host must not normalize")
	}
	if Limit(AzureDevOps) != convention.MaxBodyCharsAzureDevOps || Limit(GitHub) != convention.MaxBodyCharsGitHub || Limit("") != convention.MaxBodyCharsGitHub {
		t.Error("the limit follows the host; an unknown host gets the GitHub limit")
	}
}

func TestFromEnv(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if FromEnv(env(map[string]string{"TF_BUILD": "True"})) != AzureDevOps {
		t.Error("TF_BUILD means Azure Pipelines")
	}
	if FromEnv(env(map[string]string{"GITHUB_ACTIONS": "true"})) != GitHub {
		t.Error("GITHUB_ACTIONS means GitHub")
	}
	if FromEnv(env(nil)) != "" {
		t.Error("no CI variable means unknown")
	}
}
