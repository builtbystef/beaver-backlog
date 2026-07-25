package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/store"
)

// A freshly initialized store reads back the defaults: the shipped format
// version.
func TestConfigDefaultsAfterInit(t *testing.T) {
	st, _ := store.Discover(newStore(t))
	cfg, err := st.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg.FormatVersion != store.FormatVersion {
		t.Errorf("FormatVersion = %d, want %d", cfg.FormatVersion, store.FormatVersion)
	}
}

// A missing config file yields defaults, not an error.
func TestConfigMissingFileIsDefault(t *testing.T) {
	root := newStore(t)
	if err := os.Remove(filepath.Join(root, ".beaver", "config.yml")); err != nil {
		t.Fatal(err)
	}
	st, _ := store.Discover(root)
	cfg, err := st.Config()
	if err != nil {
		t.Fatalf("Config with no file: %v", err)
	}
	if cfg.FormatVersion != store.FormatVersion {
		t.Errorf("missing-file config = %+v, want defaults", cfg)
	}
}

// Unknown keys are tolerated (forward compatibility); malformed YAML is a real error.
func TestConfigToleratesUnknownAndRejectsMalformed(t *testing.T) {
	root := newStore(t)
	writeConfig(t, root, "format_version: 1\nfuture_setting: 42\n")
	st, _ := store.Discover(root)
	if cfg, err := st.Config(); err != nil || cfg.FormatVersion != 1 {
		t.Errorf("unknown key should be ignored: cfg=%+v err=%v", cfg, err)
	}

	writeConfig(t, root, "format_version: [not a number\n")
	if _, err := st.Config(); err == nil {
		t.Error("malformed config returned no error")
	}
}

func writeConfig(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".beaver", "config.yml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
