package main

import (
	"os"

	"github.com/lavantien/go-sudoku-solver/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
