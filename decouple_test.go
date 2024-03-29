package decouple

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/bobg/go-generics/v3/maps"
	"github.com/bobg/go-generics/v3/set"
	// "github.com/davecgh/go-spew/spew"
)

func TestCheck(t *testing.T) {
	checker, err := NewCheckerFromDir("_testdata")
	if err != nil {
		t.Fatal(err)
	}

	// if testing.Verbose() {
	// 	checker.Verbose = true
	// }

	tuples, err := checker.Check()
	if err != nil {
		t.Fatal(err)
	}

	for _, tuple := range tuples {
		t.Run(tuple.F.Name.Name, func(t *testing.T) {
			if tuple.F.Doc == nil {
				t.Fatal("no doc")
			}
			var docb bytes.Buffer
			for _, c := range tuple.F.Doc.List {
				docb.WriteString(strings.TrimLeft(c.Text, "/"))
				docb.WriteByte('\n')
			}

			var (
				dec = json.NewDecoder(&docb)
				pre map[string]map[string]string
			)
			if err := dec.Decode(&pre); err != nil {
				t.Fatalf("unmarshaling `%s`: %s", docb.String(), err)
			}

			var (
				gotParamNames  = set.New(maps.Keys(tuple.M)...)
				wantParamNames = set.New(maps.Keys(pre)...)
			)
			if !gotParamNames.Equal(wantParamNames) {
				t.Fatalf("got param names %v, want %v", gotParamNames.Slice(), wantParamNames.Slice())
			}

			for paramName, methods := range pre {
				t.Run(paramName, func(t *testing.T) {
					var (
						gotMethodNames  = set.New(maps.Keys(tuple.M[paramName])...)
						wantMethodNames = set.New(maps.Keys(methods)...)
					)
					if !gotMethodNames.Equal(wantMethodNames) {
						t.Fatalf("got method names %v, want %v", gotMethodNames.Slice(), wantMethodNames.Slice())
					}
					for methodName, sigstr := range methods {
						t.Run(methodName, func(t *testing.T) {
							typ, err := types.Eval(tuple.P.Fset, tuple.P.Types, tuple.F.Pos(), sigstr)
							if err != nil {
								t.Fatal(err)
							}
							if !types.Identical(tuple.M[paramName][methodName], typ.Type) {
								t.Errorf("got %s, want %s", tuple.M[paramName][methodName], typ.Type)
							}
						})
					}
				})
			}

			if !dec.More() {
				return
			}

			t.Run("intf", func(t *testing.T) {
				var intfnames map[string]string
				if err := dec.Decode(&intfnames); err != nil {
					t.Fatalf("unmarshaling interface names: %s", err)
				}

				for paramName, intfname := range intfnames {
					t.Run(paramName, func(t *testing.T) {
						got := checker.NameForMethods(tuple.M[paramName])
						if got != intfname {
							t.Errorf("got %s, want %s", got, intfname)
						}
					})
				}
			})
		})
	}
}

func TestGetIdent(t *testing.T) {
	var expr ast.Expr = &ast.BasicLit{Kind: token.INT, Value: "42"}

	if ident := getIdent(expr); ident != nil {
		t.Errorf("got %v, want nil", ident)
	}

	expr = ast.NewIdent("foo")

	if ident := getIdent(expr); ident == nil || ident.Name != "foo" {
		t.Errorf("got %v, want foo", ident)
	}

	expr = &ast.ParenExpr{X: expr}

	if ident := getIdent(expr); ident == nil || ident.Name != "foo" {
		t.Errorf("got %v, want foo", ident)
	}
}
