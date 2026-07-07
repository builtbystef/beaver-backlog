//go:build !windows

package store

// retryFS runs a filesystem mutation once. Only the Windows build retries; the
// transient sharing and lock failures it guards against — a scanner, indexer,
// or sync client briefly holding a file open — have no POSIX equivalent.
func retryFS(fn func() error) error {
	return fn()
}
