package store

import (
	"errors"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Config is the committed, shared project configuration at .beaver/config.yml.
// It travels with the repository, so a setting here is project-wide policy.
// It is safe to edit by hand; unknown keys are ignored on read, so a config
// written by a newer version still loads in an older one.
type Config struct {
	// FormatVersion records the on-disk layout the store was created with.
	FormatVersion int `yaml:"format_version"`
}

// Config reads the project configuration from .beaver/config.yml. A missing
// file yields defaults (the shipped format version), not an error; a
// present-but-unparseable file is a real error. Unknown keys are tolerated for
// forward compatibility.
func (s *Store) Config() (Config, error) {
	cfg := Config{FormatVersion: FormatVersion}
	data, err := os.ReadFile(s.ConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", s.ConfigPath(), err)
	}
	return cfg, nil
}

// defaultConfig is the config file Init writes: the format version, the one
// project-wide setting there is.
func defaultConfig() []byte {
	return fmt.Appendf(nil,
		`# Beaver Backlog project configuration.
# Committed and shared through version control, like the issues themselves.
# Safe to read and edit by hand.
format_version: %d
`, FormatVersion)
}
