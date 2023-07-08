package decouple

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/bobg/go-generics/v2/set"
	"github.com/bobg/go-generics/v2/slices"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/tools/go/packages"
)

// PkgMode is the minimal set of bit flags needed for the Config.Mode field of golang.org/x/go/packages
// for the result to be usable by a Checker.
const PkgMode = packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

// Checker is the object that can analyze a directory tree of Go code,
// or a set of packages loaded with "golang.org/x/go/packages".Load,
// or a single such package,
// or a function or function parameter in one.
//
// Set Verbose to true to get (very) verbose debugging output.
type Checker struct {
	Verbose bool

	pkgs            []*packages.Package
	namedInterfaces map[string]MethodMap // maps a package-qualified interface-type name to its method set
}

// NewCheckerFromDir creates a new Checker containing packages loaded
// (using "golang.org/x/go/packages".Load)
// from the given directory tree.
func NewCheckerFromDir(dir string) (Checker, error) {
	conf := &packages.Config{Dir: dir, Mode: PkgMode}
	pkgs, err := packages.Load(conf, "./...")
	if err != nil {
		return Checker{}, errors.Wrapf(err, "loading packages from %s", dir)
	}
	for _, pkg := range pkgs {
		for _, pkgerr := range pkg.Errors {
			err = multierr.Append(err, errors.Wrapf(pkgerr, "in package %s", pkg.PkgPath))
		}
	}
	if err != nil {
		return Checker{}, errors.Wrapf(err, "after loading packages from %s", dir)
	}
	return NewCheckerFromPackages(pkgs), nil
}

// NewCheckerFromPackages creates a new Checker containing the given packages,
// which should be the result of calling "golang.org/x/go/packages".Load
// with at least the bits in PkgMode set in the Config.Mode field.
func NewCheckerFromPackages(pkgs []*packages.Package) Checker {
	var (
		namedInterfaces = make(map[string]MethodMap)
		seen            = set.New[*packages.Package]()
	)
	for _, pkg := range pkgs {
		findNamedInterfaces(pkg, seen, namedInterfaces)
	}
	return Checker{pkgs: pkgs, namedInterfaces: namedInterfaces}
}

func findNamedInterfaces(pkg *packages.Package, seen set.Of[*packages.Package], namedInterfaces map[string]MethodMap) {
	if seen.Has(pkg) {
		return
	}
	seen.Add(pkg)

	for _, ipkg := range pkg.Imports {
		findNamedInterfaces(ipkg, seen, namedInterfaces)
	}

	if isInternal(pkg.PkgPath) {
		return
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gendecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if gendecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range gendecl.Specs {
				typespec, ok := spec.(*ast.TypeSpec)
				if !ok {
					// Should be impossible.
					continue
				}
				if !ast.IsExported(typespec.Name.Name) {
					continue
				}
				obj := pkg.TypesInfo.Defs[typespec.Name]
				if obj == nil {
					// Should be impossible.
					continue
				}
				intf := getInterface(obj.Type())
				if intf == nil {
					continue
				}
				mm := make(MethodMap)
				addMethodsToMap(intf, mm)
				name := pkg.PkgPath
				if strings.ContainsAny(name, "./") {
					name = `"` + name + `"`
				}
				name += "." + typespec.Name.Name
				namedInterfaces[name] = mm
			}
		}
	}

}

// Check checks all the packages in the Checker.
// It analyzes the functions in them,
// looking for parameters with concrete types that could be interfaces instead.
// The result is a list of Tuples,
// one for each function checked that has parameters eligible for decoupling.
func (ch Checker) Check() ([]Tuple, error) {
	var result []Tuple

	for _, pkg := range ch.pkgs {
		pkgResult, err := ch.CheckPackage(pkg)
		if err != nil {
			return nil, errors.Wrapf(err, "analyzing package %s", pkg.PkgPath)
		}
		result = append(result, pkgResult...)
	}

	return result, nil
}

// CheckPackage checks a single package.
// It should be one of the packages contained in the Checker.
// The result is a list of Tuples,
// one for each function checked that has parameters eligible for decoupling.
func (ch Checker) CheckPackage(pkg *packages.Package) ([]Tuple, error) {
	var result []Tuple

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fndecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			m, err := ch.CheckFunc(pkg, fndecl)
			if err != nil {
				return nil, errors.Wrapf(err, "analyzing function %s at %s", fndecl.Name.Name, pkg.Fset.Position(fndecl.Name.Pos()))
			}
			result = append(result, Tuple{
				F: fndecl,
				P: pkg,
				M: m,
			})
		}
	}

	return result, nil
}

