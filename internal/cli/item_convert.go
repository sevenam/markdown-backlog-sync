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

// markdownToRemoteItem converts a parsed markdown Item to a provider RemoteItem
// suitable for Create calls. Tracking-only fields (RemoteId, Url) are ignored.
func markdownToRemoteItem(it *markdown.Item) provider.RemoteItem {
	ri := provider.RemoteItem{Title: it.Title}
	if v, ok := it.Properties.Get("Type"); ok {
		ri.Type = v
	}
	if v, ok := it.Properties.Get("State"); ok {
		ri.State = v
	}
	if v, ok := it.Properties.Get("Assignees"); ok {
		ri.Assignees = splitLines(v)
	}
	if v, ok := it.Properties.Get("Labels"); ok {
		ri.Labels = splitLines(v)
	}
	if v, ok := it.Properties.Get("Milestone"); ok {
		ri.Milestone = v
	}
	if v, ok := it.Properties.Get("Iteration"); ok {
		ri.Iteration = v
	}
	if v, ok := it.Properties.Get("AreaPath"); ok {
		ri.AreaPath = v
	}
	if v, ok := it.Properties.Get("Parent"); ok {
		ri.Parent = v
	}
	if sec := it.Section("Description"); sec != nil {
		ri.Body = sec.Body
	}
	knownFields := knownCoreFields()
	for _, k := range it.Properties.Keys() {
		if !knownFields[k] {
			v, _ := it.Properties.Get(k)
			if ri.Properties == nil {
				ri.Properties = map[string]string{}
			}
			ri.Properties[k] = v
		}
	}
	return ri
}

// markdownToItemPatch converts a parsed markdown Item to an ItemPatch for
// Update calls. All present fields are included in the patch.
func markdownToItemPatch(it *markdown.Item) provider.ItemPatch {
	patch := provider.ItemPatch{}

	title := it.Title
	patch.Title = &title

	if v, ok := it.Properties.Get("State"); ok {
		s := v
		patch.State = &s
	}
	if v, ok := it.Properties.Get("Type"); ok {
		t := v
		patch.Type = &t
	}
	if v, ok := it.Properties.Get("Assignees"); ok {
		a := splitLines(v)
		patch.Assignees = &a
	}
	if v, ok := it.Properties.Get("Labels"); ok {
		l := splitLines(v)
		patch.Labels = &l
	}
	if v, ok := it.Properties.Get("Milestone"); ok {
		m := v
		patch.Milestone = &m
	}
	if v, ok := it.Properties.Get("Iteration"); ok {
		i := v
		patch.Iteration = &i
	}
	if v, ok := it.Properties.Get("AreaPath"); ok {
		ap := v
		patch.AreaPath = &ap
	}
	if v, ok := it.Properties.Get("Parent"); ok {
		p := v
		patch.Parent = &p
	}
	if sec := it.Section("Description"); sec != nil {
		body := sec.Body
		patch.Body = &body
	} else {
		empty := ""
		patch.Body = &empty
	}
	knownFields := knownCoreFields()
	for _, k := range it.Properties.Keys() {
		if !knownFields[k] {
			v, _ := it.Properties.Get(k)
			if patch.Properties == nil {
				patch.Properties = map[string]string{}
			}
			patch.Properties[k] = v
		}
	}
	return patch
}

// knownCoreFields returns the set of property keys that are handled as typed
// fields and should not spill into the overflow Properties map.
func knownCoreFields() map[string]bool {
	return map[string]bool{
		"Type": true, "State": true, "Assignees": true, "Labels": true,
		"Milestone": true, "Iteration": true, "AreaPath": true, "Parent": true,
		"RemoteId": true, "Url": true,
	}
}

// splitLines splits a newline-delimited property value into non-empty trimmed lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
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
