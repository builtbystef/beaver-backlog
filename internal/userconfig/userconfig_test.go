package userconfig_test

import (
	"os"
	"strings"
	"testing"

	"beaver/internal/userconfig"
)

// A saved identity round-trips through Save and Load.
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := userconfig.Save(dir, userconfig.Config{Actor: "stefan"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := userconfig.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Actor != "stefan" {
		t.Errorf("loaded actor = %q, want stefan", got.Actor)
	}
}

// A missing config file is a normal "no saved identity yet" state, not an error —
// this is what makes the interactive setup trigger on first use.
func TestLoadMissingIsEmpty(t *testing.T) {
	got, err := userconfig.Load(t.TempDir()) // dir exists, file does not
	if err != nil {
		t.Fatalf("Load of a missing file errored: %v", err)
	}
	if got.Actor != "" {
		t.Errorf("missing config actor = %q, want empty", got.Actor)
	}
}

// An empty directory path (e.g. the OS user-config dir could not be determined)
// loads as empty without error, but cannot be saved to.
func TestEmptyDirLoadsEmptySaveErrors(t *testing.T) {
	if _, err := userconfig.Load(""); err != nil {
		t.Errorf("Load(\"\") errored: %v", err)
	}
	if err := userconfig.Save("", userconfig.Config{Actor: "x"}); err == nil {
		t.Error("Save(\"\") should error: there is nowhere to write")
	}
}

// The saved file carries a human-readable header and is never committed to a repo,
// so it should be plainly recognizable on disk.
func TestSavedFileIsAnnotated(t *testing.T) {
	dir := t.TempDir()
	if err := userconfig.Save(dir, userconfig.Config{Actor: "claude"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(userconfig.Path(dir))
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "never committed") {
		t.Errorf("saved file lacks its explanatory header:\n%s", s)
	}
	if !strings.Contains(s, "actor: claude") {
		t.Errorf("saved file missing the actor field:\n%s", s)
	}
}
