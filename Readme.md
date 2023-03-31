# Decouple - find overspecified function parameters in Go code

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/decouple.svg)](https://pkg.go.dev/github.com/bobg/decouple)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/decouple)](https://goreportcard.com/report/github.com/bobg/decouple)
[![Tests](https://github.com/bobg/decouple/actions/workflows/go.yml/badge.svg)](https://github.com/bobg/decouple/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/bobg/decouple/badge.svg?branch=master)](https://coveralls.io/github/bobg/decouple?branch=master)

This is decouple,
a Go package and command that analyzes your Go code
to find “overspecified” function parameters.

A parameter is overspecified,
and eligible for “decoupling,”
if it has a more-specific type than it actually needs.

For example,
if your function takes a `*os.File` parameter,
but it’s only ever used for its `Read` method,
it could be specified as an abstract `io.Reader` instead,
making it possible to use the function
on a much wider variety of concrete types.
