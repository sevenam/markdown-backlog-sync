// Package github implements the provider.Provider interface against the
// GitHub Issues REST API using google/go-github.
//
// Supported configuration keys (in backlog.config.toml [[provider]] table):
//
//	repo  – "owner/repo" (required)
//
// Authentication is resolved via the standard auth.Resolver precedence:
//  1. MBS_TOKEN_<UPPERSNAKE(name)> env var
//  2. GITHUB_TOKEN env var
//  3. OS keyring (populated by `mbs auth login`)
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gogithub "github.com/google/go-github/v68/github"
	"github.com/sevenam/markdown-backlog-sync/internal/provider"
)

// Kind is the provider type string used in backlog.config.toml.
const Kind = "github-issues"

// Provider implements provider.Provider against the GitHub REST API.
type Provider struct {
	name   string
	client *gogithub.Client
	owner  string
	repo   string
}

// Register adds a factory for Kind to the given registry. Call this once
// at startup (e.g. from main).
func Register(r *provider.Registry) {
	r.Register(Kind, New)
}

// New is the Factory function for the github-issues provider.
func New(name string, options map[string]any, creds provider.CredentialResolver) (provider.Provider, error) {
	repo, _ := options["repo"].(string)
	if repo == "" {
		return nil, fmt.Errorf("%w: github-issues provider %q: \"repo\" option is required (owner/repo)", provider.ErrValidation, name)
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("%w: github-issues provider %q: repo must be \"owner/repo\", got %q", provider.ErrValidation, name, repo)
	}

	token, err := creds.Resolve(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", provider.ErrAuth, err)
	}

	httpClient := &http.Client{
		Transport: &tokenTransport{token: token},
	}
	client := gogithub.NewClient(httpClient)

	return &Provider{
		name:   name,
		client: client,
		owner:  parts[0],
		repo:   parts[1],
	}, nil
}

// NewWithClient constructs a Provider using a caller-supplied go-github
// client. This is intended for testing (inject an httptest-backed client)
// and for advanced callers that manage their own HTTP middleware.
func NewWithClient(name string, client *gogithub.Client, owner, repo string) *Provider {
	return &Provider{name: name, client: client, owner: owner, repo: repo}
}

func (p *Provider) Name() string { return p.name }
func (p *Provider) Kind() string { return Kind }
func (p *Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		Labels:      true,
		Milestones:  true,
		Comments:    false, // v1 – issues body only
		ParentLinks: false, // future: sub-issues
		HardDelete:  false, // GitHub issues cannot be deleted via API; close instead
		DeltaSince:  true,
	}
}

