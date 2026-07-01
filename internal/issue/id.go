package issue

import "crypto/rand"

// IDs are short, random, and collision-resistant — deliberately not sequential
// (ADR 0002). The exact alphabet and length are implementation details, left
// tunable until 1.0; today an ID is four random lowercase-alphanumeric
// characters (e.g. "a1b2"). Lowercase keeps an ID safe to embed in a filename on
// case-insensitive filesystems.
const (
	idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	idLength   = 4
)

// NewID returns a fresh random ID. Collision-resistance at the store level is
// the caller's job: generate, check the store, regenerate on the rare clash.
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
			// crypto/rand does not fail in practice; a failure here means the
			// system entropy source is broken, which we cannot paper over.
			panic("issue: cannot read random bytes: " + err.Error())
		}
		if int(buf[0]) < limit {
			return set[int(buf[0])%len(set)]
		}
	}
}
