package decouple

import (
	"context"
	"go/ast"
	"testing"

	"github.com/bobg/go-generics/maps"
	"github.com/bobg/go-generics/set"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/tools/go/packages"
	// "github.com/davecgh/go-spew/spew"
)

func TestAnalyze(t *testing.T) {
	ctx := context.Background()

	conf := &packages.Config{
		Context: ctx,
		Dir:     "_testdata",
		Mode:    PkgMode,
	}
	pkgs, err := packages.Load(conf, "./...")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, pkgerr := range pkg.Errors {
			err = multierr.Append(err, errors.Wrapf(pkgerr, "in package %s", pkg.PkgPath))
		}
	}
	if err != nil {
		t.Fatal(err)
	}

	var (
		readerMethods       = set.New[string]("Read")
		readerCloserMethods = set.New[string]("Read", "Close")
	)

	for i, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fndecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				t.Run(fndecl.Name.Name, func(t *testing.T) {
					for _, field := range fndecl.Type.Params.List {
						for _, name := range field.Names {
							if name.Name == "_" {
								continue
							}
							t.Run(name.Name, func(t *testing.T) {
								got, err := AnalyzeParam(name, fndecl, pkgs, i, false)
								if err != nil {
									t.Fatal(err)
								}
								gotMethodNames := set.New[string](maps.Keys(got)...)
								switch name.Name {
								case "r":
									if !gotMethodNames.Equal(readerMethods) {
										t.Errorf("got %v, want %v", got, readerMethods)
									}

								case "rc":
									if !gotMethodNames.Equal(readerCloserMethods) {
										t.Errorf("got %v, want %v", got, readerCloserMethods)
									}

								default:
									if gotMethodNames.Len() > 0 {
										t.Errorf("got %v, want nil", got)
									}
								}
							})
						}
					}
				})
			}
		}
	}
}
