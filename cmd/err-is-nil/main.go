package main

import (
	errisnil "github.com/prashantv/err-is-nil"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(errisnil.Analyzer) }
