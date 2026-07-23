package userconfig_test

import (
	"os"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/userconfig"
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

// A missing config file is a normal "no saved identity yet" state, not an error.
func TestLoadMissingIsEmpty(t *testing.T) {
	got, err := userconfig.Load(t.TempDir()) // dir exists, file does not
	if err != nil {
		t.Fatalf("Load of a missing file errored: %v", err)
	}
	if got.Actor != "" {
		t.Errorf("missing config actor = %q, want empty", got.Actor)
	}
}

// An empty directory path loads as empty without error, but cannot be saved to.
func TestEmptyDirLoadsEmptySaveErrors(t *testing.T) {
	if _, err := userconfig.Load(""); err != nil {
		t.Errorf("Load(\"\") errored: %v", err)
	}
	if err := userconfig.Save("", userconfig.Config{Actor: "x"}); err == nil {
		t.Error("Save(\"\") should error: there is nowhere to write")
	}
}

// Keys a hand-edit added survive a later Save instead of being dropped by a
// rewrite from the struct alone.
func TestSavePreservesHandAddedKeys(t *testing.T) {
	dir := t.TempDir()
	handEdited := "actor: old-name\nfavorite_color: green\n"
	if err := os.WriteFile(userconfig.Path(dir), []byte(handEdited), 0o644); err != nil {
		t.Fatalf("seed hand-edited config: %v", err)
	}

	if err := userconfig.Save(dir, userconfig.Config{Actor: "new-name"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := userconfig.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Actor != "new-name" {
		t.Errorf("actor = %q, want the newly saved new-name", got.Actor)
	}
	data, err := os.ReadFile(userconfig.Path(dir))
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if !strings.Contains(string(data), "favorite_color: green") {
		t.Errorf("hand-added key was dropped by Save:\n%s", data)
	}
}

// A corrupt existing file must not block saving an identity: Save rewrites from
// the given config alone rather than failing.
func TestSaveToleratesCorruptExistingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(userconfig.Path(dir), []byte(":\tnot yaml ["), 0o644); err != nil {
		t.Fatalf("seed corrupt config: %v", err)
	}
	if err := userconfig.Save(dir, userconfig.Config{Actor: "stefan"}); err != nil {
		t.Fatalf("Save over a corrupt file: %v", err)
	}
	got, err := userconfig.Load(dir)
	if err != nil || got.Actor != "stefan" {
		t.Errorf("Load = %+v, %v; want actor stefan and no error", got, err)
	}
}

// The saved file carries a human-readable header so it is recognizable on disk.
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
