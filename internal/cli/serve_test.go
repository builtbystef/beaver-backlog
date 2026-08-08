package cli_test

import (
	"context"
	"net"
	"strconv"
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

// A busy default port is not an error: serve scans forward so a second
// project's serve coexists with the first. If our own bind of 2328 fails,
// something else already holds it — either way the port is taken, which is
// exactly the setup this test needs.
func TestServeScansForwardWhenTheDefaultPortIsTaken(t *testing.T) {
	h := beavertest.New(t).Init()
	if squatter, err := net.Listen("tcp", "127.0.0.1:2328"); err == nil {
		defer squatter.Close()
	}
	interrupted, stop := context.WithCancel(context.Background())
	stop()
	h.Ctx = interrupted

	res := h.Run("serve")

	if res.Code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", res.Code, res.Stderr)
	}
	if strings.Contains(res.Stdout, "http://127.0.0.1:2328") {
		t.Errorf("stdout claims the taken port 2328: %q", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "port 2328 is taken") {
		t.Errorf("stdout does not explain the fallback: %q", res.Stdout)
	}
}

// An explicit --port is a choice, not a starting point: when it is taken,
// serve fails rather than silently binding somewhere else.
func TestServeDoesNotScanFromAnExplicitPort(t *testing.T) {
	h := beavertest.New(t).Init()
	squatter, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind a port to occupy: %v", err)
	}
	defer squatter.Close()
	port := squatter.Addr().(*net.TCPAddr).Port

	res := h.Run("serve", "--port", strconv.Itoa(port))

	if res.Code != 1 {
		t.Errorf("exit = %d, want 1 (runtime failure)", res.Code)
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
