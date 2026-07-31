package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/beavertest"
)

func TestServeRejectsPositionalArguments(t *testing.T) {
	h := beavertest.New(t).Init()

	res := h.Run("serve", "8080")
	if res.Code != 2 {
		t.Errorf("exit = %d, want 2 (usage)", res.Code)
	}
	if !strings.Contains(res.Stderr, "--port") {
		t.Errorf("stderr does not point at --port: %s", res.Stderr)
	}
}

func TestServeRejectsAnUnusablePort(t *testing.T) {
	h := beavertest.New(t).Init()

	for _, port := range []string{"nonsense", "99999", "-1"} {
		if res := h.Run("serve", "--port", port); res.Code != 2 {
			t.Errorf("--port %s: exit = %d, want 2 (usage)", port, res.Code)
		}
	}
}

// No store means no server: the same not-found exit code every other command
// answers with, before anything binds.
func TestServeOutsideAStoreExitsNotFound(t *testing.T) {
	h := beavertest.New(t) // deliberately not initialized

	res := h.Run("serve")
	if res.Code != 3 {
		t.Errorf("exit = %d, want 3 (not found)", res.Code)
	}
	if !strings.Contains(res.Stderr, "beaver init") {
		t.Errorf("stderr does not say how to fix it: %s", res.Stderr)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q, want nothing printed when no server starts", res.Stdout)
	}
}

// The happy path: bind loopback, print the URL, and return cleanly once the
// process is asked to stop.
func TestServePrintsItsURLAndShutsDownOnInterrupt(t *testing.T) {
	h := beavertest.New(t).Init()
	interrupted, stop := context.WithCancel(context.Background())
	stop()
	h.Ctx = interrupted

	res := h.Run("serve", "--port", "0")

	if res.Code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", res.Code, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "http://127.0.0.1:") {
		t.Errorf("stdout does not print a loopback URL: %q", res.Stdout)
	}
}
