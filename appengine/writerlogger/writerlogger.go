package writerlogger

import (
	"fmt"
	"io"
)

type Logger struct {
	W io.Writer
}

func (wl Logger) Infof(format string, args ...interface{}) {
	fmt.Fprint(wl.W, "I ")
	fmt.Fprintf(wl.W, format, args...)
	fmt.Fprint(wl.W, "\n")
}

func (wl Logger) Warningf(format string, args ...interface{}) {
	fmt.Fprint(wl.W, "W ")
	fmt.Fprintf(wl.W, format, args...)
	fmt.Fprint(wl.W, "\n")
}

func (wl Logger) Flush() {}
