package decouple

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/bobg/go-generics/v2/maps"
	"github.com/bobg/go-generics/v2/set"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/tools/go/packages"
	// "github.com/davecgh/go-spew/spew"
)

func TestAnalyze(t *testing.T) {
	ctx := context.Background()

	conf := &packages.Config{
		Context: ctx,
		Dir:     "_testdata/m",
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
		checker             = NewCheckerFromPackages(pkgs)
		readerMethods       = set.New[string]("Read")
		readerCloserMethods = set.New[string]("Read", "Close")
	)

	checker.Interfaces = true

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fndecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				t.Run(fndecl.Name.Name, func(t *testing.T) {
					for _, field := range fndecl.Type.Params.List {
						for _, name := range field.Names {
							switch name.Name {
							case "_", "ctx":
								continue
							}

							t.Run(name.Name, func(t *testing.T) {
								got, err := checker.CheckParam(pkg, fndecl, name)
								if err != nil {
									t.Fatal(err)
								}
								var (
									gotMethodNames = set.New[string](maps.Keys(got)...)
									methodSetName  = checker.NameForMethods(got)
								)
								switch name.Name {
								case "r":
									if !gotMethodNames.Equal(readerMethods) {
										t.Errorf("got %v, want %v", got, readerMethods)
									}
									switch methodSetName {
									case "":
										t.Error("did not find a name for this method set")
									case "io.Reader": // ok
									default:
										t.Errorf("got %s for this method set, want io.Reader", methodSetName)
									}

								case "rc":
									if !gotMethodNames.Equal(readerCloserMethods) {
										t.Errorf("got %v, want %v", got, readerCloserMethods)
									}
									switch methodSetName {
									case "":
										t.Error("did not find a name for this method set")
									case "io.ReadCloser": // ok
									default:
										t.Errorf("got %s for this method set, want io.Reader", methodSetName)
									}

								default:
									if gotMethodNames.Len() > 0 {
										t.Errorf("got %v, want nil", got)
									}
									if methodSetName != "" {
										t.Errorf("got %s for this method set, want no name", methodSetName)
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

func TestA(t *testing.T) {
	checker, err := NewCheckerFromDir("_testdata/a")
	if err != nil {
		t.Fatal(err)
	}

	checker.Interfaces = true

	tuples, err := checker.Check()
	if err != nil {
		t.Fatal(err)
	}

	var (
		fsets = make(map[string]*token.FileSet)
		pkgs  = make(map[string]*types.Package)
		poss  = make(map[string]token.Pos)
	)

	got := make(map[string]map[string]MethodMap)
	for _, tuple := range tuples {
		keyparts := []string{tuple.P.PkgPath}
		if tuple.F.Recv != nil && len(tuple.F.Recv.List) > 0 {
			recv := tuple.F.Recv.List[0].Type
			if info := tuple.P.TypesInfo.Types[recv]; info.Type != nil {
				keyparts = append(keyparts, types.TypeString(info.Type, types.RelativeTo(tuple.P.Types)))
			}
		}
		keyparts = append(keyparts, tuple.F.Name.Name)
		key := strings.Join(keyparts, ".")
		got[key] = tuple.M

		fsets[key] = tuple.P.Fset
		pkgs[key] = tuple.P.Types
		poss[key] = tuple.F.Pos()
	}

	want := map[string]map[string]map[string]string{
		"a.X": {
			"r": {
				"Read": "func([]byte) (int, error)",
			},
		},
	}

	for k, v := range want {
		fset, ok := fsets[k]
		if !ok {
			t.Fatalf("no fset for %s", k)
		}
		pkg, ok := pkgs[k]
		if !ok {
			t.Fatalf("no pkg for %s", k)
		}
		pos, ok := poss[k]
		if !ok {
			t.Fatalf("no pos for %s", k)
		}

		for paramname, methods := range v {
			for methodname, sigstr := range methods {
				typ, err := types.Eval(fset, pkg, pos, sigstr)
				if err != nil {
					t.Fatalf("%s %s %s: %s", k, paramname, methodname, err)
				}
				if !types.Identical(got[k][paramname][methodname], typ.Type) {
					t.Errorf("%s %s %s: got %s, want %s", k, paramname, methodname, got[k][paramname][methodname], typ.Type)
				}
			}
		}
	}
}
