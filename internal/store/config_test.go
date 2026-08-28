package store_test

import (
	"os"
	"path/filepath"
	"strings"
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

// Naming a project records the name beside what the config already holds: the
// format version and the comments that make it worth hand-editing.
func TestSetProjectNameKeepsTheRestOfTheConfig(t *testing.T) {
	root := newStore(t)
	st, _ := store.Discover(root)

	if err := st.SetProjectName("Apollo Guidance"); err != nil {
		t.Fatalf("SetProjectName: %v", err)
	}

	cfg, err := st.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg.Name != "Apollo Guidance" {
		t.Errorf("configured name = %q, want %q", cfg.Name, "Apollo Guidance")
	}
	if cfg.FormatVersion != store.FormatVersion {
		t.Errorf("format version = %d, want the shipped %d", cfg.FormatVersion, store.FormatVersion)
	}
	if got := st.ProjectName(); got != "Apollo Guidance" {
		t.Errorf("project name = %q, want the name just set", got)
	}
	if raw := readConfig(t, root); !strings.Contains(raw, "# Beaver Backlog project configuration.") {
		t.Errorf("the config lost the comments that make it hand-editable:\n%s", raw)
	}
}

// A second naming replaces the first rather than leaving two, and an empty name
// is refused: the way to have no name is to have no key.
func TestSetProjectNameReplacesAndRefusesEmpty(t *testing.T) {
	root := newStore(t)
	st, _ := store.Discover(root)

	if err := st.SetProjectName("Apollo"); err != nil {
		t.Fatalf("SetProjectName: %v", err)
	}
	if err := st.SetProjectName("Gemini"); err != nil {
		t.Fatalf("second SetProjectName: %v", err)
	}
	if cfg, _ := st.Config(); cfg.Name != "Gemini" {
		t.Errorf("configured name = %q, want the second name", cfg.Name)
	}
	if raw := readConfig(t, root); strings.Count(raw, "name:") != 1 {
		t.Errorf("renaming left more than one name key:\n%s", raw)
	}

	if err := st.SetProjectName("   "); err == nil {
		t.Error("an empty project name was accepted; it should be refused")
	}
	if cfg, _ := st.Config(); cfg.Name != "Gemini" {
		t.Errorf("a refused name changed the config to %q", cfg.Name)
	}
}

func readConfig(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".beaver", "config.yml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	return string(data)
}
