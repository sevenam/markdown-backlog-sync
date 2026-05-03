package cli

import (
	"github.com/sevenam/markdown-backlog-sync/internal/provider"
	githubprovider "github.com/sevenam/markdown-backlog-sync/internal/provider/github"
)

// newProviderRegistry returns a Registry with all built-in provider kinds
// registered. Add new provider packages here as they are implemented.
func newProviderRegistry() *provider.Registry {
	r := provider.NewRegistry()
	githubprovider.Register(r)
	return r
}