// Tuple is the type of a result from Checker.Check and Checker.CheckPackage.
type Tuple struct {
	// F is the function declaration that this result is about.
	F *ast.FuncDecl

	// P is the package in which the function declaration appears.
	P *packages.Package

	// M is a map from the names of function parameters eligible for decoupling
	// to MethodMaps for each such parameter.
	M map[string]MethodMap
}

// Pos computes the filename and offset
// of the function name of the Tuple.
func (t Tuple) Pos() token.Position {
	return t.P.Fset.Position(t.F.Name.Pos())
}

// MethodMap maps a set of method names to their calling signatures.
type MethodMap = map[string]*types.Signature

// CheckFunc checks a single function declaration,
// which should appear in the given package,
// which should be one of the packages contained in the Checker.
// The result is a map from parameter names eligible for decoupling to MethodMaps.
func (ch Checker) CheckFunc(pkg *packages.Package, fndecl *ast.FuncDecl) (map[string]MethodMap, error) {
	result := make(map[string]MethodMap)
	for _, field := range fndecl.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == "_" {
				continue
			}

			nameResult, err := ch.CheckParam(pkg, fndecl, name)
			if err != nil {
				return nil, errors.Wrapf(err, "analyzing parameter %s of %s", name.Name, fndecl.Name.Name)
			}
			if len(nameResult) != 0 {
				result[name.Name] = nameResult
			}
		}
	}
	return result, nil
}

// CheckParam checks a single named parameter in a given function declaration,
// which must apepar in the given package,
// which should be one of the packages in the Checker.
// The result is a MethodMap for the parameter,
// and may be nil if the parameter is not eligible for decoupling.
func (ch Checker) CheckParam(pkg *packages.Package, fndecl *ast.FuncDecl, name *ast.Ident) (_ MethodMap, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				var d derr
				if errors.As(e, &d) {
					err = d
					return
				}
			}
			panic(r)
		}
	}()

	obj, ok := pkg.TypesInfo.Defs[name]
	if !ok {
		return nil, fmt.Errorf("no def found for %s", name.Name)
	}

	var (
		intf = getInterface(obj.Type())
		mm   MethodMap
	)
	if intf != nil {
		mm = make(MethodMap)
		addMethodsToMap(intf, mm)
	}
	a := analyzer{
		name:          name,
		obj:           obj,
		pkg:           pkg,
		objmethods:    mm,
		methods:       make(MethodMap),
		enclosingFunc: &funcDeclOrLit{decl: fndecl},
		debug:         ch.Verbose,
	}
	a.debugf("fn %s param %s", fndecl.Name.Name, name.Name)
	for _, stmt := range fndecl.Body.List {
		if !a.stmt(stmt) {
			return nil, nil
		}
	}

	if len(a.objmethods) > 0 {
		if len(a.methods) < len(a.objmethods) {
			// A smaller interface will do.
			return a.methods, nil
		}
		return nil, nil
	}
	return a.methods, nil
}

// NameForMethods takes a MethodMap
// and returns the name of an interface defining exactly the methods in it,
// if it can find one among the packages in the Checker.
// If there are multiple such interfaces,
// one is chosen arbitrarily.
func (ch Checker) NameForMethods(inp MethodMap) string {
	for name, mm := range ch.namedInterfaces {
		if sameMethodMaps(mm, inp) {
			return name
		}
	}
	return ""
}

type funcDeclOrLit struct {
	decl *ast.FuncDecl
	lit  *ast.FuncLit
}

type analyzer struct {
	name *ast.Ident
	obj  types.Object
	pkg  *packages.Package

	// objmethods is input: the methodmap for obj's type,
	// if that's an interface type.
	// methods is output: the set of methods actually used.
	objmethods, methods MethodMap

	enclosingFunc       *funcDeclOrLit
	enclosingSwitchStmt *ast.SwitchStmt

	level int
	debug bool
}

func (a *analyzer) enclosingFuncInfo() (types.Type, token.Position, bool) {
	if a.enclosingFunc == nil {
		return nil, token.Position{}, false
	}
	if decl := a.enclosingFunc.decl; decl != nil {
		obj, ok := a.pkg.TypesInfo.Defs[decl.Name]
		if !ok {
			return nil, token.Position{}, false
		}
		return obj.Type(), a.pos(obj), true
	}
	lit := a.enclosingFunc.lit
	tv, ok := a.pkg.TypesInfo.Types[lit]
	if !ok {
		return nil, token.Position{}, false
	}
	return tv.Type, a.pos(lit), true
}

