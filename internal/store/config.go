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

	// CommitOnDone opts the project in to commit-per-issue: when true, `beaver done`
	// records each completed issue as its own atomic commit through the configured
	// VCS adapter. Defaults to false — by default nothing is committed and no VCS
	// is required.
	CommitOnDone bool `yaml:"commit_on_done"`
}

// Config reads the project configuration from .beaver/config.yml. A missing
// file yields defaults (shipped format version, commit-on-done off), not an
// error; a present-but-unparseable file is a real error. Unknown keys are
// tolerated for forward compatibility.
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

// defaultConfig is the config file Init writes: the format version, plus a
// commented-out pointer to the opt-in commit-on-done setting so a user
// discovers the feature by reading the file.
func defaultConfig() []byte {
	return fmt.Appendf(nil,
		`# Busy Beaver project configuration.
# Committed and shared through version control, like the issues themselves.
# Safe to read and edit by hand.
format_version: %d

# Optionally let Busy Beaver drive your VCS. With commit_on_done enabled, "beaver done"
# records each completed issue as its own atomic commit through the Git adapter.
# Off by default: Busy Beaver otherwise commits nothing and never requires a VCS.
# commit_on_done: true
`, FormatVersion)
}
