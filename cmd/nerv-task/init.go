package main

import (
	"fmt"
	"io"
)

func runInit(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "nerv-task init: not implemented yet")
	return 2
}
