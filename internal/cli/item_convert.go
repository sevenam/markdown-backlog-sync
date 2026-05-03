package cli

import (
	"strings"
	"unicode"

	"github.com/sevenam/markdown-backlog-sync/internal/markdown"
	"github.com/sevenam/markdown-backlog-sync/internal/provider"
)

// remoteItemToMarkdown converts a provider-neutral RemoteItem into the
// markdown.Item representation ready for serialization.
func remoteItemToMarkdown(item provider.RemoteItem) *markdown.Item {
	it := &markdown.Item{
		Title:      item.Title,
		Properties: markdown.NewProperties(),
	}

	// Core typed properties — order matches the canonical format.
	if item.Type != "" {
		it.Properties.Set("Type", item.Type)
	}
	if item.State != "" {
		it.Properties.Set("State", item.State)
	}
	if len(item.Assignees) > 0 {
		it.Properties.Set("Assignees", strings.Join(item.Assignees, "\n"))
	}
	if len(item.Labels) > 0 {
		it.Properties.Set("Labels", strings.Join(item.Labels, "\n"))
	}
	if item.Milestone != "" {
		it.Properties.Set("Milestone", item.Milestone)
	}
	if item.Iteration != "" {
		it.Properties.Set("Iteration", item.Iteration)
	}
	if item.AreaPath != "" {
		it.Properties.Set("AreaPath", item.AreaPath)
	}
	if item.Parent != "" {
		it.Properties.Set("Parent", item.Parent)
	}

	// Remote tracking fields — always written so re-pulls are idempotent.
	it.Properties.Set("RemoteId", item.ID)
	it.Properties.Set("Url", item.URL)

	// Overflow / provider-specific properties.
	for k, v := range item.Properties {
		it.Properties.Set(k, v)
	}

	// Item body becomes a Description section.
	if item.Body != "" {
		it.SetSection("Description", item.Body)
	}

	return it
}

// safeLocalID returns a state-store-safe local ID for a (providerName,
// remoteID) pair. It replaces characters outside [A-Za-z0-9._-] with '_'
// and trims to 128 characters.
func safeLocalID(providerName, remoteID string) string {
	raw := providerName + "-" + remoteID
	var b strings.Builder
	for i, r := range raw {
		if i == 0 && !isAlphaNum(r) {
			b.WriteRune('i')
		}
		if isAlphaNum(r) || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	s := b.String()
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

// safeFilename produces a filesystem-safe filename: "<id>-<slug>.md".
// The slug is derived by lower-casing the title and replacing non-alphanumeric
// runs with a single hyphen.
func safeFilename(id, title string) string {
	slug := slugify(title)
	if slug == "" {
		return id + ".md"
	}
	return id + "-" + slug + ".md"
}

// slugify converts a string to a URL/filename-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevHyphen := true // suppress leading hyphen
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteRune('-')
			prevHyphen = true
		}
	}
	// Trim trailing hyphen.
	out := strings.TrimRight(b.String(), "-")
	// Cap length to keep filenames reasonable.
	const maxSlug = 60
	if len(out) > maxSlug {
		out = out[:maxSlug]
		out = strings.TrimRight(out, "-")
	}
	return out
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
