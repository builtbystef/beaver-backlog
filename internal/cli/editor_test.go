package cli_test

import (
	"os"
	"strings"
	"testing"
)

// editWith returns a fake $EDITOR that applies transform to the file it is handed —
// the same read-modify-write a human performs in an editor.
func editWith(t *testing.T, transform func(string) string) func(string) error {
	t.Helper()
	return func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(path, []byte(transform(string(data))), 0o644)
	}
}

// neverCalled returns a fake editor that fails the test if it is ever opened —
// used to prove a command refused, or short-circuited, before spawning one.
func neverCalled(t *testing.T) func(string) error {
	t.Helper()
	return func(string) error {
		t.Error("the editor was opened, but the command should not have opened one")
		return nil
	}
}

// setTitleLine rewrites the `title:` frontmatter line to the given title,
// independent of the exact form the empty-title skeleton serializes to.
func setTitleLine(s, title string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if strings.HasPrefix(ln, "title:") {
			lines[i] = "title: " + title
			break
		}
	}
	return strings.Join(lines, "\n")
}
