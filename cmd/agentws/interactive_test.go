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

	data, err := buildWorkspace("myws", "repos", repos)
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
	data, err := buildWorkspace("empty", "repos", nil)
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
}
