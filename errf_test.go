package decouple

import (
	"errors"
	"testing"
)

func TestErrf(t *testing.T) {
	got := errf("What's a %d?", 412)

	var d derr
	if !errors.As(got, &d) {
		t.Errorf("got %v, want derr", got)
	}

	const want = "What's a 412?"
	if got.Error() != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
