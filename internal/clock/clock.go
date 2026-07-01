// Package clock provides an injectable source of the current time.
//
// Every timestamp Beaver writes (an issue's created/updated fields) comes from a
// Clock rather than from time.Now directly, so tests can pin time to a fixed
// instant and assert on deterministic output.
package clock

import "time"

// Clock reports the current time.
type Clock interface {
	Now() time.Time
}

// Func adapts an ordinary function to a Clock.
type Func func() time.Time

// Now calls the underlying function.
func (f Func) Now() time.Time { return f() }

// System is the real clock, backed by time.Now.
func System() Clock { return Func(time.Now) }

// Fixed returns a Clock that always reports t. Useful as a simple deterministic
// clock in tests; for a clock that can be advanced, see beavertest.FakeClock.
func Fixed(t time.Time) Clock { return Func(func() time.Time { return t }) }
