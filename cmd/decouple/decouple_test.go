package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bobg/go-generics/v3/iter"
)

func TestRunJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := run(buf, false, true, []string{"../.."}); err != nil {
		t.Fatal(err)
	}

	var (
		got []jtuple
		dec = json.NewDecoder(buf)
	)
	for dec.More() {
		var val jtuple
		if err := dec.Decode(&val); err != nil {
			t.Fatal(err)
		}
		val.FileName = filepath.Base(val.FileName)
		got = append(got, val)
	}

	want := []jtuple{{
		PackageName: "main",
		FileName:    "main.go",
		Line:        100,
		FuncName:    "showJSON",
		Params: []jparam{{
			Name: "checker",
			Methods: []string{
				"NameForMethods",
			},
		}},
	}}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRunPlain(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := run(buf, false, false, []string{"../.."}); err != nil {
		t.Fatal(err)
	}

	lines, err := iter.ToSlice(iter.Lines(buf))
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if !strings.HasSuffix(lines[0], ": showJSON") {
		t.Fatalf(`line 1 is "%s", want something ending in ": showJSON"`, lines[0])
	}

	lines[1] = strings.TrimSpace(lines[1])
	const want = "checker: [NameForMethods]"
	if lines[1] != want {
		t.Fatalf(`line 2 is "%s", want "%s"`, lines[1], want)
	}
}
