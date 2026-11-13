package main

import (
	"fmt"
	"io"
)

func runUpgrade(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "nerv-task upgrade: not implemented yet")
	return 2
}
