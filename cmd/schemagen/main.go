package main

import (
	"fmt"
	"os"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR %v\n", err)
		os.Exit(1)
	}
}
