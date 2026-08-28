package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Config is the committed, shared project configuration at .beaver/config.yml.
// It travels with the repository, so a setting here is project-wide policy.
// It is safe to edit by hand; unknown keys are ignored on read, so a config
// written by a newer version still loads in an older one.
type Config struct {
	// FormatVersion records the on-disk layout the store was created with.
	FormatVersion int `yaml:"format_version"`
	// Name is what the project is called, absent unless it chooses a name other
	// than its directory's. It is committed on purpose: the project's name for
	// everyone who clones it, not a personal alias (ADR 0004).
	Name string `yaml:"name,omitempty"`
}

// Config reads the project configuration from .beaver/config.yml. A missing
// file yields defaults (the shipped format version), not an error; a
// present-but-unparseable file is a real error. Unknown keys are tolerated for
// forward compatibility.
func (s *Store) Config() (Config, error) {
	cfg := Config{FormatVersion: FormatVersion}
	data, err := os.ReadFile(s.ConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", s.ConfigPath(), err)
	}
	return cfg, nil
}

// defaultConfig is the config file Init writes: the format version, the one
// project-wide setting there is.
func defaultConfig() []byte {
	return fmt.Appendf(nil,
		`# Beaver Backlog project configuration.
# Committed and shared through version control, like the issues themselves.
# Safe to read and edit by hand.
format_version: %d
`, FormatVersion)
}

// SetProjectName records name as the project's name in the committed config.
// It rewrites the file through the node tree rather than re-serializing a
// struct, so the comments and any keys this version does not know survive.
// An empty name is refused: no name means no key, not an empty one.
func (s *Store) SetProjectName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("a project name cannot be empty")
	}
	data, err := os.ReadFile(s.ConfigPath())
	switch {
	case errors.Is(err, os.ErrNotExist):
		data = defaultConfig() // a store whose config was never written, or removed
	case err != nil:
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("reading %s: %w", s.ConfigPath(), err)
	}
	out, err := marshalConfig(withMapping(&doc, "name", name))
	if err != nil {
		return err
	}
	return atomicWrite(s.ConfigPath(), out, 0o644)
}

// withMapping sets key to value in the document's top-level mapping, replacing
// an existing entry in place so the file keeps its order. A document with no
// mapping (an empty file) gains one.
func withMapping(doc *yaml.Node, key, value string) *yaml.Node {
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	mapping := doc.Content[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = scalar(value)
			return doc
		}
	}
	mapping.Content = append(mapping.Content, scalar(key), scalar(value))
	return doc
}

// scalar is one string node, tagged so a name that reads as a number or a
// boolean is still written and read back as a string.
func scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

// marshalConfig serializes the config document at the two-space indentation the
// file is written and hand-edited at.
func marshalConfig(doc *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
