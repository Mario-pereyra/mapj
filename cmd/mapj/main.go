package main

import (
	"os"

	"github.com/Mario-pereyra/mapj/internal/cli"
)

func main() {
	code := cli.Execute()
	os.Exit(code)
}