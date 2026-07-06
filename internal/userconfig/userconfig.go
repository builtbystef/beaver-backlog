// Package userconfig owns Busy Beaver's per-machine user configuration — the
// personal settings that live outside the project and are never committed,
// most importantly the saved actor identity. It is separate from the committed
// project config so that cloning a repository never makes a contributor
// inherit someone else's identity. The config directory is injected (in
// production, the OS user-config dir, e.g. ~/.config/beaver) so tests never
// touch a real home.
package userconfig

import (
	"errors"
	"maps"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// fileName is the config file's name within the user-config directory.
const fileName = "config.yml"

// header prefaces the saved file so a human who finds it understands what it is.
const header = "# Busy Beaver per-machine user configuration.\n" +
	"# Personal and never committed; safe to read and edit by hand.\n"

// Config is the per-machine user configuration.
type Config struct {
	// Actor is the human's saved identity, established interactively and reused
	// on later interactive runs. It is never written to a committed file or to
	// BUSY_BEAVER_ACTOR, either of which a child agent process would inherit and
	// act under.
	Actor string `yaml:"actor,omitempty"`
}

// Path is the config file's location within dir.
func Path(dir string) string { return filepath.Join(dir, fileName) }

// Load reads the user config from dir. A missing file — or an empty dir path —
// yields a zero Config and no error: having no saved identity yet is a normal
// state.
func Load(dir string) (Config, error) {
	if dir == "" {
		return Config{}, nil
	}
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes c to dir, creating the directory if needed. Keys a hand-edit
// added are carried through the rewrite with c's fields laid over them
// (hand-written comments are not preserved). The file is small and rarely
// rewritten, so a plain (non-atomic) write is sufficient.
func Save(dir string, c Config) error {
	if dir == "" {
		return errors.New("no user config directory available; set one to save an identity")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	merged, err := mergeExisting(Path(dir), c)
	if err != nil {
		return err
	}
	body, err := yaml.Marshal(merged)
	if err != nil {
		return err
	}
	return os.WriteFile(Path(dir), append([]byte(header), body...), 0o644)
}

// mergeExisting lays c's fields over whatever keys the file at path already
// holds, so a rewrite preserves what a hand-edit added. The overlay goes
// through YAML so any Config field folds in without this function knowing it.
// A missing or unparseable file contributes nothing: saving must not fail on a
// corrupt personal config.
func mergeExisting(path string, c Config) (map[string]any, error) {
	existing := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if yaml.Unmarshal(data, &existing) != nil || existing == nil {
			existing = map[string]any{}
		}
	}
	overlay, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	var own map[string]any
	if err := yaml.Unmarshal(overlay, &own); err != nil {
		return nil, err
	}
	maps.Copy(existing, own)
	return existing, nil
}
