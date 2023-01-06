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
	debug := flag.Bool("debug", false, "turn on debugging output")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-debug] DIR\n", os.Args[0])
		os.Exit(1)
	}

	tuples, err := decouple.Load(context.Background(), flag.Arg(0), *debug)
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
