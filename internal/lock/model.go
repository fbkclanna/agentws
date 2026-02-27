package lock

// File represents workspace.lock.yaml.
type File struct {
	Version     int              `yaml:"version"`
	Name        string           `yaml:"name"`
	GeneratedAt string           `yaml:"generated_at"`
	ToolVersion string           `yaml:"tool_version"`
	Repos       map[string]*Repo `yaml:"repos"`
}

// Repo records the pinned state of a single repository.
type Repo struct {
	URL    string `yaml:"url,omitempty"`
	Ref    string `yaml:"ref"`
	Commit string `yaml:"commit"`
}