func (a *analyzer) getSig(expr ast.Expr) *types.Signature {
	return getSig(a.pkg.TypesInfo.Types[expr].Type)
}

// Does expr denote the object in a?
func (a *analyzer) isObj(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.Ident:
		obj := a.pkg.TypesInfo.Uses[expr]
		return obj == a.obj

	case *ast.ParenExpr:
		return a.isObj(expr.X)

	default:
		return false
	}
}

func (a *analyzer) stmt(stmt ast.Stmt) (ok bool) {
	a.level++
	a.debugf("> stmt %#v", stmt)
	defer func() {
		a.debugf("< stmt %#v %v", stmt, ok)
		a.level--
	}()

	if stmt == nil {
		return true
	}

	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		for _, lhs := range stmt.Lhs {
			// I think we can ignore the rhs value if a.isObj(lhs).
			// What matters is only how our object is being used,
			// not what's being assigned to it.
			if !a.expr(lhs) {
				return false
			}
		}
		for i, rhs := range stmt.Rhs {
			// xxx do a recursive analysis of how this var is used!
			if a.isObj(rhs) && stmt.Tok != token.DEFINE {
				if stmt.Tok != token.ASSIGN {
					// Reject OP=
					return false
				}
				tv, ok := a.pkg.TypesInfo.Types[stmt.Lhs[i]]
				if !ok {
					panic(errf("no type info for lvalue %d in assignment at %s", i, a.pos(stmt)))
				}
				intf := getInterface(tv.Type)
				if intf == nil {
					return false
				}
				a.addMethods(intf)
				continue
			}
			if !a.expr(rhs) {
				return false
			}
		}
		return true

	case *ast.BlockStmt:
		for _, s := range stmt.List {
			if !a.stmt(s) {
				return false
			}
		}
		return true

	case *ast.BranchStmt:
		return true

	case *ast.CaseClause:
		for _, expr := range stmt.List {
			if a.isObj(expr) {
				if a.enclosingSwitchStmt == nil {
					panic(errf("case clause with no enclosing switch statement at %s", a.pos(stmt)))
				}
				if a.enclosingSwitchStmt.Tag == nil {
					return false // would require our obj to evaluate as a boolean
				}
				tv, ok := a.pkg.TypesInfo.Types[a.enclosingSwitchStmt.Tag]
				if !ok {
					panic(errf("no type info for switch tag at %s", a.pos(a.enclosingSwitchStmt.Tag)))
				}
				t1, t2 := a.obj.Type(), tv.Type
				if !types.AssignableTo(t1, t2) && !types.AssignableTo(t2, t1) {
					// "In any comparison, the first operand must be assignable to the type of the second operand, or vice versa."
					// https://go.dev/ref/spec#Comparison_operators
					return false
				}
				continue
			}

			if !a.expr(expr) {
				return false
			}
		}
		for _, s := range stmt.Body {
			if !a.stmt(s) {
				return false
			}
		}
		return true

	case *ast.CommClause:
		if !a.stmt(stmt.Comm) {
			return false
		}
		for _, s := range stmt.Body {
			if !a.stmt(s) {
				return false
			}
		}
		return true

	case *ast.DeclStmt:
		return a.decl(stmt.Decl)

	case *ast.DeferStmt:
		return a.expr(stmt.Call)

	case *ast.ExprStmt:
		if a.isObj(stmt.X) {
			// This probably can't happen in a well-formed program.
			return false
		}
		return a.expr(stmt.X)

	case *ast.ForStmt:
		if !a.stmt(stmt.Init) {
			return false
		}
		if a.isObj(stmt.Cond) {
			return false
		}
		if !a.expr(stmt.Cond) {
			return false
		}
		if !a.stmt(stmt.Post) {
			return false
		}
		return a.stmt(stmt.Body)

	case *ast.GoStmt:
		return a.expr(stmt.Call)

	case *ast.IfStmt:
		if !a.stmt(stmt.Init) {
			return false
		}
		if a.isObj(stmt.Cond) {
			return false
		}
		if !a.expr(stmt.Cond) {
			return false
		}
		if !a.stmt(stmt.Body) {
			return false
		}
		return a.stmt(stmt.Else)

	case *ast.IncDecStmt:
		if a.isObj(stmt.X) {
			return false
		}
		return a.expr(stmt.X)

	case *ast.LabeledStmt:
		return a.stmt(stmt.Stmt)

	case *ast.RangeStmt:
		// As with AssignStmt,
		// if our object appears on the lhs we don't care.
		if a.isObj(stmt.X) {
			return false
		}
		if !a.expr(stmt.X) {
			return false
		}
		return a.stmt(stmt.Body)

	case *ast.ReturnStmt:
		for i, expr := range stmt.Results {
			if a.isObj(expr) {
				typ, fpos, ok := a.enclosingFuncInfo()
				if !ok {
					panic(errf("no type info for function containing return statement at %s", a.pos(expr)))
				}
				sig, ok := typ.(*types.Signature)
				if !ok {
					panic(errf("got %T, want *types.Signature for type of function at %s", typ, fpos))
				}
				if i >= sig.Results().Len() {
					panic(errf("cannot return %d value(s) from %d-value-returning function at %s", i+1, sig.Results().Len(), a.pos(stmt)))
				}
				resultvar := sig.Results().At(i)
				intf := getInterface(resultvar.Type())
				if intf == nil {
					return false
				}
				a.addMethods(intf)
				continue
			}
			if !a.expr(expr) {
				return false
			}
		}
		return true

	case *ast.SelectStmt:
		return a.stmt(stmt.Body)

	case *ast.SendStmt:
		if a.isObj(stmt.Chan) {
			return false
		}
		if !a.expr(stmt.Chan) {
			return false
		}
		if a.isObj(stmt.Value) {
			tv, ok := a.pkg.TypesInfo.Types[stmt.Chan]
			if !ok {
				panic(errf("no type info for channel in send statement at %s", a.pos(stmt)))
			}
			chtyp := getChanType(tv.Type)
			if chtyp == nil {
				panic(errf("got %T, want channel for type of channel in send statement at %s", tv.Type, a.pos(stmt)))
			}
			intf := getInterface(chtyp.Elem())
			if intf == nil {
				return false
			}
			a.addMethods(intf)
			return true
		}
		return a.expr(stmt.Value)

	case *ast.SwitchStmt:
		return a.switchStmt(stmt)

	case *ast.TypeSwitchStmt:
		if !a.stmt(stmt.Init) {
			return false
		}
		// Can skip stmt.Assign.
		return a.stmt(stmt.Body)
	}

	return false
}

