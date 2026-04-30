package provider

import (
	"errors"
	"testing"
)

type stubCreds struct{}

func (stubCreds) Resolve(string) (string, error) { return "x", nil }

func TestRegistryBuildAndKinds(t *testing.T) {
	r := NewRegistry()
	r.Register("dummy", func(name string, _ map[string]any, _ CredentialResolver) (Provider, error) {
		return nil, nil
	})
	if _, err := r.Build("dummy", "n", nil, stubCreds{}); err != nil {
		t.Fatal(err)
	}
	_, err := r.Build("nope", "n", nil, stubCreds{})
	if !errors.Is(err, ErrUnknownType) {
		t.Fatalf("want ErrUnknownType, got %v", err)
	}
	kinds := r.Kinds()
	if len(kinds) != 1 || kinds[0] != "dummy" {
		t.Fatalf("kinds: %v", kinds)
	}
}

func TestRegistryDuplicatePanics(t *testing.T) {
	r := NewRegistry()
	r.Register("dup", func(string, map[string]any, CredentialResolver) (Provider, error) { return nil, nil })
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate Register")
		}
	}()
	r.Register("dup", func(string, map[string]any, CredentialResolver) (Provider, error) { return nil, nil })
}
