package issue

import "strings"

// slugMaxLen caps a slug so filenames stay reasonable; longer titles are cut on
// a word boundary. The ID, not the slug, is identity, so a truncated or stale
// slug never affects correctness.
const slugMaxLen = 60

// Slug derives a human-readable, filename-safe label from a title: lowercase,
// with each run of non-alphanumeric ASCII collapsed to a single hyphen. A title
// with no ASCII alphanumerics (e.g. only punctuation or non-Latin script) yields
// the empty string, in which case the file is named for the ID alone.
func Slug(title string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingHyphen = false
			b.WriteRune(r)
		default:
			// Defer emitting the separator so leading/trailing/repeated
			// non-alphanumerics never produce edge or doubled hyphens.
			pendingHyphen = true
		}
	}
	return capSlug(b.String())
}

func capSlug(s string) string {
	if len(s) <= slugMaxLen {
		return s
	}
	s = s[:slugMaxLen]
	if i := strings.LastIndexByte(s, '-'); i > 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, "-")
}

// FileName returns the canonical file name for an issue: "<id>-<slug>.md", or
// "<id>.md" when the slug is empty.
func FileName(id, slug string) string {
	if slug == "" {
		return id + ".md"
	}
	return id + "-" + slug + ".md"
}

// IDFromFileName extracts the ID portion of a canonical issue file name — the
// text before the first hyphen, or the whole stem when there is none. This reads
// the filename's idea of the ID; the frontmatter remains authoritative.
func IDFromFileName(name string) string {
	name = strings.TrimSuffix(name, ".md")
	id, _, _ := strings.Cut(name, "-")
	return id
}
