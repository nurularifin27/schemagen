package main

import (
	"fmt"
	"io"
)

type Logger struct {
	out     io.Writer
	errOut  io.Writer
	verbose bool
	quiet   bool
}

func newLogger(out, errOut io.Writer, verbose, quiet bool) Logger {
	return Logger{
		out:     out,
		errOut:  errOut,
		verbose: verbose,
		quiet:   quiet,
	}
}

func (l Logger) Infof(format string, args ...any) {
	if l.quiet {
		return
	}
	_, _ = fmt.Fprintf(l.out, "INFO  "+format+"\n", args...)
}

func (l Logger) Verbosef(format string, args ...any) {
	if l.quiet || !l.verbose {
		return
	}
	_, _ = fmt.Fprintf(l.out, "INFO  "+format+"\n", args...)
}

func (l Logger) Warnf(format string, args ...any) {
	if l.quiet {
		return
	}
	_, _ = fmt.Fprintf(l.errOut, "WARN  "+format+"\n", args...)
}

func (l Logger) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(l.errOut, "ERROR "+format+"\n", args...)
}
