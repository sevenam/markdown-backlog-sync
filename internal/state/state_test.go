package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".sync")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := &ItemState{
		LocalID:      "abc-123",
		LocalPath:    "backlog/foo.md",
		Provider:     "gh-main",
		RemoteID:     "42",
		RemoteURL:    "https://github.com/o/r/issues/42",
		RemoteRev:    "etag-xyz",
		ContentHash:  Hash("# Hello\n"),
		LastSyncedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.SaveItem(want); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveSnapshot("abc-123", "# Hello\n"); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadItem("abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.RemoteID != "42" || got.ContentHash != want.ContentHash {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	snap, err := s.LoadSnapshot("abc-123")
	if err != nil || snap != "# Hello\n" {
		t.Fatalf("snapshot round-trip: %q %v", snap, err)
	}

	idx, err := s.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	idx.Mapping[MapKey("gh-main", "42")] = "abc-123"
	if err := s.SaveIndex(idx); err != nil {
		t.Fatal(err)
	}
	idx2, err := s.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if idx2.Mapping[MapKey("gh-main", "42")] != "abc-123" {
		t.Fatalf("index mapping not persisted: %+v", idx2)
	}

	if err := s.DeleteItem("abc-123"); err != nil {
		t.Fatal(err)
	}
	again, err := s.LoadItem("abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if again != nil {
		t.Fatalf("item should be deleted, got %+v", again)
	}
	if snap, _ := s.LoadSnapshot("abc-123"); snap != "" {
		t.Fatalf("snapshot should be deleted, got %q", snap)
	}
}

func TestRejectsUnsafeLocalID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".sync")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "..", "a/b", "..\\evil", "/abs", "a\x00b", strings.Repeat("a", 200)} {
		t.Run(bad, func(t *testing.T) {
			err := s.SaveItem(&ItemState{LocalID: bad})
			if err == nil {
				t.Fatalf("expected error for %q", bad)
			}
		})
	}
}

func TestSchemaTooNew(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".sync")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Hand-write an item state with a future schema version.
	bad := []byte(`{"schema":99,"local_id":"x","local_path":"p","provider":"q","remote_id":"1","content_hash":""}`)
	p := filepath.Join(dir, "items", "x.json")
	if err := os.WriteFile(p, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadItem("x"); err == nil || !strings.Contains(err.Error(), "newer than supported") {
		t.Fatalf("want schema error, got %v", err)
	}
}

func TestLoadMissingItemAndIndex(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".sync")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	it, err := s.LoadItem("missing")
	if err != nil || it != nil {
		t.Fatalf("missing item: %v %v", it, err)
	}
	idx, err := s.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Mapping) != 0 {
		t.Fatalf("empty index expected")
	}
}
