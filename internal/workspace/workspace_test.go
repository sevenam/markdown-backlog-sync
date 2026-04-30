package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverNotFound(t *testing.T) {
	dir := t.TempDir()
	if _, err := Discover(dir); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCreateAndDiscover(t *testing.T) {
	root := t.TempDir()
	ws, wrote, err := Create(root, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !wrote {
		t.Fatal("expected fresh config to be written")
	}
	if _, err := os.Stat(ws.ConfigPath); err != nil {
		t.Fatalf("config not created: %v", err)
	}
	if _, err := os.Stat(ws.ItemsDir); err != nil {
		t.Fatalf("items dir not created: %v", err)
	}
	if _, err := os.Stat(ws.StateDir); err != nil {
		t.Fatalf("state dir not created: %v", err)
	}

	// Discovery from a nested directory should return the workspace root.
	nested := filepath.Join(ws.ItemsDir, "deep", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(nested)
	if err != nil {
		t.Fatal(err)
	}
	gotResolved, _ := filepath.EvalSymlinks(got)
	wantResolved, _ := filepath.EvalSymlinks(ws.Root)
	if gotResolved != wantResolved {
		t.Fatalf("discover: got %q want %q", got, ws.Root)
	}

	// Re-Create without overwrite should be a no-op for the config.
	_, wrote2, err := Create(root, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if wrote2 {
		t.Fatal("did not expect config to be rewritten")
	}
}

func TestCreateCustomItemsDir(t *testing.T) {
	root := t.TempDir()
	ws, _, err := Create(root, "items", false)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ws.ItemsDir) != "items" {
		t.Fatalf("items dir = %q", ws.ItemsDir)
	}
}
