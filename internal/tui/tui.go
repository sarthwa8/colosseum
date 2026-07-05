// Package tui holds small terminal-formatting helpers shared by CLI commands.
// Kept dependency-free: raw ANSI, auto-disabled when stdout isn't a terminal.
package tui

import (
	"fmt"
	"os"

	"github.com/sarthaksukhral/colosseum/internal/judge"
)

var color = isTTY(os.Stdout)

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

const (
	reset  = "\033[0m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	gray   = "\033[90m"
	bold   = "\033[1m"
)

func paint(c, s string) string {
	if !color {
		return s
	}
	return c + s + reset
}

func Bold(s string) string  { return paint(bold, s) }
func Dim(s string) string   { return paint(gray, s) }

// VerdictBadge renders a fixed-width, colored verdict tag.
func VerdictBadge(v judge.Verdict) string {
	label := fmt.Sprintf("%-3s", string(v))
	switch v {
	case judge.Accepted:
		return paint(green, label)
	case judge.WrongAnswer:
		return paint(red, label)
	case judge.TimeLimit, judge.MemoryLimit, judge.OutputLimit:
		return paint(yellow, label)
	case judge.CompileError, judge.RuntimeError:
		return paint(red, label)
	default:
		return paint(gray, label)
	}
}
