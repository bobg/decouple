package decouple

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"strings"
	"testing"

	"github.com/bobg/go-generics/v4/set"
	"golang.org/x/tools/go/packages"
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
				gotParamNames  = set.Collect(maps.Keys(tuple.M))
				wantParamNames = set.Collect(maps.Keys(pre))
			)
			if !gotParamNames.Equal(wantParamNames) {
				t.Fatalf("got param names %v, want %v", gotParamNames.Slice(), wantParamNames.Slice())
			}

			for paramName, methods := range pre {
				t.Run(paramName, func(t *testing.T) {
					var (
						gotMethodNames  = set.Collect(maps.Keys(tuple.M[paramName]))
						wantMethodNames = set.Collect(maps.Keys(methods))
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
						gotPkg, gotName := checker.NameForMethods(tuple.M[paramName])
						if gotName == "" {
							t.Fatalf("no named interface found for param %s", paramName)
						}
						got := gotPkg.PkgPath + "." + gotName
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

func TestBestChooser(t *testing.T) {
	cases := []struct {
		name                 string
		pkgpath1, pkgpath2   string
		modpath1, modpath2   string
		isAlias1, isAlias2   bool
		isMain1, isMain2     bool
		isDirect1, isDirect2 bool
		want1                bool
	}{{
		name:      "stdlib vs non-stdlib",
		pkgpath1:  "fmt",
		pkgpath2:  "github.com/bobg/decouple",
		modpath2:  "github.com/bobg/decouple",
		isAlias1:  false,
		isAlias2:  false,
		isMain1:   false,
		isMain2:   true,
		isDirect1: true,
		isDirect2: true,
		want1:     true,
	}, {
		name:      "non-stdlib vs stdlib",
		pkgpath1:  "github.com/bobg/decouple",
		modpath1:  "github.com/bobg/decouple",
		pkgpath2:  "fmt",
		isAlias1:  false,
		isAlias2:  false,
		isMain1:   true,
		isMain2:   false,
		isDirect1: true,
		isDirect2: true,
		want1:     false,
	}, {
		name:     "higher alias vs lower non-alias in main module",
		pkgpath1: "google.golang.org/protobuf/proto",
		modpath1: "google.golang.org/protobuf",
		pkgpath2: "google.golang.org/protobuf/reflect/protoreflect",
		modpath2: "google.golang.org/protobuf",
		isMain1:  true,
		isMain2:  true,
		isAlias1: true,
		isAlias2: false,
		want1:    true,
	}, {
		name:     "lower non-alias vs higher alias in main module",
		pkgpath1: "google.golang.org/protobuf/reflect/protoreflect",
		modpath1: "google.golang.org/protobuf",
		pkgpath2: "google.golang.org/protobuf/proto",
		modpath2: "google.golang.org/protobuf",
		isMain1:  true,
		isMain2:  true,
		isAlias1: false,
		isAlias2: true,
		want1:    false,
	}, {
		name:      "higher alias vs lower non-alias in direct dependency",
		pkgpath1:  "google.golang.org/protobuf/proto",
		modpath1:  "google.golang.org/protobuf",
		pkgpath2:  "google.golang.org/protobuf/reflect/protoreflect",
		modpath2:  "google.golang.org/protobuf",
		isDirect1: true,
		isDirect2: true,
		isAlias1:  true,
		isAlias2:  false,
		want1:     true,
	}, {
		name:      "lower non-alias vs higher alias in direct dependency",
		pkgpath1:  "google.golang.org/protobuf/reflect/protoreflect",
		modpath1:  "google.golang.org/protobuf",
		pkgpath2:  "google.golang.org/protobuf/proto",
		modpath2:  "google.golang.org/protobuf",
		isDirect1: true,
		isDirect2: true,
		isAlias1:  false,
		isAlias2:  true,
		want1:     false,
	}, {
		name:     "higher alias vs lower non-alias in other dependency",
		pkgpath1: "google.golang.org/protobuf/proto",
		modpath1: "google.golang.org/protobuf",
		pkgpath2: "google.golang.org/protobuf/reflect/protoreflect",
		modpath2: "google.golang.org/protobuf",
		isAlias1: true,
		isAlias2: false,
		want1:    true,
	}, {
		name:     "lower non-alias vs higher alias in other dependency",
		pkgpath1: "google.golang.org/protobuf/reflect/protoreflect",
		modpath1: "google.golang.org/protobuf",
		pkgpath2: "google.golang.org/protobuf/proto",
		modpath2: "google.golang.org/protobuf",
		isAlias1: false,
		isAlias2: true,
		want1:    false,
	}, {
		name:     "same height alias vs non-alias in same other-dependency module",
		pkgpath1: "google.golang.org/protobuf/proto",
		modpath1: "google.golang.org/protobuf",
		pkgpath2: "google.golang.org/protobuf/protoadapt",
		modpath2: "google.golang.org/protobuf",
		isAlias1: true,
		isAlias2: false,
		want1:    true,
	}, {
		name:     "same height non-alias vs alias in same other-dependency module",
		pkgpath1: "google.golang.org/protobuf/proto",
		modpath1: "google.golang.org/protobuf",
		pkgpath2: "google.golang.org/protobuf/protoadapt",
		modpath2: "google.golang.org/protobuf",
		isAlias1: false,
		isAlias2: true,
		want1:    true,
	}, {
		name:      "alias vs non-alias in different direct dependency modules",
		pkgpath1:  "github.com/foo/pkg",
		modpath1:  "github.com/foo/pkg",
		pkgpath2:  "github.com/bar/pkg",
		modpath2:  "github.com/bar/pkg",
		isDirect1: true,
		isDirect2: true,
		isAlias1:  true,
		isAlias2:  false,
		want1:     false,
	}, {
		name:      "non-alias vs alias in different direct dependency modules",
		pkgpath1:  "github.com/foo/pkg",
		modpath1:  "github.com/foo/pkg",
		pkgpath2:  "github.com/bar/pkg",
		modpath2:  "github.com/bar/pkg",
		isDirect1: true,
		isDirect2: true,
		isAlias1:  false,
		isAlias2:  true,
		want1:     true,
	}, {
		name:     "alias vs non-alias in different non-direct-dependency modules",
		pkgpath1: "github.com/foo/pkg",
		modpath1: "github.com/foo/pkg",
		pkgpath2: "github.com/bar/pkg",
		modpath2: "github.com/bar/pkg",
		isAlias1: true,
		isAlias2: false,
		want1:    false,
	}, {
		name:     "non-alias vs alias in different non-direct-dependency modules",
		pkgpath1: "github.com/foo/pkg",
		modpath1: "github.com/foo/pkg",
		pkgpath2: "github.com/bar/pkg",
		modpath2: "github.com/bar/pkg",
		isAlias1: false,
		isAlias2: true,
		want1:    true,
	}, {
		name:      "alias vs non-alias in main vs direct-dependency module",
		pkgpath1:  "github.com/bobg/decouple",
		modpath1:  "github.com/bobg/decouple",
		pkgpath2:  "github.com/some/dependency",
		modpath2:  "github.com/some/dependency",
		isMain1:   true,
		isDirect2: true,
		isAlias1:  true,
		isAlias2:  false,
		want1:     true,
	}, {
		name:      "non-alias vs alias in main vs direct-dependency module",
		pkgpath1:  "github.com/bobg/decouple",
		modpath1:  "github.com/bobg/decouple",
		pkgpath2:  "github.com/some/dependency",
		modpath2:  "github.com/some/dependency",
		isMain1:   true,
		isDirect2: true,
		isAlias1:  false,
		isAlias2:  true,
		want1:     true,
	}, {
		name:      "alias vs non-alias in direct-dependency vs main module",
		pkgpath1:  "github.com/bobg/decouple",
		modpath1:  "github.com/bobg/decouple",
		pkgpath2:  "github.com/some/dependency",
		modpath2:  "github.com/some/dependency",
		isDirect1: true,
		isMain2:   true,
		isAlias1:  true,
		isAlias2:  false,
		want1:     false,
	}, {
		name:      "non-alias vs alias in direct-dependency vs main module",
		pkgpath1:  "github.com/bobg/decouple",
		modpath1:  "github.com/bobg/decouple",
		pkgpath2:  "github.com/some/dependency",
		modpath2:  "github.com/some/dependency",
		isDirect1: true,
		isMain2:   true,
		isAlias1:  false,
		isAlias2:  true,
		want1:     false,
	}, {
		name:      "main module vs direct dependency",
		pkgpath1:  "github.com/bobg/decouple",
		modpath1:  "github.com/bobg/decouple",
		pkgpath2:  "github.com/some/dependency",
		modpath2:  "github.com/some/dependency",
		isMain1:   true,
		isDirect2: true,
		want1:     true,
	}, {
		name:      "direct dependency vs main module",
		pkgpath1:  "github.com/some/dependency",
		modpath1:  "github.com/some/dependency",
		pkgpath2:  "github.com/bobg/decouple",
		modpath2:  "github.com/bobg/decouple",
		isDirect1: true,
		isMain2:   true,
		want1:     false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkg1 := &packages.Package{
				PkgPath: tc.pkgpath1,
				Module: &packages.Module{
					Main:     tc.isMain1,
					Indirect: !tc.isDirect1,
				},
			}
			var chooser bestChooser
			chooser.choose(pkg1, "x", tc.isAlias1)

			pkg2 := &packages.Package{
				PkgPath: tc.pkgpath2,
				Module: &packages.Module{
					Main:     tc.isMain2,
					Indirect: !tc.isDirect2,
				},
			}

			chooser.choose(pkg2, "y", tc.isAlias2)

			if tc.want1 {
				if chooser.name != "x" {
					t.Errorf("got name %q, want x", chooser.name)
				}
			} else {
				if chooser.name != "y" {
					t.Errorf("got name %q, want y", chooser.name)
				}
			}
		})
	}
}
