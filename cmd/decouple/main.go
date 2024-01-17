package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/bobg/go-generics/v3/maps"

	"github.com/bobg/decouple"
)

func main() {
	var (
		verbose bool
		doJSON  bool
	)
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&doJSON, "json", false, "output in JSON format")
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

	if doJSON {
		if err := showJSON(checker, tuples); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

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

func showJSON(checker decouple.Checker, tuples []decouple.Tuple) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	for _, tuple := range tuples {
		p := tuple.Pos()
		jt := jtuple{
			PackageName: tuple.P.Name,
			FileName:    p.Filename,
			Line:        p.Line,
			FuncName:    tuple.F.Name.Name,
		}
		for param, mm := range tuple.M {
			if len(mm) == 0 {
				continue
			}
			jp := jparam{
				Name:    param,
				Methods: maps.Keys(mm),
			}
			sort.Strings(jp.Methods)
			if intfName := checker.NameForMethods(mm); intfName != "" {
				jp.InterfaceName = intfName
			}
			jt.Params = append(jt.Params, jp)
		}
		if len(jt.Params) == 0 {
			continue
		}
		sort.Slice(jt.Params, func(i, j int) bool {
			return jt.Params[i].Name < jt.Params[j].Name
		})
		if err := enc.Encode(jt); err != nil {
			return err
		}
	}

	return nil
}

type jtuple struct {
	PackageName string
	FileName    string
	Line        int
	FuncName    string
	Params      []jparam
}

type jparam struct {
	Name          string
	Methods       []string `json:",omitempty"`
	InterfaceName string   `json:",omitempty"`
}
