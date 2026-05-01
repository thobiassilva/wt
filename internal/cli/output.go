package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// ANSI escape sequences used to colorize output. Kept as package-level
// constants so tests can assert on them.
const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiBold   = "\x1b[1m"
)

// Output is a small helper that mirrors the bash `info`/`warn`/`die` helpers
// in the original `wt` script. It is designed for testability: writers and
// the process-exit function are injected, so tests never touch the real
// terminal or terminate the test binary.
type Output struct {
	// Stdout receives Info and Section output.
	Stdout io.Writer
	// Stderr receives Warn, Error and Die output.
	Stderr io.Writer
	// NoColor disables ANSI color sequences when true.
	NoColor bool
	// Exit is called by Die. Defaults to os.Exit. Tests inject a fake.
	Exit func(int)
}

// NewOutput builds an Output wired to the real process streams. It enables
// colors only when both stdout and stderr are TTYs and NO_COLOR is not set.
func NewOutput() *Output {
	noColor := false

	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		noColor = true
	}

	if !term.IsTerminal(int(os.Stdout.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		noColor = true
	}

	return &Output{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		NoColor: noColor,
		Exit:    os.Exit,
	}
}

// color wraps s in the given ANSI color when colors are enabled, otherwise
// returns s unchanged.
func (o *Output) color(code, s string) string {
	if o.NoColor {
		return s
	}
	return code + s + ansiReset
}

// Info prints a green ">>>" prefix followed by the formatted message to
// stdout, with a trailing newline. Mirrors bash `info`.
func (o *Output) Info(format string, args ...any) {
	prefix := o.color(ansiGreen, ">>>")
	fmt.Fprintf(o.Stdout, "%s %s\n", prefix, fmt.Sprintf(format, args...))
}

// Warn prints a yellow "aviso:" prefix followed by the formatted message to
// stderr, with a trailing newline. Mirrors bash `warn`.
func (o *Output) Warn(format string, args ...any) {
	prefix := o.color(ansiYellow, "aviso:")
	fmt.Fprintf(o.Stderr, "%s %s\n", prefix, fmt.Sprintf(format, args...))
}

// Error prints a red "erro:" prefix followed by the formatted message to
// stderr. Unlike Die, it does NOT terminate the process — the caller decides
// whether to exit, return, or continue.
func (o *Output) Error(format string, args ...any) {
	prefix := o.color(ansiRed, "erro:")
	fmt.Fprintf(o.Stderr, "%s %s\n", prefix, fmt.Sprintf(format, args...))
}

// Die prints an error message and exits with status 1. Mirrors bash `die`.
// The exit function is taken from o.Exit so tests can substitute it.
func (o *Output) Die(format string, args ...any) {
	o.Error(format, args...)
	exit := o.Exit
	if exit == nil {
		exit = os.Exit
	}
	exit(1)
}

// Section prints a bold/cyan divider with the given title above and below,
// to stdout. Used to visually separate summary blocks.
func (o *Output) Section(title string) {
	line := fmt.Sprintf("--- %s ---", title)
	fmt.Fprintf(o.Stdout, "\n%s\n", o.color(ansiBold+ansiCyan, line))
}
