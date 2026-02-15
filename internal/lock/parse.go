package lock

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads a workspace.lock.yaml file.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is workspace lock file path
	if err != nil {
		return nil, fmt.Errorf("reading lock file: %w", err)
	}
	return Parse(data)
}

// Parse parses workspace.lock.yaml content.
func Parse(data []byte) (*File, error) {
	var lf File
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing lock YAML: %w", err)
	}
	return &lf, nil
}

// Save writes the lock file to disk.
func Save(path string, lf *File) error {
	data, err := yaml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("marshaling lock file: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil { //nolint:gosec // lock file needs to be readable
		return fmt.Errorf("writing lock file: %w", err)
	}
	return nil
}
