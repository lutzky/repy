package writerlogger

import (
	"fmt"
	"io"

	"github.com/lutzky/repy"
)

var _ repy.Logger = Logger{}

// Logger is a repy.Logger that writes to W, an io.Writer
type Logger struct {
	W io.Writer
}

// Infof implements Logger.Infof
func (wl Logger) Infof(format string, args ...interface{}) {
	fmt.Fprint(wl.W, "I ")
	fmt.Fprintf(wl.W, format, args...)
	fmt.Fprint(wl.W, "\n")
}

// Warningf implements Logger.Warningf
func (wl Logger) Warningf(format string, args ...interface{}) {
	fmt.Fprint(wl.W, "W ")
	fmt.Fprintf(wl.W, format, args...)
	fmt.Fprint(wl.W, "\n")
}

// Flush implements Logger.Flush
func (wl Logger) Flush() {}
