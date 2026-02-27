package main

import (
	"testing"

	"github.com/fbkclanna/agentws/internal/manifest"
)

func TestRepoIDFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"git@github.com:org/my-repo.git", "my-repo"},
		{"git@github.com:org/my-repo", "my-repo"},
		{"https://github.com/org/my-repo.git", "my-repo"},
		{"https://github.com/org/my-repo", "my-repo"},
		{"git@gitlab.com:group/subgroup/repo.git", "repo"},
		{"https://gitlab.com/group/subgroup/repo.git", "repo"},
		{"ssh://git@github.com/org/backend.git", "backend"},
		{"git@github.com:org/frontend.git", "frontend"},
		// Trailing slash
		{"https://github.com/org/my-repo/", "my-repo"},
		{"git@github.com:org/my-repo.git/", "my-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := repoIDFromURL(tt.url)
			if got != tt.want {
				t.Errorf("repoIDFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestBuildWorkspace(t *testing.T) {
	repos := []manifest.Repo{
		{ID: "backend", URL: "git@github.com:org/backend.git", Path: "repos/backend", Ref: "main"},
		{ID: "frontend", URL: "git@github.com:org/frontend.git", Path: "repos/frontend", Ref: "develop"},
	}

	data, err := buildWorkspace("myws", "repos", "main", repos)
	if err != nil {
		t.Fatalf("buildWorkspace error: %v", err)
	}
	ws, err := manifest.Parse(data)
	if err != nil {
		t.Fatalf("buildWorkspace produced invalid manifest: %v", err)
	}
	if ws.Name != "myws" {
		t.Errorf("name = %q, want %q", ws.Name, "myws")
	}
	if ws.Version != 1 {
		t.Errorf("version = %d, want 1", ws.Version)
	}
	if ws.ReposRoot != "repos" {
		t.Errorf("repos_root = %q, want %q", ws.ReposRoot, "repos")
	}
	if ws.Defaults.BaseRef != "main" {
		t.Errorf("defaults.base_ref = %q, want %q", ws.Defaults.BaseRef, "main")
	}
	if len(ws.Repos) != 2 {
		t.Fatalf("repos count = %d, want 2", len(ws.Repos))
	}
	if ws.Repos[0].ID != "backend" {
		t.Errorf("repos[0].id = %q, want %q", ws.Repos[0].ID, "backend")
	}
	if ws.Repos[1].Ref != "develop" {
		t.Errorf("repos[1].ref = %q, want %q", ws.Repos[1].Ref, "develop")
	}
}

func TestBuildWorkspace_empty(t *testing.T) {
	data, err := buildWorkspace("empty", "repos", "", nil)
	if err != nil {
		t.Fatalf("buildWorkspace error: %v", err)
	}
	ws, err := manifest.Parse(data)
	if err != nil {
		t.Fatalf("buildWorkspace with empty repos produced invalid manifest: %v", err)
	}
	if ws.Name != "empty" {
		t.Errorf("name = %q, want %q", ws.Name, "empty")
	}
	if len(ws.Repos) != 0 {
		t.Errorf("repos count = %d, want 0", len(ws.Repos))
	}
	if ws.Defaults.BaseRef != "" {
		t.Errorf("defaults.base_ref = %q, want empty", ws.Defaults.BaseRef)
	}
}

func TestBuildWorkspace_withBaseRef(t *testing.T) {
	repos := []manifest.Repo{
		{ID: "svc", URL: "git@github.com:org/svc.git", Path: "repos/svc", Ref: "develop"},
	}

	data, err := buildWorkspace("ws", "repos", "develop", repos)
	if err != nil {
		t.Fatalf("buildWorkspace error: %v", err)
	}
	ws, err := manifest.Parse(data)
	if err != nil {
		t.Fatalf("buildWorkspace produced invalid manifest: %v", err)
	}
	if ws.Defaults.BaseRef != "develop" {
		t.Errorf("defaults.base_ref = %q, want %q", ws.Defaults.BaseRef, "develop")
	}
}

func TestLocalRepoIDValidator(t *testing.T) {
	seenIDs := map[string]bool{"existing": true}
	validate := localRepoIDValidator(seenIDs)

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"empty string", "", true, "repository name is required"},
		{"dot", ".", true, `invalid repository name "."`},
		{"double dot", "..", true, `invalid repository name ".."`},
		{"slash", "foo/bar", true, "repository name must not contain path separators"},
		{"backslash", `foo\bar`, true, "repository name must not contain path separators"},
		{"duplicate", "existing", true, `repository ID "existing" is already added`},
		{"valid name", "my-service", false, ""},
		{"valid with underscore", "my_service", false, ""},
		{"whitespace trimmed", "  my-svc  ", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBuildWorkspace_mixedLocalAndRemote(t *testing.T) {
	repos := []manifest.Repo{
		{
			ID:      "backend",
			URL:     "git@github.com:org/backend.git",
			Path:    "repos/backend",
			Ref:     "main",
			BaseRef: "main",
		},
		{
			ID:    "config",
			Local: true,
			Path:  "repos/config",
			Ref:   "main",
		},
	}

	data, err := buildWorkspace("mixed", "repos", "main", repos)
	if err != nil {
		t.Fatalf("buildWorkspace error: %v", err)
	}
	ws, err := manifest.Parse(data)
	if err != nil {
		t.Fatalf("buildWorkspace produced invalid manifest: %v", err)
	}
	if ws.Name != "mixed" {
		t.Errorf("name = %q, want %q", ws.Name, "mixed")
	}
	if len(ws.Repos) != 2 {
		t.Fatalf("repos count = %d, want 2", len(ws.Repos))
	}

	// Remote repo
	if ws.Repos[0].URL != "git@github.com:org/backend.git" {
		t.Errorf("repos[0].url = %q, want remote URL", ws.Repos[0].URL)
	}
	if ws.Repos[0].Local {
		t.Errorf("repos[0].local = true, want false")
	}

	// Local repo
	if ws.Repos[1].URL != "" {
		t.Errorf("repos[1].url = %q, want empty", ws.Repos[1].URL)
	}
	if !ws.Repos[1].Local {
		t.Errorf("repos[1].local = false, want true")
	}
	if ws.Repos[1].ID != "config" {
		t.Errorf("repos[1].id = %q, want %q", ws.Repos[1].ID, "config")
	}
	if ws.Repos[1].Path != "repos/config" {
		t.Errorf("repos[1].path = %q, want %q", ws.Repos[1].Path, "repos/config")
	}
	if ws.Repos[1].Ref != "main" {
		t.Errorf("repos[1].ref = %q, want %q", ws.Repos[1].Ref, "main")
	}
}
