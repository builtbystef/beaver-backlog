// Package userconfig owns Busy Beaver's per-machine user configuration — the
// personal settings that live outside the project and are never committed
// (ADR 0008), most importantly the saved actor identity.
//
// It is deliberately separate from the committed project config the store owns.
// Folding identity into the project config would make every contributor who
// clones the repository inherit the initializer's identity; keeping the personal
// thing per-machine and the shared thing committed is what lets Busy Beaver serve
// solo devs, teams, and unbounded open-source contributors with no per-contributor
// registration. In production this config lives under the OS user-config directory
// (e.g. ~/.config/beaver); the directory is injected so tests never touch a real
// home.
package userconfig

import (
	"errors"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// fileName is the config file's name within the user-config directory.
const fileName = "config.yml"

// header prefaces the saved file so a human who finds it understands what it is.
const header = "# Busy Beaver per-machine user configuration.\n" +
	"# Personal and never committed; safe to read and edit by hand.\n"

// Config is the per-machine user configuration. Today it carries only the saved
// actor identity; it has room for further personal settings later.
type Config struct {
	// Actor is the human's saved Busy Beaver identity, established interactively
	// (seeded from the VCS or typed at a prompt) and reused on later interactive
	// runs. It is used only interactively and is never written to a committed file
	// or to BUSY_BEAVER_ACTOR, both of which a child agent process would inherit and act
	// under (ADR 0008, ADR 0010).
	Actor string `yaml:"actor,omitempty"`
}

// Path is the config file's location within dir.
func Path(dir string) string { return filepath.Join(dir, fileName) }

// Load reads the user config from dir. A missing file — or an empty dir path —
// yields a zero Config and no error: having no saved identity yet is a normal
// state, not a failure, and is exactly what triggers the interactive setup.
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

// Save writes c to dir, creating the directory if needed. The file is small and
// rewritten only on an interactive identity change, so a plain write (not the
// store's atomic temp-and-rename) is sufficient.
func Save(dir string, c Config) error {
	if dir == "" {
		return errors.New("no user config directory available; set one to save an identity")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(Path(dir), append([]byte(header), body...), 0o644)
}