func (a *analyzer) pos(p interface{ Pos() token.Pos }) token.Position {
	return a.pkg.Fset.Position(p.Pos())
}

type methoder interface {
	NumMethods() int
	Method(int) *types.Func
}

func (a *analyzer) addMethods(intf methoder) {
	addMethodsToMap(intf, a.methods)
}

func addMethodsToMap(intf methoder, mm MethodMap) {
	for i := 0; i < intf.NumMethods(); i++ {
		m := intf.Method(i)

		// m is a *types.Func, and the Type() of a *types.Func is always *types.Signature.
		mm[m.Name()] = m.Type().(*types.Signature)
	}
}

func (a *analyzer) expr(expr ast.Expr) (ok bool) {
	a.level++
	a.debugf("> expr %#v", expr)
	defer func() {
		a.debugf("< expr %#v %v", expr, ok)
		a.level--
	}()

	if expr == nil {
		return true
	}

	switch expr := expr.(type) {
	case *ast.BinaryExpr:
		var other ast.Expr
		if a.isObj(expr.X) {
			other = expr.Y
		} else if a.isObj(expr.Y) {
			other = expr.X
		}
		if other != nil {
			switch expr.Op {
			case token.EQL, token.NEQ:
				if a.isObj(other) {
					return true
				}
				tv, ok := a.pkg.TypesInfo.Types[other]
				if !ok {
					panic(errf("no type info for expr at %s", a.pos(other)))
				}
				intf := getInterface(tv.Type)
				if intf == nil {
					return false
				}
				a.addMethods(intf)
				// Continue below.

			default:
				return false
			}
		}

		return a.expr(expr.X) && a.expr(expr.Y)

	case *ast.CallExpr:
		if a.isObj(expr.Fun) {
			return false
		}
		if !a.expr(expr.Fun) {
			return false
		}
		for i, arg := range expr.Args {
			if a.isObj(arg) {
				if i == len(expr.Args)-1 && expr.Ellipsis != token.NoPos {
					// This is "obj..." using our object, requiring it to be a slice.
					return false
				}
				tv, ok := a.pkg.TypesInfo.Types[expr.Fun]
				if !ok {
					panic(errf("no type info for function in call expression at %s", a.pos(expr)))
				}
				sig := getSig(tv.Type)
				if sig == nil {
					// This could be a type conversion expression; e.g. int(x).
					if len(expr.Args) == 1 {
						return false
					}
					panic(errf("got %T, want *types.Signature for type of function in call expression at %s", tv.Type, a.pos(expr)))
				}
				var (
					params = sig.Params()
					plen   = params.Len()
					ptype  types.Type
				)
				if sig.Variadic() && i >= plen-1 {
					ptype = params.At(plen - 1).Type()
					slice, ok := ptype.(*types.Slice)
					if !ok {
						panic(errf("got %T, want slice for type of final parameter of variadic function in call expression at %s", ptype, a.pos(expr)))
					}
					ptype = slice.Elem()
				} else if i >= plen {
					panic(errf("cannot send %d argument(s) to %d-parameter function in call expression at %s", i+1, plen, a.pos(expr)))
				} else {
					ptype = params.At(i).Type()
				}
				intf := getInterface(ptype)
				if intf == nil {
					return false
				}
				a.addMethods(intf)
				continue
			}
			if !a.expr(arg) {
				return false
			}
		}
		return true

	case *ast.CompositeLit:
		// Can skip expr.Type.
		for i, elt := range expr.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				if a.isObj(kv.Key) {
					tv, ok := a.pkg.TypesInfo.Types[expr]
					if !ok {
						panic(errf("no type info for composite literal at %s", a.pos(expr)))
					}
					mapType := getMap(tv.Type)
					if mapType == nil {
						return false
					}
					intf := getInterface(mapType.Key())
					if intf == nil {
						return false
					}
					a.addMethods(intf)
				} else if !a.expr(kv.Key) {
					return false
				}
				if a.isObj(kv.Value) {
					tv, ok := a.pkg.TypesInfo.Types[expr]
					if !ok {
						panic(errf("no type info for composite literal at %s", a.pos(expr)))
					}

					literalType := tv.Type
					if named, ok := literalType.(*types.Named); ok { // xxx should this be a loop?
						literalType = named.Underlying()
					}

					var elemType types.Type

					switch literalType := literalType.(type) {
					case *types.Map:
						elemType = literalType.Elem()

					case *types.Struct:
						id := getIdent(kv.Key)
						if id == nil {
							panic(errf("got %T, want *ast.Ident in key-value entry of struct-typed composite literal at %s", kv.Key, a.pos(kv)))
						}

						for j := 0; j < literalType.NumFields(); j++ {
							field := literalType.Field(j)
							if field.Name() == id.Name {
								elemType = field.Type()
								break
							}
						}
						if elemType == nil {
							panic(errf("assignment to unknown struct field %s at %s", id.Name, a.pos(kv)))
						}

					case *types.Slice:
						elemType = literalType.Elem()

					case *types.Array:
						elemType = literalType.Elem()

					default:
						return false
					}

					intf := getInterface(elemType)
					if intf == nil {
						return false
					}
					a.addMethods(intf)

				} else if !a.expr(kv.Value) {
					return false
				}
				continue
			}
			if a.isObj(elt) {
				tv, ok := a.pkg.TypesInfo.Types[expr]
				if !ok {
					panic(errf("no type info for composite literal at %s", a.pos(expr)))
				}

				literalType := tv.Type
				if named, ok := literalType.(*types.Named); ok { // xxx should this be a loop?
					literalType = named.Underlying()
				}

				var elemType types.Type

				switch literalType := literalType.(type) {
				case *types.Struct:
					if i >= literalType.NumFields() {
						panic(errf("cannot assign field %d of %d-field struct at %s", i, literalType.NumFields(), a.pos(elt)))
					}
					elemType = literalType.Field(i).Type()

				case *types.Slice:
					elemType = literalType.Elem()

				case *types.Array:
					elemType = literalType.Elem()
				}

				intf := getInterface(elemType)
				if intf == nil {
					return false
				}
				a.addMethods(intf)

				continue
			}
			if !a.expr(elt) {
				return false
			}
		}
		return true

	case *ast.Ellipsis:
		if a.isObj(expr.Elt) {
			return false
		}
		return a.expr(expr.Elt)

	case *ast.FuncLit:
		return a.funcLit(expr)

	case *ast.Ident:
		return true

	case *ast.IndexExpr:
		if a.isObj(expr.X) {
			return false
		}
		if !a.expr(expr.X) {
			return false
		}
		if a.isObj(expr.Index) {
			// In expression x[index],
			// index can be an interface
			// if x is a map.
			tv, ok := a.pkg.TypesInfo.Types[expr.X]
			if !ok {
				panic(errf("no type info for index expression at %s", a.pos(expr)))
			}
			mapType := getMap(tv.Type)
			if mapType == nil {
				return false
			}
			intf := getInterface(mapType.Key())
			if intf == nil {
				return false
			}
			a.addMethods(intf)
			return true
		}
		return a.expr(expr.Index)

	case *ast.IndexListExpr:
		if a.isObj(expr.X) {
			return false
		}
		if !a.expr(expr.X) {
			return false
		}
		for _, idx := range expr.Indices {
			if a.isObj(idx) {
				return false
			}
			if !a.expr(idx) {
				return false
			}
		}
		return true

	case *ast.KeyValueExpr:
		panic("did not expect to reach the KeyValueExpr clause")

	case *ast.ParenExpr:
		return a.expr(expr.X)

	case *ast.SelectorExpr:
		if a.isObj(expr.X) {
			if sig := a.getSig(expr); sig != nil {
				a.methods[expr.Sel.Name] = sig
				return true
			}
			return false
		}
		return a.expr(expr.X)

	case *ast.SliceExpr:
		if a.isObj(expr.X) {
			return false
		}
		if !a.expr(expr.X) {
			return false
		}
		if a.isObj(expr.Low) {
			return false
		}
		if !a.expr(expr.Low) {
			return false
		}
		if a.isObj(expr.High) {
			return false
		}
		if !a.expr(expr.High) {
			return false
		}
		if a.isObj(expr.Max) {
			return false
		}
		return a.expr(expr.Max)

	case *ast.StarExpr:
		if a.isObj(expr.X) {
			return false
		}
		return a.expr(expr.X)

	case *ast.TypeAssertExpr:
		// Can skip expr.Type.
		return a.expr(expr.X)

	case *ast.UnaryExpr:
		if a.isObj(expr.X) {
			return expr.Op == token.AND
		}
		return a.expr(expr.X)
	}

	return true
}

