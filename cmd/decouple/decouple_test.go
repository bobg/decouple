package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRun(t *testing.T) {
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
