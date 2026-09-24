package main

import (
	"github.com/wrouesnel/golang-template/pkg/entrypoints"
	"os"
)

func main() {
	// The real entry point is in the entrypoint package, which allows for efficient test integration.
	// Do not add more code to this file (it should also be excluded from coverage tracking).
	exitCode := application_example.Entrypoint(ctxstdio.Set(os.Stdout, os.Stderr, os.Stdin))
	os.Exit(exitCode)
}
