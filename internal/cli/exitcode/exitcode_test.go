package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestCodeOf(t *testing.T) {
	if got := CodeOf(nil); got != OK {
		t.Fatalf("nil err: want OK, got %d", got)
	}
	if got := CodeOf(errors.New("plain")); got != Generic {
		t.Fatalf("plain err: want Generic, got %d", got)
	}
	wrapped := fmt.Errorf("outer: %w", Errorf(Auth, "bad token"))
	if got := CodeOf(wrapped); got != Auth {
		t.Fatalf("wrapped: want Auth, got %d", got)
	}
}

func TestErrorMessage(t *testing.T) {
	e := Errorf(Network, "boom %d", 42)
	if e.Error() != "boom 42" {
		t.Fatalf("unexpected message: %q", e.Error())
	}
	if e.Code != Network {
		t.Fatalf("code: got %d want %d", e.Code, Network)
	}
}

func TestWrapNil(t *testing.T) {
	if Wrap(Network, nil) != nil {
		t.Fatal("Wrap(_, nil) should be nil")
	}
}
