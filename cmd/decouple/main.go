package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/bobg/go-generics/v2/maps"

	"github.com/bobg/decouple"
)

func main() {
	var verbose, interfaces bool
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&interfaces, "interfaces", false, "check interface-typed function parameters too")
	flag.Parse()

	var dir string
	switch flag.NArg() {
	case 0:
		dir = "."
	case 1:
		dir = flag.Arg(0)
	default:
		fmt.Fprintf(os.Stderr, "Usage: %s [-v] [DIR]\n", os.Args[0])
		os.Exit(1)
	}

	checker, err := decouple.NewCheckerFromDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	checker.Verbose = verbose
	checker.Interfaces = interfaces

	tuples, err := checker.Check()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sort.Slice(tuples, func(i, j int) bool {
		iPos, jPos := tuples[i].Pos(), tuples[j].Pos()
		if iPos.Filename < jPos.Filename {
			return true
		}
		if iPos.Filename > jPos.Filename {
			return false
		}
		return iPos.Offset < jPos.Offset
	})

	for _, tuple := range tuples {
		var showedFuncName bool

		params := maps.Keys(tuple.M)
		sort.Strings(params)
		for _, param := range params {
			mm := tuple.M[param]
			if len(mm) == 0 {
				continue
			}

			if !showedFuncName {
				fmt.Printf("%s: %s\n", tuple.Pos(), tuple.F.Name.Name)
				showedFuncName = true
			}

			if intfName := checker.NameForMethods(mm); intfName != "" {
				fmt.Printf("    %s: %s\n", param, intfName)
				continue
			}

			methods := maps.Keys(tuple.M[param])
			sort.Strings(methods)
			fmt.Printf("    %s: %v\n", param, methods)
		}
	}
}
