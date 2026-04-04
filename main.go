// Package main is the entry point for the shu RSS aggregator CLI.
//
// shu collects RSS/Atom feeds on a schedule and stores their entries in a local
// SQLite database. It is designed with a clean separation between the CLI layer
// (cmd package), business logic (core package), and storage (store package), so
// the core logic can be reused in non-CLI contexts such as AWS Lambda.
package main

import (
	"os"

	"github.com/SuzumiyaAoba/shu/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
