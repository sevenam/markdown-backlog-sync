// Package auth resolves and stores Personal Access Tokens used by
// providers. v1 is PAT-only.
//
// Resolution precedence at call time:
//  1. The --token flag (caller passes via WithExplicitToken).
//  2. Provider-specific env var: MBS_TOKEN_<UPPERSNAKE(name)>
//  3. Generic env vars matched against provider kind:
//     GITHUB_TOKEN for github-issues / github-projects-v2
//     AZURE_DEVOPS_PAT for azure-devops-boards
//  4. The configured Backend (OS keyring or encrypted file).
//
// Tokens are never logged.
package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Backend abstracts persistent credential storage.
type Backend interface {
	Get(account string) (string, error)
	Set(account, secret string) error
	Delete(account string) error
	Name() string
}

// ErrNotFound is returned by Backend.Get when no credential exists.
var ErrNotFound = errors.New("credential not found")

// Service is the OS-keyring service name used by all backends.
const Service = "markdown-backlog-sync"

// Resolver implements provider.CredentialResolver against a Backend, with
// env-var and explicit overrides.
type Resolver struct {
	Backend        Backend
	ExplicitTokens map[string]string // providerName -> token (from --token)
	ProviderKinds  map[string]string // providerName -> kind (used for generic envs and account scoping)
	ProviderScopes map[string]string // providerName -> stable remote scope (e.g. "owner/repo")
}

// NewResolver returns a Resolver wrapping the given backend.
func NewResolver(b Backend) *Resolver {
	return &Resolver{
		Backend:        b,
		ExplicitTokens: map[string]string{},
		ProviderKinds:  map[string]string{},
		ProviderScopes: map[string]string{},
	}
}

// AccountKey returns the backend account string used to store the
// credential for the named provider. Including kind+scope avoids
// collisions when two workspaces both use a provider named, e.g.,
// "default" against different remotes. When neither kind nor scope is
// known, the bare provider name is used.
func (r *Resolver) AccountKey(providerName string) string {
	kind := r.ProviderKinds[providerName]
	scope := r.ProviderScopes[providerName]
	if kind == "" && scope == "" {
		return providerName
	}
	return kind + "|" + scope + "|" + providerName
}

// Resolve returns the token for the named provider, applying the precedence
// described on the package doc.
func (r *Resolver) Resolve(providerName string) (string, error) {
	if t := r.ExplicitTokens[providerName]; t != "" {
		return t, nil
	}
	if t := os.Getenv(specificEnv(providerName)); t != "" {
		return t, nil
	}
	if kind := r.ProviderKinds[providerName]; kind != "" {
		if env := genericEnvForKind(kind); env != "" {
			if t := os.Getenv(env); t != "" {
				return t, nil
			}
		}
	}
	if r.Backend == nil {
		return "", fmt.Errorf("no credential for provider %q (no backend configured)", providerName)
	}
	t, err := r.Backend.Get(r.AccountKey(providerName))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("no credential for provider %q (run `mbs auth login --provider %s`)", providerName, providerName)
		}
		return "", err
	}
	return t, nil
}

// Login stores a token for the named provider in the backend.
func (r *Resolver) Login(providerName, token string) error {
	if r.Backend == nil {
		return errors.New("no credential backend configured")
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("token is empty")
	}
	return r.Backend.Set(r.AccountKey(providerName), token)
}

// Logout removes a stored token.
func (r *Resolver) Logout(providerName string) error {
	if r.Backend == nil {
		return errors.New("no credential backend configured")
	}
	return r.Backend.Delete(r.AccountKey(providerName))
}

// Has returns true if the backend currently holds a credential for name.
func (r *Resolver) Has(providerName string) (bool, error) {
	if r.Backend == nil {
		return false, nil
	}
	_, err := r.Backend.Get(r.AccountKey(providerName))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return false, err
}

func specificEnv(providerName string) string {
	var b strings.Builder
	b.WriteString("MBS_TOKEN_")
	for _, ch := range providerName {
		switch {
		case ch >= 'a' && ch <= 'z':
			b.WriteRune(ch - 'a' + 'A')
		case ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func genericEnvForKind(kind string) string {
	switch kind {
	case "github-issues", "github-projects-v2":
		return "GITHUB_TOKEN"
	case "azure-devops-boards":
		return "AZURE_DEVOPS_PAT"
	}
	return ""
}
