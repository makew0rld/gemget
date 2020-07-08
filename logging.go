package main

import (
	"fmt"
	"os"
	"strings"
)

func fatal(format string, a ...interface{}) {
	urlError(format, a...)
	os.Exit(1)
}

func urlError(format string, a ...interface{}) {
	format = "Error: " + strings.TrimRight(format, "\n") + "\n"
	fmt.Fprintf(os.Stderr, format, a...)
	if !*errorSkip {
		os.Exit(1)
	}
}

func info(format string, a ...interface{}) {
	if quiet {
		return
	}
	format = "Info: " + strings.TrimRight(format, "\n") + "\n"
	fmt.Fprintf(os.Stderr, format, a...)
}
