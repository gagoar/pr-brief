// Package host names the place a pull request lives and the limits that place sets.
// It detects the host from facts the developer already has: the git remote, the CI
// environment or the tool that opens the PR. It is not a setting.
package host

import (
	"os/exec"
	"strings"

	"github.com/gagoar/pr-brief/internal/convention"
)

// The two hosts pr-brief writes for.
const (
	GitHub      = "github"
	AzureDevOps = "azure-devops"
)

// Normalize maps the names people type to a host constant. ok is false for an unknown name.
// An empty name is valid and means "not known".
func Normalize(s string) (host string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return "", true
	case "github", "github.com", "gh":
		return GitHub, true
	case "azure-devops", "azuredevops", "azure", "ado", "dev.azure.com":
		return AzureDevOps, true
	}
	return "", false
}

// Detect reads a git remote URL. It returns "" when the URL names neither host.
func Detect(remoteURL string) string {
	u := strings.ToLower(strings.TrimSpace(remoteURL))
	switch {
	case u == "":
		return ""
	case strings.Contains(u, "dev.azure.com"), strings.Contains(u, ".visualstudio.com"):
		return AzureDevOps
	case strings.Contains(u, "github.com"):
		return GitHub
	}
	return ""
}

// Remote is the URL of the origin remote of the repository at dir, or "" when there is none.
func Remote(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// FromGit detects the host from the origin remote of the repository at dir.
func FromGit(dir string) string { return Detect(Remote(dir)) }

// FromEnv detects the host from the variables CI systems set.
func FromEnv(getenv func(string) string) string {
	switch {
	case getenv("TF_BUILD") != "":
		return AzureDevOps
	case getenv("GITHUB_ACTIONS") != "":
		return GitHub
	}
	return ""
}

// Limit is the longest description the host accepts. An unknown host gets the GitHub limit.
func Limit(h string) int {
	if h == AzureDevOps {
		return convention.MaxBodyCharsAzureDevOps
	}
	return convention.MaxBodyCharsGitHub
}

// Name is the host as a person writes it.
func Name(h string) string {
	if h == AzureDevOps {
		return "Azure DevOps"
	}
	return "GitHub"
}
