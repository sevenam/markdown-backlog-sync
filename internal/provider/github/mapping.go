package github

import (
	"strconv"

	gogithub "github.com/google/go-github/v68/github"
	"github.com/sevenam/markdown-backlog-sync/internal/provider"
)

// issueToRemoteItem converts a GitHub Issue into a provider-neutral RemoteItem.
func issueToRemoteItem(iss *gogithub.Issue) provider.RemoteItem {
	item := provider.RemoteItem{
		ID:        strconv.Itoa(iss.GetNumber()),
		URL:       iss.GetHTMLURL(),
		Title:     iss.GetTitle(),
		Body:      iss.GetBody(),
		Type:      "issue",
		State:     iss.GetState(),
		UpdatedAt: iss.GetUpdatedAt().Time,
		Rev:       revOf(iss),
	}

	for _, a := range iss.Assignees {
		item.Assignees = append(item.Assignees, a.GetLogin())
	}

	for _, l := range iss.Labels {
		item.Labels = append(item.Labels, l.GetName())
	}

	if ms := iss.Milestone; ms != nil {
		item.Milestone = ms.GetTitle()
	}

	// Preserve state_reason for closed-as-not-planned vs completed.
	if sr := iss.GetStateReason(); sr != "" {
		item.Properties = map[string]string{"state_reason": sr}
	}

	return item
}

// revOf returns the opaque revision token for an issue. We use the
// UpdatedAt timestamp in RFC3339 nanosecond form so it is stable and
// comparable across calls.
func revOf(iss *gogithub.Issue) string {
	if iss == nil || iss.UpdatedAt == nil {
		return ""
	}
	return iss.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z")
}
