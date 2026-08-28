package issue

import (
	"errors"
	"fmt"
)

// Validate reports whether iss is a usable issue: a present, well-formed id and
// a legal state. Validation is deliberately narrow: everything else a valid
// issue can get wrong (filename drift, unknown keys, non-canonical formatting)
// is lint for doctor, not a validation failure. The returned error names the
// specific defect so a caller can pair it with the file name; frontmatter that
// does not parse at all is caught earlier, by Unmarshal.
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

// validID reports whether s is a non-empty run of lowercase ASCII letters and
// digits. It deliberately does not pin the length: a store may hold ids of
// other lengths, so validity turns on the character set alone, which is what
// keeps an id safe in a filename.
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
