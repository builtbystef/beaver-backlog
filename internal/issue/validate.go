package issue

import (
	"errors"
	"fmt"
)

// Validate reports whether iss is a usable issue — the hard-error half of the
// store's integrity contract (ADR 0005). A valid issue has a present, well-formed
// id and a legal state. A file that fails either is not a usable issue: read
// paths skip it (with a loud warning that names it) and keep serving the valid
// issues, rather than failing the whole command on one bad file.
//
// Validation is deliberately narrow. Everything a *valid* issue can still get
// wrong — a filename that has drifted from its id, a slug that no longer matches
// the title, unknown frontmatter keys (preserved verbatim, ADR 0014),
// non-canonical formatting — is lint, not a validation failure: the issue still
// loads, and doctor reports and tidies it (n9b4a7). Keeping the two apart is the
// whole point of the contract — a broken file is skipped, an untidy one is used.
//
// The returned error states the specific defect (naming the offending value), so
// a caller can pair it with the file name for an actionable message. Frontmatter
// that does not parse at all is caught earlier, by Unmarshal.
func Validate(iss Issue) error {
	switch {
	case iss.ID == "":
		return errors.New("missing id")
	case !validID(iss.ID):
		return fmt.Errorf("malformed id %q (want lowercase letters and digits)", iss.ID)
	case iss.State == "":
		return errors.New("missing state")
	case !iss.State.Valid():
		return fmt.Errorf("invalid state %q (want one of: todo, in-progress, done, cancelled)", iss.State)
	}
	return nil
}

// validID reports whether s is a well-formed issue id: a non-empty run of the id
// alphabet — lowercase ASCII letters and digits (see NewID). It deliberately does
// not pin the length. The id length is tunable until 1.0, and a store may still
// hold shorter ids minted by an earlier version, so validity turns on the
// character set (what keeps an id safe in a filename) and not on a length that is
// only ever a generation-time policy.
func validID(s string) bool {
	for _, r := range s {
		lower := r >= 'a' && r <= 'z'
		digit := r >= '0' && r <= '9'
		if !lower && !digit {
			return false
		}
	}
	return s != ""
}
