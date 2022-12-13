package decouple

import (
	"context"
	"reflect"
	"testing"

	"github.com/bobg/go-generics/set"
	// "github.com/davecgh/go-spew/spew"
)

func TestLoad(t *testing.T) {
	ctx := context.Background()
	res, err := Load(ctx, "_testdata")
	if err != nil {
		t.Fatal(err)
	}

	// spew.Dump(res)

	got := make(map[string]map[string]set.Of[string])
	for _, tuple := range res {
		got[tuple.F.Name.Name] = tuple.M
	}

	want := map[string]map[string]set.Of[string]{
		"Yes1":  {"f": set.New[string]("Read")},
		"Yes2":  {"f": set.New[string]("Read")},
		"Yes5":  {"f": set.New[string]("Read")},
		"Yes7":  {"f": set.New[string]("Read", "Close")},
		"Yes8":  {"i": set.New[string]("x")},
		"Yes11": {"f": set.New[string]("Read")},
		"Yes13": {"f": set.New[string]("Read")},
		"Yes14": {"f": set.New[string]("Read")},
		"Yes16": {"f": set.New[string]("Read")},
		"Yes17": {"f": set.New[string]("Read")},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
