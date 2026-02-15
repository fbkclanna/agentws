package manifest

import (
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
