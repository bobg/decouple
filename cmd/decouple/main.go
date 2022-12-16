package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/bobg/go-generics/maps"

	"github.com/bobg/decouple"
)

func main() {
	var verbose bool
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.Parse()

	var dir string
	switch flag.NArg() {
	case 0:
		dir = "."
	case 1:
		dir = flag.Arg(0)
	default:
		fmt.Fprintf(os.Stderr, "Usage: %s DIR\n", os.Args[0])
		os.Exit(1)
	}

	tuples, err := decouple.Load(context.Background(), dir, verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sort.Slice(tuples, func(i, j int) bool {
		if tuples[i].P.Filename < tuples[j].P.Filename {
			return true
		}
		if tuples[i].P.Filename > tuples[j].P.Filename {
			return false
		}
		return tuples[i].P.Offset < tuples[j].P.Offset
	})

	for _, tuple := range tuples {
		fmt.Printf("%s: %s\n", tuple.P, tuple.F.Name.Name)
		params := maps.Keys(tuple.M)
		sort.Strings(params)
		for _, param := range params {
			methods := tuple.M[param].Slice()
			sort.Strings(methods)
			fmt.Printf("    %s: %v\n", param, methods)
		}
	}
}
