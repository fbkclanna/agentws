package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Validate checks the workspace manifest for errors.
func Validate(ws *Workspace) error { return validate(ws) }

// Save validates and writes a workspace manifest to disk.
func Save(path string, ws *Workspace) error {
	if err := validate(ws); err != nil {
		return err
	}
	data, err := yaml.Marshal(ws)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return nil
}

// Load reads and validates a workspace.yaml file.
func Load(path string) (*Workspace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	return Parse(data)
}

// Parse parses and validates workspace.yaml content.
func Parse(data []byte) (*Workspace, error) {
	var ws Workspace
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("parsing manifest YAML: %w", err)
	}
	if err := validate(&ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

func validate(ws *Workspace) error {
	if ws.Version != 1 {
		return fmt.Errorf("unsupported manifest version: %d (expected 1)", ws.Version)
	}
	if ws.Name == "" {
		return fmt.Errorf("manifest: name is required")
	}

	if err := validateBaseRef(ws.Defaults.BaseRef, "defaults.base_ref"); err != nil {
		return err
	}

	seen := make(map[string]bool, len(ws.Repos))
	for i, r := range ws.Repos {
		if err := validateRepo(i, r, seen); err != nil {
			return err
		}
		seen[r.ID] = true
	}

	if ws.ReposRoot != "" {
		if err := validatePath(ws.ReposRoot, "repos_root"); err != nil {
			return err
		}
	}

	return nil
}

func validateRepo(i int, r Repo, seen map[string]bool) error {
	if r.ID == "" {
		return fmt.Errorf("manifest: repos[%d].id is required", i)
	}
	if r.URL == "" {
		return fmt.Errorf("manifest: repos[%d] (%s).url is required", i, r.ID)
	}
	if r.Path == "" {
		return fmt.Errorf("manifest: repos[%d] (%s).path is required", i, r.ID)
	}
	if err := validatePath(r.Path, r.ID); err != nil {
		return err
	}
	if seen[r.ID] {
		return fmt.Errorf("manifest: duplicate repo id %q", r.ID)
	}
	if err := validateBaseRef(r.BaseRef, fmt.Sprintf("repos[%d] (%s).base_ref", i, r.ID)); err != nil {
		return err
	}
	for j, ps := range r.PostSync {
		if len(ps.Cmd) == 0 {
			return fmt.Errorf("manifest: repos[%d] (%s).post_sync[%d].cmd is required", i, r.ID, j)
		}
		if ps.WorkDir != "" {
			label := fmt.Sprintf("repos[%d] (%s).post_sync[%d].workdir", i, r.ID, j)
			if err := validatePath(ps.WorkDir, label); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateBaseRef ensures a base_ref is a branch name only (no origin/ or refs/ prefix).
func validateBaseRef(v, label string) error {
	if v == "" {
		return nil
	}
	if strings.HasPrefix(v, "origin/") || strings.HasPrefix(v, "refs/") {
		return fmt.Errorf("manifest: %s must be branch name only (no origin/ or refs/ prefix): %s", label, v)
	}
	return nil
}

// validatePath ensures a path is relative and does not escape the workspace.
func validatePath(p, label string) error {
	if filepath.IsAbs(p) {
		return fmt.Errorf("manifest: %s: absolute path is not allowed: %s", label, p)
	}
	cleaned := filepath.Clean(p)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("manifest: %s: path must not escape workspace (contains ..): %s", label, p)
	}
	return nil
}

// FilterRepos returns the subset of repos matching the given profile.
func FilterRepos(ws *Workspace, profileName string) ([]Repo, error) {
	prof, ok := ws.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in manifest", profileName)
	}
	return filterByProfile(ws.Repos, prof), nil
}

func filterByProfile(repos []Repo, prof Profile) []Repo {
	excludeSet := toSet(prof.ExcludeRepoIDs)
	includeSet := toSet(prof.IncludeRepoIDs)
	tagSet := toSet(prof.IncludeTags)

	var result []Repo
	for _, r := range repos {
		if excludeSet[r.ID] {
			continue
		}
		if includeSet[r.ID] {
			result = append(result, r)
			continue
		}
		if len(tagSet) > 0 && hasAnyTag(r.Tags, tagSet) {
			result = append(result, r)
			continue
		}
		// If profile has no tags and no includes, nothing matches by default.
	}
	return result
}

// FilterByIDs returns repos matching --only / --skip flags.
func FilterByIDs(repos []Repo, only, skip []string) []Repo {
	if len(only) == 0 && len(skip) == 0 {
		return repos
	}
	onlySet := toSet(only)
	skipSet := toSet(skip)

	var result []Repo
	for _, r := range repos {
		if len(onlySet) > 0 && !onlySet[r.ID] {
			continue
		}
		if skipSet[r.ID] {
			continue
		}
		result = append(result, r)
	}
	return result
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func hasAnyTag(tags []string, tagSet map[string]bool) bool {
	for _, t := range tags {
		if tagSet[t] {
			return true
		}
	}
	return false
}
