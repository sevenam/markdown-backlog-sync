// Package markdown parses and serializes backlog item files.
//
// Files are pure Markdown — no YAML/TOML front-matter. The first non-empty
// line MUST be an H1 with the item title. A single "## Properties" section
// MUST appear before any other H2; properties are encoded as "Key: Value"
// lines (one per line). Repeat the key for multi-value properties:
//
//	Labels: bug
//	Labels: priority/high
//
// The legacy definition-list format (term on its own line, value on the
// next line prefixed with ":   ") is still accepted for backward
// compatibility but is not produced by Serialize.
package markdown

import (
	"bufio"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Item is the in-memory representation of a backlog item file.
type Item struct {
	Title      string
	Properties Properties
	// Sections holds H2 sections (other than Properties) in document
	// order. The body of each section is the raw text between the H2 line
	// (exclusive) and the next H2 (exclusive), with trailing newlines
	// trimmed.
	Sections []Section
}

// Section is a top-level H2 section in an item file.
type Section struct {
	Heading string // text after "## "
	Body    string // raw markdown, no trailing newlines
}

// Properties is an ordered, case-sensitive collection of definition-list
// entries. Insertion order is preserved for deterministic round-trip.
type Properties struct {
	keys   []string
	values map[string]string
}

// NewProperties returns an empty Properties.
func NewProperties() Properties {
	return Properties{values: map[string]string{}}
}

// Set adds or updates a property.
func (p *Properties) Set(key, value string) {
	if p.values == nil {
		p.values = map[string]string{}
	}
	if _, ok := p.values[key]; !ok {
		p.keys = append(p.keys, key)
	}
	p.values[key] = value
}

// add is like Set but errors on duplicate keys; used during parsing to
// surface authoring mistakes.
func (p *Properties) add(key, value string) error {
	if p.values == nil {
		p.values = map[string]string{}
	}
	if _, ok := p.values[key]; ok {
		return fmt.Errorf("duplicate property %q", key)
	}
	p.keys = append(p.keys, key)
	p.values[key] = value
	return nil
}

// Get returns the value and whether the key exists.
func (p *Properties) Get(key string) (string, bool) {
	v, ok := p.values[key]
	return v, ok
}

// Keys returns property keys in insertion order.
func (p *Properties) Keys() []string {
	out := make([]string, len(p.keys))
	copy(out, p.keys)
	return out
}

// Len returns the number of properties.
func (p *Properties) Len() int { return len(p.keys) }

// Section returns the named section, or nil.
func (it *Item) Section(name string) *Section {
	for i, s := range it.Sections {
		if strings.EqualFold(s.Heading, name) {
			return &it.Sections[i]
		}
	}
	return nil
}

// SetSection upserts a section. If body is empty the section is removed.
// New sections are appended at the end in the order they are first added.
func (it *Item) SetSection(name, body string) {
	body = strings.TrimRight(body, "\n")
	for i, s := range it.Sections {
		if strings.EqualFold(s.Heading, name) {
			if body == "" {
				it.Sections = append(it.Sections[:i], it.Sections[i+1:]...)
				return
			}
			it.Sections[i].Body = body
			return
		}
	}
	if body == "" {
		return
	}
	it.Sections = append(it.Sections, Section{Heading: name, Body: body})
}

// ParseError is returned for malformed item files; it carries a 1-based
// line number to make errors actionable.
type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
	}
	return e.Msg
}

func parseErrf(line int, format string, a ...any) *ParseError {
	return &ParseError{Line: line, Msg: fmt.Sprintf(format, a...)}
}

// Parse reads an item file from src.
func Parse(src string) (*Item, error) {
	// Normalize line endings; a final newline is optional.
	// Strip a leading UTF-8 BOM (EF BB BF) if present — editors on Windows
	// sometimes write one and it would confuse the H1 detection.
	src = strings.TrimPrefix(src, "\xef\xbb\xbf")
	src = strings.ReplaceAll(src, "\r\n", "\n")
	scanner := bufio.NewScanner(strings.NewReader(src))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	type line struct {
		n int
		s string
	}
	var lines []line
	for i := 1; scanner.Scan(); i++ {
		lines = append(lines, line{n: i, s: scanner.Text()})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Find first non-empty line — must be H1.
	idx := 0
	for idx < len(lines) && strings.TrimSpace(lines[idx].s) == "" {
		idx++
	}
	if idx >= len(lines) {
		return nil, &ParseError{Line: 0, Msg: "empty file"}
	}
	first := lines[idx]
	if !strings.HasPrefix(first.s, "# ") {
		return nil, parseErrf(first.n, "first non-empty line must be an H1 (\"# Title\")")
	}
	it := &Item{Title: strings.TrimSpace(strings.TrimPrefix(first.s, "# ")), Properties: NewProperties()}
	idx++

	// Collect H2 sections.
	type rawSection struct {
		heading string
		startN  int
		body    []string
	}
	var sections []rawSection
	var cur *rawSection
	inFence := false
	var fenceMarker string
	for ; idx < len(lines); idx++ {
		l := lines[idx]
		// Detect entering/leaving a fenced code block. We track the
		// opening marker (``` or ~~~ with optional info string) and only
		// treat a matching closing marker as the end. Inside fences,
		// heading-like lines are body content, not section boundaries.
		if isFenceLine(l.s) {
			if !inFence {
				inFence = true
				fenceMarker = fenceMarkerOf(l.s)
			} else if fenceMarkerOf(l.s) == fenceMarker {
				inFence = false
				fenceMarker = ""
			}
			if cur != nil {
				cur.body = append(cur.body, l.s)
			}
			continue
		}
		if !inFence && strings.HasPrefix(l.s, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(l.s, "## "))
			sections = append(sections, rawSection{heading: heading, startN: l.n})
			cur = &sections[len(sections)-1]
			continue
		}
		if !inFence && strings.HasPrefix(l.s, "# ") {
			return nil, parseErrf(l.n, "unexpected H1; only one H1 allowed per item")
		}
		if cur == nil {
			// Content between title and first H2 is not allowed.
			if strings.TrimSpace(l.s) == "" {
				continue
			}
			return nil, parseErrf(l.n, "content before first H2 is not allowed (expected \"## Properties\")")
		}
		cur.body = append(cur.body, l.s)
	}

	if len(sections) == 0 || !strings.EqualFold(sections[0].heading, "Properties") {
		return nil, parseErrf(first.n, "missing required \"## Properties\" section as the first H2")
	}

	// Parse the Properties section.
	props, err := parseProperties(sections[0].body, sections[0].startN+1)
	if err != nil {
		return nil, err
	}
	it.Properties = props

	// Remaining sections are kept verbatim, trailing blank lines trimmed.
	for _, s := range sections[1:] {
		body := strings.Join(s.body, "\n")
		body = strings.TrimRight(body, "\n")
		it.Sections = append(it.Sections, Section{Heading: s.heading, Body: body})
	}
	return it, nil
}

