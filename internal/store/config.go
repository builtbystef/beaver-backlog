package store

import (
	"errors"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Config is the committed, shared project configuration Busy Beaver keeps at
// .beaver/config.yml. Unlike the per-machine user config that holds a human's
// identity (ADR 0008), it travels with the repository, so a setting here is a
// project-wide policy every actor — human or agent — sees on checkout.
//
// It is intentionally small and, like the issue files, safe to read and edit by
// hand. Unknown keys are ignored on read rather than rejected, so a config written
// by a newer Busy Beaver still loads in an older one.
type Config struct {
	// FormatVersion records the on-disk layout the store was created with. It is
	// informational today; a future version reads it to detect and migrate an older
	// store.
	FormatVersion int `yaml:"format_version"`

	// CommitOnDone opts the project in to commit-per-issue: when true, `beaver done`
	// records each completed issue as its own atomic commit through the configured
	// VCS adapter (ADR 0007). It defaults to false — Busy Beaver commits nothing and
	// never requires a VCS (ADR 0006) — so the auto-commit is never a surprising
	// default that injects noise into a project's history.
	CommitOnDone bool `yaml:"commit_on_done"`
}

// Config reads the project configuration from .beaver/config.yml. A missing file is
// not an error: it yields defaults (the format version the store ships, commit-on-
// done off), matching how an older store — or this project's own, created before a
// config file existed — still runs. A present-but-unparseable file is a real error
// the caller surfaces, since a hand-edit has corrupted a committed file every actor
// relies on. Unknown keys are tolerated, so a forward-compatible field never fails
// the read.
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

// defaultConfig is the config file Init writes for a new store: the format version,
// plus a commented-out, discoverable pointer to the opt-in commit-on-done setting so
// a user finds the feature by reading the file the tool created. The setting is left
// commented (and thus off) because Busy Beaver commits nothing by default (ADR 0006).
func defaultConfig() []byte {
	return fmt.Appendf(nil,
		`# Busy Beaver project configuration.
# Committed and shared through version control, like the issues themselves.
# Safe to read and edit by hand.
format_version: %d

# Optionally let Busy Beaver drive your VCS. With commit_on_done enabled, "beaver done"
# records each completed issue as its own atomic commit through the Git adapter.
# Off by default: Busy Beaver otherwise commits nothing and never requires a VCS
# (ADR 0006, ADR 0007).
# commit_on_done: true
`, FormatVersion)
}
