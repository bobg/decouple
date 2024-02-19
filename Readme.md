# Decouple - find overspecified function parameters in Go code

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/decouple.svg)](https://pkg.go.dev/github.com/bobg/decouple)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/decouple)](https://goreportcard.com/report/github.com/bobg/decouple)
[![Tests](https://github.com/bobg/decouple/actions/workflows/go.yml/badge.svg)](https://github.com/bobg/decouple/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/bobg/decouple/badge.svg?branch=main)](https://coveralls.io/github/bobg/decouple?branch=main)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

This is decouple,
a Go package and command that analyzes your Go code
to find “overspecified” function parameters.

A parameter is overspecified,
and eligible for “decoupling,”
if it has a more-specific type than it actually needs.

For example,
if your function takes a `*os.File` parameter,
but it’s only ever used for its `Read` method,
it could be specified as an abstract `io.Reader` instead.

## Why decouple?

When you decouple a function parameter from its too-specific type,
you broaden the set of values on which it can operate.

You also make it easier to test.
For a simple example,
suppose you’re testing this function:

```go
func CountLines(f *os.File) (int, error) {
  var result int
  sc := bufio.NewScanner(f)
  for sc.Scan() {
    result++
  }
  return result, sc.Err()
}
```

Your unit test will need to open a testdata file and pass it to this function to get a result.
But as `decouple` can tell you,
`f` is only ever used as an `io.Reader`
(the type of the argument to [bufio.NewScanner](https://pkg.go.dev/bufio#NewScanner)).

If you were testing `func CountLines(r io.Reader) (int, error)` instead,
the unit test can simply pass it something like `strings.NewReader("a\nb\nc")`.

## Installation

```sh
go install github.com/bobg/decouple/cmd/decouple@latest
```

## Usage

```sh
decouple [-v] [-json] [DIR]
```

This produces a report about the Go packages rooted at DIR
(the current directory by default).
With -v,
very verbose debugging output is printed along the way.
With -json,
the output is in JSON format.

The report will be empty if decouple has no findings.
Otherwise, it will look something like this (without -json):

```
$ decouple
/home/bobg/kodigcs/handle.go:105:18: handleDir
    req: [Context]
    w: io.Writer
/home/bobg/kodigcs/handle.go:167:18: handleNFO
    req: [Context]
    w: [Header Write]
/home/bobg/kodigcs/handle.go:428:6: isStale
    t: [Before]
/home/bobg/kodigcs/imdb.go:59:6: parseIMDbPage
    cl: [Do]
```

This is the output when running decouple on [the current commit](https://github.com/bobg/kodigcs/commit/f4e8cf0e44de0ea98fa7ad4f88705324ff446444)
of [kodigcs](https://github.com/bobg/kodigcs).
It’s saying that:

- In the function [handleDir](https://github.com/bobg/kodigcs/blob/f4e8cf0e44de0ea98fa7ad4f88705324ff446444/handle.go#L105),
  the `req` parameter is being used only for its `Context` method
  and so could be declared as `interface{ Context() context.Context }`,
  allowing objects other than `*http.Request` values to be passed in here
  (or, better still, the function could be rewritten to take a `context.Context` parameter instead);
- Also in [handleDir](https://github.com/bobg/kodigcs/blob/f4e8cf0e44de0ea98fa7ad4f88705324ff446444/handle.go#L105),
  `w` could be an `io.Writer`,
  allowing more types to be used than just `http.ResponseWriter`;
- Similarly in [handleNFO](https://github.com/bobg/kodigcs/blob/f4e8cf0e44de0ea98fa7ad4f88705324ff446444/handle.go#L167),
  `req` is used only for its `Context` method,
  and `w` for its `Write` and `Header` methods
  (more than `io.Writer`, but less than `http.ResponseWriter`);
- Anything with a `Before(time.Time) bool` method
  could be used in [isStale](https://github.com/bobg/kodigcs/blob/f4e8cf0e44de0ea98fa7ad4f88705324ff446444/handle.go#L428),
  it does not need to be limited to `time.Time`;
- The `*http.Client` argument of [parseIMDbPage](https://github.com/bobg/kodigcs/blob/f4e8cf0e44de0ea98fa7ad4f88705324ff446444/imdb.go#L59)
  is being used only for its `Do` method.

Note that,
in the report,
the presence of square brackets means “this is a set of methods,”
while the absence of them means “this is an existing type that already has the right method set”
(as in the `io.Writer` line in the example above).
Decouple can’t always find a suitable existing type even when one exists,
and if two or more types match,
it doesn’t always choose the best one.

The same report with `-json` specified looks like this:

```
{
  "PackageName": "main",
  "FileName": "/home/bobg/kodigcs/handle.go",
  "Line": 105,
  "Column": 18,
  "FuncName": "handleDir",
  "Params": [
    {
      "Name": "req",
      "Methods": [
        "Context"
      ]
    },
    {
      "Name": "w",
      "Methods": [
        "Write"
      ],
      "InterfaceName": "io.Writer"
    }
  ]
}
{
  "PackageName": "main",
  "FileName": "/home/bobg/kodigcs/handle.go",
  "Line": 167,
  "Column": 18,
  "FuncName": "handleNFO",
  "Params": [
    {
      "Name": "req",
      "Methods": [
        "Context"
      ]
    },
    {
      "Name": "w",
      "Methods": [
        "Header",
        "Write"
      ]
    }
  ]
}
{
  "PackageName": "main",
  "FileName": "/home/bobg/kodigcs/handle.go",
  "Line": 428,
  "Column": 6,
  "FuncName": "isStale",
  "Params": [
    {
      "Name": "t",
      "Methods": [
        "Before"
      ]
    }
  ]
}
{
  "PackageName": "main",
  "FileName": "/home/bobg/kodigcs/imdb.go",
  "Line": 59,
  "Column": 6,
  "FuncName": "parseIMDbPage",
  "Params": [
    {
      "Name": "cl",
      "Methods": [
        "Do"
      ]
    }
  ]
}
```

## Performance note

Replacing overspecified function parameters with more-abstract ones,
which this tool helps you to do,
is often but not always the right thing,
and it should not be done blindly.

Using Go interfaces can impose an _abstraction penalty_ compared to using concrete types.
Function arguments that could have been on [the stack](https://en.wikipedia.org/wiki/Stack-based_memory_allocation)
may end up in [the heap](https://en.wikipedia.org/wiki/Heap-based_memory_allocation),
and method calls may involve [a virtual-dispatch step](https://en.wikipedia.org/wiki/Dynamic_dispatch).

In many cases this penalty is small and can be ignored,
especially since the Go compiler may optimize some or all of it away.
But in tight inner loops
and other performance-critical code
it is often preferable to operate only on concrete types when possible.

That said,
avoid the fallacy of [premature optimization](https://wiki.c2.com/?PrematureOptimization).
Write your code for clarity and utility first.
Then sacrifice those for the sake of performance
not in the places where you _think_ they’ll make a difference,
but in the places where you’ve _measured_ that they’re needed.
