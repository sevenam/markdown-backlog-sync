// Package mock provides an in-memory Provider used in tests.
package mock

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/sevenam/markdown-backlog-sync/internal/provider"
)

// Provider is an in-memory provider useful for tests and local dry-runs.
type Provider struct {
	name string

	mu     sync.Mutex
	nextID int
	items  map[string]provider.RemoteItem
}

// New returns a fresh in-memory provider.
func New(name string) *Provider {
	return &Provider{name: name, items: map[string]provider.RemoteItem{}}
}

func (p *Provider) Name() string { return p.name }
func (p *Provider) Kind() string { return "mock" }
func (p *Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		Labels:      true,
		ParentLinks: true,
		DeltaSince:  true,
	}
}

func (p *Provider) List(_ context.Context, since *time.Time) ([]provider.RemoteItem, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]provider.RemoteItem, 0, len(p.items))
	for _, it := range p.items {
		if since != nil && it.UpdatedAt.Before(*since) {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

func (p *Provider) Get(_ context.Context, id string) (provider.RemoteItem, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	it, ok := p.items[id]
	if !ok {
		return provider.RemoteItem{}, provider.ErrItemNotFound
	}
	return it, nil
}

func (p *Provider) Create(_ context.Context, in provider.RemoteItem) (provider.RemoteItem, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nextID++
	in.ID = strconv.Itoa(p.nextID)
	in.URL = "mock://" + p.name + "/" + in.ID
	in.UpdatedAt = time.Now().UTC()
	in.Rev = strconv.FormatInt(in.UpdatedAt.UnixNano(), 10)
	p.items[in.ID] = in
	return in, nil
}

func (p *Provider) Update(_ context.Context, id string, patch provider.ItemPatch, baseRev string) (provider.RemoteItem, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur, ok := p.items[id]
	if !ok {
		return provider.RemoteItem{}, provider.ErrItemNotFound
	}
	if baseRev != "" && baseRev != cur.Rev {
		return provider.RemoteItem{}, provider.ErrConflict
	}
	if patch.Title != nil {
		cur.Title = *patch.Title
	}
	if patch.Body != nil {
		cur.Body = *patch.Body
	}
	if patch.State != nil {
		cur.State = *patch.State
	}
	if patch.Labels != nil {
		cur.Labels = append([]string(nil), (*patch.Labels)...)
	}
	cur.UpdatedAt = time.Now().UTC()
	cur.Rev = strconv.FormatInt(cur.UpdatedAt.UnixNano(), 10)
	p.items[id] = cur
	return cur, nil
}

func (p *Provider) Delete(_ context.Context, id string, baseRev string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur, ok := p.items[id]
	if !ok {
		return provider.ErrItemNotFound
	}
	if baseRev != "" && baseRev != cur.Rev {
		return provider.ErrConflict
	}
	delete(p.items, id)
	return nil
}
