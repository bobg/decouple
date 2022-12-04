package decouple

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestLoad(t *testing.T) {
	ctx := context.Background()
	res, err := Load(ctx, "_testdata")
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(res)
}