func (a *analyzer) decl(decl ast.Decl) bool {
	switch decl := decl.(type) {
	case *ast.GenDecl:
		if decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			valspec, ok := spec.(*ast.ValueSpec)
			if !ok {
				panic(errf("got %T, want *ast.ValueSpec in variable declaration at %s", spec, a.pos(decl)))
			}
			for _, val := range valspec.Values {
				if a.isObj(val) {
					if valspec.Type == nil {
						continue
					}
					tv, ok := a.pkg.TypesInfo.Types[valspec.Type]
					if !ok {
						panic(errf("no type info for variable declaration at %s", a.pos(valspec)))
					}
					intf := getInterface(tv.Type)
					if intf == nil {
						return false
					}
					a.addMethods(intf)
					continue
				}
				if !a.expr(val) {
					return false
				}
			}
		}
		return true

	case *ast.FuncDecl:
		outer := a.enclosingFunc
		a.enclosingFunc = &funcDeclOrLit{decl: decl}
		defer func() { a.enclosingFunc = outer }()

		return a.stmt(decl.Body)

	default:
		return true
	}
}

func (a *analyzer) funcLit(expr *ast.FuncLit) bool {
	outer := a.enclosingFunc
	a.enclosingFunc = &funcDeclOrLit{lit: expr}
	defer func() {
		a.enclosingFunc = outer
	}()

	return a.stmt(expr.Body)
}

func (a *analyzer) switchStmt(stmt *ast.SwitchStmt) bool {
	outer := a.enclosingSwitchStmt
	a.enclosingSwitchStmt = stmt
	defer func() {
		a.enclosingSwitchStmt = outer
	}()

	if !a.stmt(stmt.Init) {
		return false
	}
	// It's OK if stmt.Tag is our object.
	if !a.expr(stmt.Tag) {
		return false
	}
	return a.stmt(stmt.Body)
}

func getIdent(expr ast.Expr) *ast.Ident {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr
	case *ast.ParenExpr:
		return getIdent(expr.X)
	default:
		return nil
	}
}

func isInternal(path string) bool {
	parts := strings.Split(path, "/")
	return slices.Contains(parts, "internal")
}

func sameMethodMaps(a, b MethodMap) bool {
	if len(a) != len(b) {
		return false
	}
	for name, asig := range a {
		bsig, ok := b[name]
		if !ok {
			return false
		}
		if !types.Identical(asig, bsig) {
			return false
		}
	}
	return true
}
