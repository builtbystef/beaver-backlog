//go:build windows

package store

import (
	"errors"
	"time"

	"golang.org/x/sys/windows"
)

// retryFS runs a filesystem mutation, retrying the transient sharing and lock
// failures Windows raises when a scanner, indexer, or sync client briefly holds
// the file open. The backoff is short and bounded (a blip, not a permanent
// hold); a deterministic failure like a missing file returns at once.
func retryFS(fn func() error) error {
	const attempts = 5
	delay := 10 * time.Millisecond
	err := fn()
	for i := 1; i < attempts && sharingFailure(err); i++ {
		time.Sleep(delay)
		delay *= 2
		err = fn()
	}
	return err
}

// sharingFailure reports whether err is a transient sharing or locking error a
// retry can clear.
func sharingFailure(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_ACCESS_DENIED) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION)
}
