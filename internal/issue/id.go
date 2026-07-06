package issue

import "crypto/rand"

// An ID is six random lowercase-alphanumeric characters (e.g. "a1b2c3").
// Lowercase keeps an ID safe to embed in a filename on case-insensitive
// filesystems.
const (
	idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	idLength   = 6
)

// NewID returns a fresh random ID. Collision checking is the caller's job:
// generate, check the store, regenerate on the rare clash.
func NewID() string {
	b := make([]byte, idLength)
	for i := range b {
		b[i] = pick(idAlphabet)
	}
	return string(b)
}

// pick returns a uniformly random byte from set, using rejection sampling to
// avoid the modulo bias a naive `rand % len` would introduce.
func pick(set string) byte {
	limit := 256 - (256 % len(set))
	var buf [1]byte
	for {
		if _, err := rand.Read(buf[:]); err != nil {
			// A failure here means the system entropy source is broken,
			// which we cannot paper over.
			panic("issue: cannot read random bytes: " + err.Error())
		}
		if int(buf[0]) < limit {
			return set[int(buf[0])%len(set)]
		}
	}
}