// List returns all non-PR issues, optionally filtered to those updated after
// since. Pages through all results honoring rate limits.
func (p *Provider) List(ctx context.Context, since *time.Time) ([]provider.RemoteItem, error) {
	opts := &gogithub.IssueListByRepoOptions{
		State: "all",
		ListOptions: gogithub.ListOptions{
			PerPage: 100,
		},
	}
	if since != nil {
		opts.Since = *since
	}

	var items []provider.RemoteItem
	for {
		issues, resp, err := p.client.Issues.ListByRepo(ctx, p.owner, p.repo, opts)
		if err != nil {
			return nil, wrapErr(err)
		}

		for _, iss := range issues {
			// Skip pull requests — the Issues API returns them too.
			if iss.PullRequestLinks != nil {
				continue
			}
			items = append(items, issueToRemoteItem(iss))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return items, nil
}

// Get fetches a single issue by its remote ID (issue number as string).
func (p *Provider) Get(ctx context.Context, remoteID string) (provider.RemoteItem, error) {
	num, err := strconv.Atoi(remoteID)
	if err != nil {
		return provider.RemoteItem{}, fmt.Errorf("%w: invalid issue number %q", provider.ErrValidation, remoteID)
	}
	iss, _, err := p.client.Issues.Get(ctx, p.owner, p.repo, num)
	if err != nil {
		return provider.RemoteItem{}, wrapErr(err)
	}
	if iss.PullRequestLinks != nil {
		return provider.RemoteItem{}, fmt.Errorf("%w: %s is a pull request", provider.ErrItemNotFound, remoteID)
	}
	return issueToRemoteItem(iss), nil
}

// Create opens a new GitHub issue.
func (p *Provider) Create(ctx context.Context, item provider.RemoteItem) (provider.RemoteItem, error) {
	req := &gogithub.IssueRequest{
		Title:     gogithub.Ptr(item.Title),
		Body:      gogithub.Ptr(item.Body),
		Assignees: &item.Assignees,
	}
	if len(item.Labels) > 0 {
		req.Labels = &item.Labels
	}
	if item.Milestone != "" {
		ms, err := p.resolveMilestone(ctx, item.Milestone)
		if err != nil {
			return provider.RemoteItem{}, err
		}
		if ms != nil {
			req.Milestone = ms.Number
		}
	}

	iss, _, err := p.client.Issues.Create(ctx, p.owner, p.repo, req)
	if err != nil {
		return provider.RemoteItem{}, wrapErr(err)
	}
	return issueToRemoteItem(iss), nil
}

// Update applies a sparse patch to an existing GitHub issue.
// baseRev is the UpdatedAt timestamp observed by the caller; if the issue
// has been modified remotely since then, ErrConflict is returned.
func (p *Provider) Update(ctx context.Context, remoteID string, patch provider.ItemPatch, baseRev string) (provider.RemoteItem, error) {
	num, err := strconv.Atoi(remoteID)
	if err != nil {
		return provider.RemoteItem{}, fmt.Errorf("%w: invalid issue number %q", provider.ErrValidation, remoteID)
	}

	// Optimistic concurrency: compare updated_at.
	if baseRev != "" {
		cur, _, err := p.client.Issues.Get(ctx, p.owner, p.repo, num)
		if err != nil {
			return provider.RemoteItem{}, wrapErr(err)
		}
		if revOf(cur) != baseRev {
			return provider.RemoteItem{}, fmt.Errorf("%w: issue #%s was modified remotely", provider.ErrConflict, remoteID)
		}
	}

	req := &gogithub.IssueRequest{}
	if patch.Title != nil {
		req.Title = patch.Title
	}
	if patch.Body != nil {
		req.Body = patch.Body
	}
	if patch.State != nil {
		req.State = patch.State
		// GitHub requires state_reason when closing as not-planned.
		if v, ok := patchProperty(patch, "state_reason"); ok {
			req.StateReason = gogithub.Ptr(v)
		}
	}
	if patch.Assignees != nil {
		req.Assignees = patch.Assignees
	}
	if patch.Labels != nil {
		req.Labels = patch.Labels
	}
	if patch.Milestone != nil {
		ms, err := p.resolveMilestone(ctx, *patch.Milestone)
		if err != nil {
			return provider.RemoteItem{}, err
		}
		if ms != nil {
			req.Milestone = ms.Number
		} else {
			// clear milestone: GitHub uses 0
			zero := 0
			req.Milestone = &zero
		}
	}

	iss, _, err := p.client.Issues.Edit(ctx, p.owner, p.repo, num, req)
	if err != nil {
		return provider.RemoteItem{}, wrapErr(err)
	}
	return issueToRemoteItem(iss), nil
}

// Delete closes the issue (GitHub issues cannot be deleted via the API).
// If baseRev is non-empty it performs optimistic concurrency checking first.
func (p *Provider) Delete(ctx context.Context, remoteID string, baseRev string) error {
	closed := "closed"
	_, err := p.Update(ctx, remoteID, provider.ItemPatch{State: &closed}, baseRev)
	return err
}

// resolveMilestone looks up a milestone by title. Returns nil if title is
// empty, returns an error if the milestone is not found.
func (p *Provider) resolveMilestone(ctx context.Context, title string) (*gogithub.Milestone, error) {
	if title == "" {
		return nil, nil
	}
	opts := &gogithub.MilestoneListOptions{
		State:       "all",
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}
	for {
		ms, resp, err := p.client.Issues.ListMilestones(ctx, p.owner, p.repo, opts)
		if err != nil {
			return nil, wrapErr(err)
		}
		for _, m := range ms {
			if strings.EqualFold(m.GetTitle(), title) {
				return m, nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return nil, fmt.Errorf("%w: milestone %q not found in %s/%s", provider.ErrValidation, title, p.owner, p.repo)
}

// patchProperty reads an entry from patch.Properties (nil-safe).
func patchProperty(patch provider.ItemPatch, key string) (string, bool) {
	if patch.Properties == nil {
		return "", false
	}
	v, ok := patch.Properties[key]
	return v, ok
}

// tokenTransport injects a Bearer token on every request.
type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(r)
}

// wrapErr maps go-github errors to provider sentinel errors so the sync
// engine and exit-code logic can act on them without importing go-github.
func wrapErr(err error) error {
	if err == nil {
		return nil
	}

	// go-github rate limit error.
	var rl *gogithub.RateLimitError
	if errors.As(err, &rl) {
		var wait time.Duration
		if rl.Rate.Reset.After(time.Now()) {
			wait = time.Until(rl.Rate.Reset.Time)
		}
		return &provider.RateLimitError{
			RetryAfter: wait,
			Err:        fmt.Errorf("%w: %v", provider.ErrRateLimited, err),
		}
	}

	// go-github secondary (abuse) rate limit error.
	var abuse *gogithub.AbuseRateLimitError
	if errors.As(err, &abuse) {
		var wait time.Duration
		if abuse.RetryAfter != nil {
			wait = *abuse.RetryAfter
		}
		return &provider.RateLimitError{
			RetryAfter: wait,
			Err:        fmt.Errorf("%w: secondary rate limit: %v", provider.ErrRateLimited, err),
		}
	}

	// HTTP error responses.
	var ghErr *gogithub.ErrorResponse
	if errors.As(err, &ghErr) {
		switch ghErr.Response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("%w: %v", provider.ErrAuth, err)
		case http.StatusNotFound:
			return fmt.Errorf("%w: %v", provider.ErrItemNotFound, err)
		case http.StatusUnprocessableEntity:
			return fmt.Errorf("%w: %v", provider.ErrValidation, err)
		case http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return fmt.Errorf("%w: %v", provider.ErrTransient, err)
		}
	}

	return err
}
