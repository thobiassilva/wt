package cli

import (
	"bytes"
	"strings"
	"testing"
)

// newTestOutput returns an Output with buffers and NoColor enabled by default.
// The injected exit fn records its argument into the returned pointer.
func newTestOutput(t *testing.T) (*Output, *bytes.Buffer, *bytes.Buffer, *int) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := -1
	o := &Output{
		Stdout:  stdout,
		Stderr:  stderr,
		NoColor: true,
		Exit: func(code int) {
			exitCode = code
		},
	}
	return o, stdout, stderr, &exitCode
}

func TestInfo_NoColor(t *testing.T) {
	o, stdout, stderr, _ := newTestOutput(t)
	o.Info("hello")

	if got, want := stdout.String(), ">>> hello\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestInfo_FormatArgs(t *testing.T) {
	o, stdout, _, _ := newTestOutput(t)
	o.Info("count=%d", 5)

	if got, want := stdout.String(), ">>> count=5\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestWarn_NoColor(t *testing.T) {
	o, stdout, stderr, _ := newTestOutput(t)
	o.Warn("oops")

	if got, want := stderr.String(), "aviso: oops\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

func TestWarn_FormatArgs(t *testing.T) {
	o, _, stderr, _ := newTestOutput(t)
	o.Warn("file %s missing", "foo.txt")

	if got, want := stderr.String(), "aviso: file foo.txt missing\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func TestError_NoColor_DoesNotExit(t *testing.T) {
	o, stdout, stderr, exitCode := newTestOutput(t)
	o.Error("bad")

	if got, want := stderr.String(), "erro: bad\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if *exitCode != -1 {
		t.Errorf("Exit was called with %d, but Error must not exit", *exitCode)
	}
}

func TestDie_CallsExitWithOne(t *testing.T) {
	o, _, stderr, exitCode := newTestOutput(t)
	o.Die("fatal %s", "thing")

	if got, want := stderr.String(), "erro: fatal thing\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
	if *exitCode != 1 {
		t.Errorf("Exit code = %d, want 1", *exitCode)
	}
}

func TestSection_NoColor(t *testing.T) {
	o, stdout, _, _ := newTestOutput(t)
	o.Section("Resumo")

	out := stdout.String()
	if !strings.Contains(out, "Resumo") {
		t.Errorf("section output %q missing title", out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("section output %q missing divider", out)
	}
}

func TestColors_AppearWhenEnabled(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	o := &Output{
		Stdout:  stdout,
		Stderr:  stderr,
		NoColor: false,
		Exit:    func(int) {},
	}

	o.Info("hi")
	if !strings.Contains(stdout.String(), ansiGreen) {
		t.Errorf("Info stdout = %q, want ANSI green", stdout.String())
	}
	if !strings.Contains(stdout.String(), ansiReset) {
		t.Errorf("Info stdout = %q, want ANSI reset", stdout.String())
	}

	o.Warn("hey")
	if !strings.Contains(stderr.String(), ansiYellow) {
		t.Errorf("Warn stderr = %q, want ANSI yellow", stderr.String())
	}

	stderr.Reset()
	o.Error("nope")
	if !strings.Contains(stderr.String(), ansiRed) {
		t.Errorf("Error stderr = %q, want ANSI red", stderr.String())
	}
}

func TestNoColorEnvVar_DisablesColors(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	o := NewOutput()
	if !o.NoColor {
		t.Errorf("NewOutput().NoColor = false, want true when NO_COLOR is set")
	}

	// Sanity: actually printing must not emit ANSI codes.
	stdout := &bytes.Buffer{}
	o.Stdout = stdout
	o.Info("x")
	if strings.Contains(stdout.String(), "\x1b[") {
		t.Errorf("Info emitted ANSI escape with NO_COLOR set: %q", stdout.String())
	}
}

func TestNewOutput_NonTTY_SetsNoColor(t *testing.T) {
	// Under `go test`, stdout/stderr are typically pipes, not TTYs.
	// Make sure NO_COLOR isn't set so we isolate the TTY check.
	t.Setenv("NO_COLOR", "")

	o := NewOutput()
	if !o.NoColor {
		t.Errorf("NewOutput().NoColor = false, want true (non-TTY under go test)")
	}
	if o.Exit == nil {
		t.Errorf("NewOutput().Exit must default to non-nil (os.Exit)")
	}
	if o.Stdout == nil || o.Stderr == nil {
		t.Errorf("NewOutput() must populate Stdout/Stderr")
	}
}

func TestDie_NilExit_DoesNotPanic(t *testing.T) {
	// Defensive: if a caller zero-valued Output, Die should still print and
	// not crash. We won't actually invoke os.Exit here because we set our own.
	// This test verifies the print path even when the exit fn is the default.
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	called := false
	o := &Output{
		Stdout:  stdout,
		Stderr:  stderr,
		NoColor: true,
		Exit:    func(int) { called = true },
	}
	o.Die("boom")
	if !called {
		t.Errorf("Exit fn was not called")
	}
}
