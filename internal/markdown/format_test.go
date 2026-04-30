package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSerializeRoundTrip(t *testing.T) {
	src := "# Hello world\n" +
		"\n" +
		"## Properties\n" +
		"Type\n" +
		":   Feature\n" +
		"State\n" +
		":   Proposed\n" +
		"Labels\n" +
		":   one\n" +
		":   two\n" +
		"\n" +
		"## Summary\n" +
		"This is the summary.\n" +
		"\n" +
		"It has two paragraphs.\n" +
		"\n" +
		"## Notes\n" +
		"- a\n" +
		"- b\n"
	it, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if it.Title != "Hello world" {
		t.Fatalf("title = %q", it.Title)
	}
	if v, _ := it.Properties.Get("Type"); v != "Feature" {
		t.Fatalf("Type = %q", v)
	}
	if v, _ := it.Properties.Get("Labels"); v != "one\ntwo" {
		t.Fatalf("Labels = %q", v)
	}
	if it.Section("Summary").Body != "This is the summary.\n\nIt has two paragraphs." {
		t.Fatalf("Summary body = %q", it.Section("Summary").Body)
	}
	out, err := Serialize(it)
	if err != nil {
		t.Fatal(err)
	}
	// Re-parse to ensure stability.
	it2, err := Parse(out)
	if err != nil {
		t.Fatalf("re-parse: %v\n%s", err, out)
	}
	out2, err := Serialize(it2)
	if err != nil {
		t.Fatal(err)
	}
	if out != out2 {
		t.Fatalf("serialization not stable\nfirst:\n%s\nsecond:\n%s", out, out2)
	}
}

func TestGoldenFiles(t *testing.T) {
	dir := filepath.Join("testdata", "golden")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no golden files")
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			// Normalize Windows checkouts: golden files are LF-canonical.
			normalized := strings.ReplaceAll(string(b), "\r\n", "\n")
			it, err := Parse(normalized)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			out, err := Serialize(it)
			if err != nil {
				t.Fatal(err)
			}
			if out != normalized {
				t.Fatalf("not byte-stable\nwant:\n%q\ngot:\n%q", normalized, out)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"empty":              "",
		"no-h1":              "Just text\n",
		"no-properties":      "# Title\n\n## Summary\nbody\n",
		"content-before-h2":  "# Title\nstray text\n## Properties\nT\n:   v\n",
		"property-no-value":  "# Title\n\n## Properties\nT\n",
		"value-without-term": "# Title\n\n## Properties\n:   no term\n",
		"second-h1":          "# Title\n\n## Properties\nT\n:   v\n\n# Another\n",
		"duplicate-property": "# Title\n\n## Properties\nT\n:   a\nT\n:   b\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(src); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParsePreservesHeadingsInsideFences(t *testing.T) {
	src := "# Title\n" +
		"\n## Properties\n" +
		"Type\n:   Feature\n" +
		"\n## Summary\n" +
		"Example with a fenced block:\n\n" +
		"```markdown\n" +
		"# Not a real H1\n" +
		"## Not a section\n" +
		"```\n" +
		"\n## Notes\n" +
		"after\n"
	it, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if it.Section("Notes") == nil {
		t.Fatal("Notes section missing — fence boundary mishandled")
	}
	body := it.Section("Summary").Body
	if !strings.Contains(body, "# Not a real H1") || !strings.Contains(body, "## Not a section") {
		t.Fatalf("fence content lost: %q", body)
	}
	out, err := Serialize(it)
	if err != nil {
		t.Fatal(err)
	}
	if it2, err := Parse(out); err != nil {
		t.Fatalf("re-parse: %v", err)
	} else if it2.Section("Notes") == nil || it2.Section("Summary") == nil {
		t.Fatalf("round-trip lost sections")
	}
}

func TestSetSection(t *testing.T) {
	it := &Item{Title: "T", Properties: NewProperties()}
	it.Properties.Set("Type", "Feature")
	it.SetSection("Summary", "hello")
	it.SetSection("Notes", "n1")
	it.SetSection("Summary", "updated")
	if it.Section("Summary").Body != "updated" {
		t.Fatal("update failed")
	}
	it.SetSection("Notes", "")
	if it.Section("Notes") != nil {
		t.Fatal("delete failed")
	}
}

func TestParseAcceptsCRLF(t *testing.T) {
	src := "# Title\r\n\r\n## Properties\r\nT\r\n:   v\r\n"
	if _, err := Parse(src); err != nil {
		t.Fatal(err)
	}
}
