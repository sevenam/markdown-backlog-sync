package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "backlog.config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadValid(t *testing.T) {
	p := writeTemp(t, `
[workspace]
items_dir = "items"

[[provider]]
name = "gh"
type = "github-issues"
repo = "owner/repo"

[[provider]]
name = "azdo"
type = "azure-devops-boards"
organization = "contoso"
project = "Platform"
team = "Platform Team"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Workspace.ItemsDir != "items" {
		t.Fatalf("items_dir = %q", cfg.Workspace.ItemsDir)
	}
	if len(cfg.Providers) != 2 {
		t.Fatalf("providers = %d", len(cfg.Providers))
	}
	gh, ok := cfg.Provider("gh")
	if !ok {
		t.Fatal("missing gh provider")
	}
	if gh.Options["repo"] != "owner/repo" {
		t.Fatalf("gh repo opt = %v", gh.Options["repo"])
	}
}

func TestApplyDefaults(t *testing.T) {
	p := writeTemp(t, `[[provider]]
name = "gh"
type = "github-issues"
repo = "owner/repo"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ApplyDefaults()
	if cfg.Workspace.ItemsDir != "backlog" {
		t.Fatalf("default items_dir = %q", cfg.Workspace.ItemsDir)
	}
}

func TestValidateDuplicateName(t *testing.T) {
	p := writeTemp(t, `
[[provider]]
name = "x"
type = "github-issues"
repo = "a/b"

[[provider]]
name = "x"
type = "github-issues"
repo = "c/d"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("want duplicate error, got %v", err)
	}
}

func TestValidateMissingType(t *testing.T) {
	p := writeTemp(t, `
[[provider]]
name = "x"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("want error for missing type")
	}
}
