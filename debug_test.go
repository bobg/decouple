package decouple

import (
	"io"
	"os"
	"testing"
)

func TestDebugf(t *testing.T) {
	a := analyzer{debug: true, level: 2}

	f, err := os.CreateTemp("", "decouple")
	if err != nil {
		t.Fatal(err)
	}
	tmpname := f.Name()
	defer os.Remove(tmpname)
	defer f.Close()

	oldStderr := os.Stderr
	os.Stderr = f
	defer func() { os.Stderr = oldStderr }()

	a.debugf("What's a %d?", 412)

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	f, err = os.Open(tmpname)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	const want = "    What's a 412?\n"

	if string(got) != want {
		t.Errorf("got %s, want %s", string(got), want)
	}
}
