package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_valid(t *testing.T) {
	data := []byte(`
version: 1
name: foo
repos_root: repos
defaults:
  depth: 50
profiles:
  core:
    include_tags: ["core"]
repos:
  - id: backend
    url: git@github.com:org/backend.git
    path: repos/backend
    ref: main
    tags: ["core"]
  - id: frontend
    url: git@github.com:org/frontend.git
    path: repos/frontend
    tags: ["core"]
`)
	ws, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Name != "foo" {
		t.Errorf("name = %q, want %q", ws.Name, "foo")
	}
	if len(ws.Repos) != 2 {
		t.Errorf("repos count = %d, want 2", len(ws.Repos))
	}
	if ws.Defaults.Depth == nil || *ws.Defaults.Depth != 50 {
		t.Error("defaults.depth should be 50")
	}
}

func TestParse_missingVersion(t *testing.T) {
	data := []byte(`
name: foo
repos: []
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestParse_missingName(t *testing.T) {
	data := []byte(`
version: 1
repos: []
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParse_duplicateRepoID(t *testing.T) {
	data := []byte(`
version: 1
name: foo
repos:
  - id: backend
    url: git@github.com:org/a.git
    path: repos/a
  - id: backend
    url: git@github.com:org/b.git
    path: repos/b
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for duplicate repo id")
	}
}

func TestParse_absolutePath(t *testing.T) {
	data := []byte(`
version: 1
name: foo
repos:
  - id: backend
    url: git@github.com:org/a.git
    path: /tmp/repos/backend
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestParse_dotdotPath(t *testing.T) {
	data := []byte(`
version: 1
name: foo
repos:
  - id: backend
    url: git@github.com:org/a.git
    path: ../outside
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for .. path")
	}
}

func TestParse_missingRepoFields(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{"missing id", `
version: 1
name: foo
repos:
  - url: git@github.com:org/a.git
    path: repos/a
`},
		{"missing url", `
version: 1
name: foo
repos:
  - id: a
    path: repos/a
`},
		{"missing path", `
version: 1
name: foo
repos:
  - id: a
    url: git@github.com:org/a.git
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParse_localRepo(t *testing.T) {
	t.Run("local without url is valid", func(t *testing.T) {
		data := []byte(`
version: 1
name: foo
repos:
  - id: my-service
    local: true
    path: repos/my-service
    ref: main
`)
		ws, err := Parse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ws.Repos[0].Local {
			t.Error("expected local=true")
		}
		if !ws.Repos[0].IsLocal() {
			t.Error("IsLocal() should return true")
		}
		if ws.Repos[0].URL != "" {
			t.Errorf("URL should be empty, got %q", ws.Repos[0].URL)
		}
	})

	t.Run("local with url is rejected", func(t *testing.T) {
		data := []byte(`
version: 1
name: foo
repos:
  - id: my-service
    local: true
    url: git@github.com:org/a.git
    path: repos/my-service
`)
		_, err := Parse(data)
		if err == nil {
			t.Fatal("expected error for local repo with url")
		}
	})

	t.Run("non-local without url is rejected", func(t *testing.T) {
		data := []byte(`
version: 1
name: foo
repos:
  - id: my-service
    path: repos/my-service
`)
		_, err := Parse(data)
		if err == nil {
			t.Fatal("expected error for non-local repo without url")
		}
	})

	t.Run("local repo round-trip", func(t *testing.T) {
		ws := &Workspace{
			Version:   1,
			Name:      "local-test",
			ReposRoot: "repos",
			Repos: []Repo{
				{ID: "remote-svc", URL: "https://example.com/a.git", Path: "repos/remote-svc", Ref: "main"},
				{ID: "local-svc", Local: true, Path: "repos/local-svc", Ref: "main"},
			},
		}

		dir := t.TempDir()
		path := filepath.Join(dir, "workspace.yaml")

		if err := Save(path, ws); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loaded, err := Load(path)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if len(loaded.Repos) != 2 {
			t.Fatalf("repos count = %d, want 2", len(loaded.Repos))
		}
		if loaded.Repos[0].URL != "https://example.com/a.git" {
			t.Errorf("repos[0].url = %q, want remote URL", loaded.Repos[0].URL)
		}
		if !loaded.Repos[1].Local {
			t.Error("repos[1] should be local")
		}
		if loaded.Repos[1].URL != "" {
			t.Errorf("repos[1].url should be empty, got %q", loaded.Repos[1].URL)
		}
	})
}

func TestFilterRepos_byProfile(t *testing.T) {
	ws := &Workspace{
		Profiles: map[string]Profile{
			"core": {IncludeTags: []string{"core"}},
		},
		Repos: []Repo{
			{ID: "a", Tags: []string{"core"}},
			{ID: "b", Tags: []string{"data"}},
			{ID: "c", Tags: []string{"core", "data"}},
		},
	}
	repos, err := FilterRepos(ws, "core")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2", len(repos))
	}
	ids := map[string]bool{}
	for _, r := range repos {
		ids[r.ID] = true
	}
	if !ids["a"] || !ids["c"] {
		t.Errorf("expected a and c, got %v", ids)
	}
}

func TestFilterRepos_unknownProfile(t *testing.T) {
	ws := &Workspace{
		Profiles: map[string]Profile{},
	}
	_, err := FilterRepos(ws, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func TestFilterByIDs(t *testing.T) {
	repos := []Repo{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}

	t.Run("only", func(t *testing.T) {
		result := FilterByIDs(repos, []string{"a", "c"}, nil)
		if len(result) != 2 {
			t.Errorf("got %d, want 2", len(result))
		}
	})

	t.Run("skip", func(t *testing.T) {
		result := FilterByIDs(repos, nil, []string{"b"})
		if len(result) != 2 {
			t.Errorf("got %d, want 2", len(result))
		}
	})

	t.Run("none", func(t *testing.T) {
		result := FilterByIDs(repos, nil, nil)
		if len(result) != 3 {
			t.Errorf("got %d, want 3", len(result))
		}
	})
}

func TestRepo_EffectiveRef(t *testing.T) {
	r := Repo{Ref: "develop"}
	if r.EffectiveRef() != "develop" {
		t.Error("expected develop")
	}
	r2 := Repo{}
	if r2.EffectiveRef() != "main" {
		t.Error("expected main as default")
	}
}

func TestRepo_IsRequired(t *testing.T) {
	r := Repo{}
	if !r.IsRequired() {
		t.Error("default should be required")
	}
	f := false
	r2 := Repo{Required: &f}
	if r2.IsRequired() {
		t.Error("should not be required")
	}
}

func TestValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ws := &Workspace{
			Version: 1,
			Name:    "test",
			Repos: []Repo{
				{ID: "a", URL: "https://example.com/a.git", Path: "repos/a"},
			},
		}
		if err := Validate(ws); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		ws := &Workspace{Version: 0, Name: "test"}
		if err := Validate(ws); err == nil {
			t.Fatal("expected error for invalid version")
		}
	})

	t.Run("duplicate ID", func(t *testing.T) {
		ws := &Workspace{
			Version: 1,
			Name:    "test",
			Repos: []Repo{
				{ID: "a", URL: "https://example.com/a.git", Path: "repos/a"},
				{ID: "a", URL: "https://example.com/b.git", Path: "repos/b"},
			},
		}
		if err := Validate(ws); err == nil {
			t.Fatal("expected error for duplicate ID")
		}
	})
}

func TestRepo_EffectiveBaseRef(t *testing.T) {
	t.Run("repo level", func(t *testing.T) {
		r := Repo{BaseRef: "staging"}
		got := r.EffectiveBaseRef(Defaults{})
		if got != "staging" {
			t.Errorf("got %q, want %q", got, "staging")
		}
	})

	t.Run("defaults level", func(t *testing.T) {
		r := Repo{}
		got := r.EffectiveBaseRef(Defaults{BaseRef: "develop"})
		if got != "develop" {
			t.Errorf("got %q, want %q", got, "develop")
		}
	})

	t.Run("empty when unset", func(t *testing.T) {
		r := Repo{}
		got := r.EffectiveBaseRef(Defaults{})
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("repo overrides defaults", func(t *testing.T) {
		r := Repo{BaseRef: "staging"}
		got := r.EffectiveBaseRef(Defaults{BaseRef: "develop"})
		if got != "staging" {
			t.Errorf("got %q, want %q", got, "staging")
		}
	})
}

func TestParse_baseRef(t *testing.T) {
	data := []byte(`
version: 1
name: foo
defaults:
  base_ref: develop
repos:
  - id: backend
    url: git@github.com:org/backend.git
    path: repos/backend
    ref: main
  - id: frontend
    url: git@github.com:org/frontend.git
    path: repos/frontend
    ref: main
    base_ref: staging
`)
	ws, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Defaults.BaseRef != "develop" {
		t.Errorf("defaults.base_ref = %q, want %q", ws.Defaults.BaseRef, "develop")
	}
	if ws.Repos[0].BaseRef != "" {
		t.Errorf("repos[0].base_ref = %q, want empty", ws.Repos[0].BaseRef)
	}
	if ws.Repos[1].BaseRef != "staging" {
		t.Errorf("repos[1].base_ref = %q, want %q", ws.Repos[1].BaseRef, "staging")
	}
}

func TestValidateBaseRef(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "origin prefix rejected in defaults",
			yaml: `
version: 1
name: foo
defaults:
  base_ref: origin/main
repos:
  - id: a
    url: https://example.com/a.git
    path: repos/a
`,
			wantErr: true,
		},
		{
			name: "refs prefix rejected in repo",
			yaml: `
version: 1
name: foo
repos:
  - id: a
    url: https://example.com/a.git
    path: repos/a
    base_ref: refs/heads/main
`,
			wantErr: true,
		},
		{
			name: "slash in branch name allowed",
			yaml: `
version: 1
name: foo
defaults:
  base_ref: release/2026q1
repos:
  - id: a
    url: https://example.com/a.git
    path: repos/a
`,
			wantErr: false,
		},
		{
			name: "empty base_ref ok",
			yaml: `
version: 1
name: foo
repos:
  - id: a
    url: https://example.com/a.git
    path: repos/a
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSave_roundTrip(t *testing.T) {
	ws := &Workspace{
		Version:   1,
		Name:      "roundtrip",
		ReposRoot: "repos",
		Repos: []Repo{
			{ID: "backend", URL: "https://example.com/backend.git", Path: "repos/backend", Ref: "main"},
			{ID: "frontend", URL: "https://example.com/frontend.git", Path: "repos/frontend", Ref: "develop", Tags: []string{"web"}},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.yaml")

	if err := Save(path, ws); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Name != ws.Name {
		t.Errorf("name = %q, want %q", loaded.Name, ws.Name)
	}
	if loaded.ReposRoot != ws.ReposRoot {
		t.Errorf("repos_root = %q, want %q", loaded.ReposRoot, ws.ReposRoot)
	}
	if len(loaded.Repos) != len(ws.Repos) {
		t.Fatalf("repos count = %d, want %d", len(loaded.Repos), len(ws.Repos))
	}
	for i, r := range loaded.Repos {
		if r.ID != ws.Repos[i].ID {
			t.Errorf("repos[%d].id = %q, want %q", i, r.ID, ws.Repos[i].ID)
		}
		if r.URL != ws.Repos[i].URL {
			t.Errorf("repos[%d].url = %q, want %q", i, r.URL, ws.Repos[i].URL)
		}
	}
}
