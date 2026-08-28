package issue

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Errors returned by Unmarshal when a file is not a well-formed issue.
var (
	ErrMissingFrontmatter      = errors.New("missing YAML frontmatter (file must start with '---')")
	ErrUnterminatedFrontmatter = errors.New("unterminated YAML frontmatter (missing closing '---')")
)

// frontmatter is the YAML projection of an Issue; field order here is the
// canonical on-disk order. Optional fields carry omitempty so an unset field is
// absent from the file. Created/Updated also carry omitempty because a
// hand-authored file may lack them, and a zero time must stay absent on rewrite
// rather than serialize as the year-1 sentinel.
//
// Custom is an inline catch-all: any frontmatter key with no matching field
// lands here on read and is written back on save, so user-added fields survive
// a round-trip. Custom keys follow the known fields, sorted by the encoder.
type frontmatter struct {
	ID        string         `yaml:"id"`
	Title     string         `yaml:"title"`
	State     string         `yaml:"state"`
	Assignee  string         `yaml:"assignee,omitempty"`
	Priority  string         `yaml:"priority,omitempty"`
	Labels    []string       `yaml:"labels,omitempty"`
	DependsOn []string       `yaml:"depends_on,omitempty"`
	Parent    string         `yaml:"parent,omitempty"`
	Created   yamlTime       `yaml:"created,omitempty"`
	Updated   yamlTime       `yaml:"updated,omitempty"`
	Custom    map[string]any `yaml:",inline"`
}

// Marshal renders an Issue as its on-disk Markdown file: a YAML frontmatter
// block delimited by "---" fences, followed by the body.
func Marshal(iss Issue) ([]byte, error) {
	fm := frontmatter{
		ID:        iss.ID,
		Title:     iss.Title,
		State:     string(iss.State),
		Assignee:  iss.Assignee,
		Priority:  string(iss.Priority),
		Labels:    iss.Labels,
		DependsOn: iss.DependsOn,
		Parent:    iss.Parent,
		Created:   yamlTime{iss.Created},
		Updated:   yamlTime{iss.Updated},
		Custom:    iss.Custom,
	}
	y, err := yaml.Marshal(fm)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	b.WriteString("---\n")
	b.Write(y)
	b.WriteString("---\n")
	if iss.Body != "" {
		b.WriteByte('\n')
		b.WriteString(iss.Body)
		if !strings.HasSuffix(iss.Body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.Bytes(), nil
}

// Unmarshal parses an issue file back into an Issue. The body is preserved
// verbatim, including any "---" lines it contains.
func Unmarshal(data []byte) (Issue, error) {
	front, body, err := splitFrontmatter(data)
	if err != nil {
		return Issue{}, err
	}
	var fm frontmatter
	if err := yaml.Unmarshal(front, &fm); err != nil {
		return Issue{}, fmt.Errorf("invalid frontmatter: %w", err)
	}
	return Issue{
		ID:        fm.ID,
		Title:     fm.Title,
		State:     State(fm.State),
		Assignee:  fm.Assignee,
		Priority:  Priority(fm.Priority),
		Labels:    fm.Labels,
		DependsOn: fm.DependsOn,
		Parent:    fm.Parent,
		Created:   fm.Created.Time,
		Updated:   fm.Updated.Time,
		Body:      string(body),
		Custom:    fm.Custom,
	}, nil
}

// splitFrontmatter separates the YAML frontmatter from the body, dropping a
// single blank separator line. The frontmatter is the block between the opening
// "---" fence on the first line and the next "---" fence.
func splitFrontmatter(data []byte) (front, body []byte, err error) {
	s := bytes.TrimPrefix(data, []byte("\ufeff")) // drop a UTF-8 BOM if present

	open, next := readLine(s, 0)
	if !isFence(s[open.start:open.end]) {
		return nil, nil, ErrMissingFrontmatter
	}

	fmStart := next
	for pos := next; pos < len(s); {
		line, after := readLine(s, pos)
		if isFence(s[line.start:line.end]) {
			return s[fmStart:line.start], trimOneLeadingNewline(s[after:]), nil
		}
		pos = after
	}
	return nil, nil, ErrUnterminatedFrontmatter
}

type lineSpan struct{ start, end int }

// readLine returns the span of the line beginning at pos (excluding its newline)
// and the index at which the following line begins.
func readLine(s []byte, pos int) (lineSpan, int) {
	nl := bytes.IndexByte(s[pos:], '\n')
	if nl < 0 {
		return lineSpan{pos, len(s)}, len(s)
	}
	end := pos + nl
	return lineSpan{pos, end}, end + 1
}

func isFence(line []byte) bool {
	return string(bytes.TrimRight(line, "\r")) == "---"
}

func trimOneLeadingNewline(b []byte) []byte {
	switch {
	case len(b) >= 2 && b[0] == '\r' && b[1] == '\n':
		return b[2:]
	case len(b) >= 1 && b[0] == '\n':
		return b[1:]
	default:
		return b
	}
}

// yamlTime serializes a time.Time as a plain, unquoted RFC3339 timestamp in
// UTC, rather than the quoted form the default string encoding would produce.
type yamlTime struct{ time.Time }

func (t yamlTime) MarshalYAML() (any, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!timestamp",
		Value: t.UTC().Format(time.RFC3339),
	}, nil
}

func (t *yamlTime) UnmarshalYAML(node *yaml.Node) error {
	parsed, err := parseTime(node.Value)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

func parseTime(s string) (time.Time, error) {
	// The RFC3339 layout also accepts fractional seconds, so one parse covers
	// both the canonical form and a hand-written fractional timestamp.
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q (want RFC3339, e.g. 2006-01-02T15:04:05Z)", s)
	}
	return t.UTC(), nil
}
