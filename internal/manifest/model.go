package manifest

// Workspace represents the top-level workspace.yaml manifest.
type Workspace struct {
	Version     int                `yaml:"version"`
	Name        string             `yaml:"name"`
	Description string             `yaml:"description,omitempty"`
	ReposRoot   string             `yaml:"repos_root,omitempty"`
	Profiles    map[string]Profile `yaml:"profiles,omitempty"`
	Defaults    Defaults           `yaml:"defaults,omitempty"`
	Repos       []Repo             `yaml:"repos"`
}

// Defaults defines default clone/checkout options applied to all repos
// unless overridden at the repo level.
type Defaults struct {
	Depth          *int   `yaml:"depth,omitempty"`
	PartialClone   bool   `yaml:"partial_clone,omitempty"`
	SparseCheckout bool   `yaml:"sparse_checkout,omitempty"`
	BaseRef        string `yaml:"base_ref,omitempty"`
}

// Profile selects a subset of repos by tags or explicit IDs.
type Profile struct {
	IncludeTags    []string `yaml:"include_tags,omitempty"`
	IncludeRepoIDs []string `yaml:"include_repo_ids,omitempty"`
	ExcludeRepoIDs []string `yaml:"exclude_repo_ids,omitempty"`
}

// Repo represents a single repository entry in the manifest.
type Repo struct {
	ID           string     `yaml:"id"`
	URL          string     `yaml:"url"`
	Path         string     `yaml:"path"`
	Ref          string     `yaml:"ref,omitempty"`
	BaseRef      string     `yaml:"base_ref,omitempty"`
	Tags         []string   `yaml:"tags,omitempty"`
	Required     *bool      `yaml:"required,omitempty"`
	Depth        *int       `yaml:"depth,omitempty"`
	PartialClone *bool      `yaml:"partial_clone,omitempty"`
	Sparse       []string   `yaml:"sparse,omitempty"`
	PostSync     []PostSync `yaml:"post_sync,omitempty"`
}

// PostSync defines a command to run after syncing a repo.
type PostSync struct {
	Name    string   `yaml:"name,omitempty"`
	WorkDir string   `yaml:"workdir,omitempty"`
	Cmd     []string `yaml:"cmd"`
}

// EffectiveDepth returns the depth for this repo, falling back to defaults.
func (r *Repo) EffectiveDepth(d Defaults) *int {
	if r.Depth != nil {
		return r.Depth
	}
	return d.Depth
}

// EffectivePartialClone returns the partial_clone setting, falling back to defaults.
func (r *Repo) EffectivePartialClone(d Defaults) bool {
	if r.PartialClone != nil {
		return *r.PartialClone
	}
	return d.PartialClone
}

// EffectiveBaseRef returns the base_ref for this repo, falling back to defaults.
// Returns empty string if neither repo nor defaults specifies base_ref.
func (r *Repo) EffectiveBaseRef(d Defaults) string {
	if r.BaseRef != "" {
		return r.BaseRef
	}
	return d.BaseRef
}

// EffectiveRef returns the ref for this repo, defaulting to "main".
func (r *Repo) EffectiveRef() string {
	if r.Ref != "" {
		return r.Ref
	}
	return "main"
}

// IsRequired returns whether this repo is required (default true).
func (r *Repo) IsRequired() bool {
	if r.Required != nil {
		return *r.Required
	}
	return true
}