func parseProperties(body []string, startLine int) (Properties, error) {
	p := NewProperties()
	i := 0
	for i < len(body) {
		raw := body[i]
		ln := startLine + i
		// Skip blank lines.
		if strings.TrimSpace(raw) == "" {
			i++
			continue
		}

		// New format: "Key: Value" on a single line.
		// The key is everything before the first ": "; it must be non-empty
		// and the line must not start with ":" (which would be a legacy value
		// continuation).
		if colonIdx := strings.Index(raw, ": "); colonIdx > 0 && !strings.HasPrefix(raw, ":") {
			key := strings.TrimSpace(raw[:colonIdx])
			value := strings.TrimSpace(raw[colonIdx+2:])
			if key == "" {
				return p, parseErrf(ln, "empty property key")
			}
			// Repeated keys accumulate into a multi-value property (newline-
			// separated internally, matching the legacy multi-value convention).
			if existing, ok := p.values[key]; ok {
				p.values[key] = existing + "\n" + value
			} else {
				if err := p.add(key, value); err != nil {
					return p, parseErrf(ln, "%v", err)
				}
			}
			i++
			continue
		}

		// Legacy definition-list format: term on its own line, followed by
		// one or more ":   Value" lines. Kept for backward compatibility.
		if strings.HasPrefix(raw, ":") {
			return p, parseErrf(ln, "unexpected definition value without a preceding term")
		}
		term := strings.TrimSpace(raw)
		if term == "" {
			return p, parseErrf(ln, "empty property term")
		}
		i++
		var values []string
		sawValue := false
		for i < len(body) {
			next := body[i]
			if strings.HasPrefix(next, ":") {
				// Strip leading ":" and exactly one space (canonical form
				// uses ":   " with three spaces; accept any whitespace).
				v := strings.TrimPrefix(next, ":")
				v = strings.TrimLeft(v, " \t")
				values = append(values, v)
				sawValue = true
				i++
				continue
			}
			if strings.TrimSpace(next) == "" {
				// Blank line ends a single definition.
				i++
				break
			}
			break
		}
		if !sawValue {
			return p, parseErrf(ln, "property %q has no value (expected \"Key: Value\" format or \":   value\" on the next line)", term)
		}
		if err := p.add(term, strings.Join(values, "\n")); err != nil {
			return p, parseErrf(ln, "%v", err)
		}
	}
	if p.Len() == 0 {
		return p, parseErrf(startLine, "Properties section must define at least one property")
	}
	return p, nil
}

// Serialize renders an Item to canonical Markdown form.
func Serialize(it *Item) (string, error) {
	if it == nil {
		return "", errors.New("nil item")
	}
	if strings.TrimSpace(it.Title) == "" {
		return "", errors.New("item title is required")
	}
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(it.Title)
	b.WriteString("\n\n## Properties\n")
	for _, k := range it.Properties.keys {
		v := it.Properties.values[k]
		// Multi-value properties are stored as newline-separated strings;
		// emit one "Key: Value" line per value.
		lines := strings.Split(v, "\n")
		for _, ln := range lines {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(ln)
			b.WriteString("\n")
		}
	}
	for _, s := range it.Sections {
		b.WriteString("\n## ")
		b.WriteString(s.Heading)
		b.WriteString("\n")
		if s.Body != "" {
			b.WriteString(s.Body)
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

// isFenceLine reports whether s opens or closes a fenced code block. We
// recognize a line whose first non-space run is at least three backticks
// or three tildes; CommonMark allows up to three leading spaces of
// indentation before a fence.
func isFenceLine(s string) bool {
	t := strings.TrimLeft(s, " ")
	if len(s)-len(t) > 3 {
		return false
	}
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

// fenceMarkerOf returns the fence marker rune (``` or ~~~) for a fence
// line. Used to ensure the closing fence matches the opening one.
func fenceMarkerOf(s string) string {
	t := strings.TrimLeft(s, " ")
	if strings.HasPrefix(t, "```") {
		return "```"
	}
	if strings.HasPrefix(t, "~~~") {
		return "~~~"
	}
	return ""
}

// CanonicalKeys returns Properties keys in insertion order; convenience
// helper for callers that want to enforce a specific order.
func (it *Item) CanonicalKeys() []string {
	keys := it.Properties.Keys()
	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
