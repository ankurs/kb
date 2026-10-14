package main

import (
	"fmt"
	"os"

	"github.com/ankurs/kb/internal/cli"
)

func main() {
	if err := cli.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "kb:", err)
		os.Exit(1)
	}
}
