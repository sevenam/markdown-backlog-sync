package auth

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestMemoryBackend(t *testing.T) {
	b := NewMemoryBackend()
	if _, err := b.Get("x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := b.Set("x", "secret"); err != nil {
		t.Fatal(err)
	}
	v, err := b.Get("x")
	if err != nil || v != "secret" {
		t.Fatalf("get: %q %v", v, err)
	}
	if err := b.Delete("x"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Get("x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}

func TestFileBackendRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "creds.bin")
	b := NewFileBackend(p, "passphrase")
	if err := b.Set("acct", "tok-12345"); err != nil {
		t.Fatal(err)
	}
	// Re-open to ensure persistence.
	b2 := NewFileBackend(p, "passphrase")
	v, err := b2.Get("acct")
	if err != nil || v != "tok-12345" {
		t.Fatalf("round-trip: %q %v", v, err)
	}
	// Wrong passphrase fails.
	bad := NewFileBackend(p, "nope")
	if _, err := bad.Get("acct"); err == nil {
		t.Fatal("expected decryption failure with wrong passphrase")
	}
}

func TestResolverPrecedence(t *testing.T) {
	be := NewMemoryBackend()
	r := NewResolver(be)
	r.ProviderKinds["gh-main"] = "github-issues"
	// Seed the backend via Resolver so the account-key formula is applied.
	if err := r.Login("gh-main", "from-backend"); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GITHUB_TOKEN", "from-generic-env")
	t.Setenv("MBS_TOKEN_GH_MAIN", "from-specific-env")

	// Explicit beats env beats backend.
	r.ExplicitTokens["gh-main"] = "from-flag"
	if got, _ := r.Resolve("gh-main"); got != "from-flag" {
		t.Fatalf("explicit: %q", got)
	}
	delete(r.ExplicitTokens, "gh-main")
	if got, _ := r.Resolve("gh-main"); got != "from-specific-env" {
		t.Fatalf("specific env: %q", got)
	}
	t.Setenv("MBS_TOKEN_GH_MAIN", "")
	if got, _ := r.Resolve("gh-main"); got != "from-generic-env" {
		t.Fatalf("generic env: %q", got)
	}
	t.Setenv("GITHUB_TOKEN", "")
	if got, _ := r.Resolve("gh-main"); got != "from-backend" {
		t.Fatalf("backend: %q", got)
	}
}

func TestAccountKeyScoping(t *testing.T) {
	be := NewMemoryBackend()
	r := NewResolver(be)
	if got := r.AccountKey("default"); got != "default" {
		t.Fatalf("unscoped: %q", got)
	}
	r.ProviderKinds["default"] = "github-issues"
	r.ProviderScopes["default"] = "owner/repo"
	if got := r.AccountKey("default"); got != "github-issues|owner/repo|default" {
		t.Fatalf("scoped: %q", got)
	}
}

func TestResolverMissing(t *testing.T) {
	r := NewResolver(NewMemoryBackend())
	if _, err := r.Resolve("nope"); err == nil {
		t.Fatal("expected error for missing credential")
	}
}

func TestSpecificEnvNormalization(t *testing.T) {
	if got := specificEnv("gh-main"); got != "MBS_TOKEN_GH_MAIN" {
		t.Fatalf("specificEnv: %q", got)
	}
	if got := specificEnv("AzDo.Platform"); got != "MBS_TOKEN_AZDO_PLATFORM" {
		t.Fatalf("specificEnv: %q", got)
	}
}
