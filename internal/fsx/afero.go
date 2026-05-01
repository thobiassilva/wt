package fsx

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"
)

// aferoFS implements FS on top of an afero.Fs (typically an in-memory filesystem
// for tests).
type aferoFS struct {
	Fs afero.Fs
}

// NewAfero returns an FS backed by the given afero.Fs implementation.
// Passing afero.NewMemMapFs() produces a fast, isolated in-memory filesystem
// suitable for unit tests.
func NewAfero(fs afero.Fs) FS {
	return &aferoFS{Fs: fs}
}

// Stat returns the FileInfo for path within the wrapped filesystem.
func (a *aferoFS) Stat(path string) (os.FileInfo, error) {
	return a.Fs.Stat(path)
}

// MkdirAll creates a directory along with any necessary parents.
func (a *aferoFS) MkdirAll(path string, perm os.FileMode) error {
	return a.Fs.MkdirAll(path, perm)
}

// ReadFile reads the named file from the wrapped filesystem.
func (a *aferoFS) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(a.Fs, path)
}

// Exists reports whether path exists in the wrapped filesystem.
func (a *aferoFS) Exists(path string) bool {
	ok, err := afero.Exists(a.Fs, path)
	if err != nil {
		return false
	}
	return ok
}

// CopyFile copies the contents of src to dst within the wrapped filesystem,
// preserving src's file mode.
func (a *aferoFS) CopyFile(src, dst string) (err error) {
	in, oerr := a.Fs.Open(src)
	if oerr != nil {
		return fmt.Errorf("open source %q: %w", src, oerr)
	}
	defer in.Close()

	info, serr := in.Stat()
	if serr != nil {
		return fmt.Errorf("stat source %q: %w", src, serr)
	}

	out, cerr := a.Fs.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if cerr != nil {
		return fmt.Errorf("create destination %q: %w", dst, cerr)
	}

	defer func() {
		closeErr := out.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close destination %q: %w", dst, closeErr)
		}
	}()

	if _, copyErr := io.Copy(out, in); copyErr != nil {
		return fmt.Errorf("copy %q to %q: %w", src, dst, copyErr)
	}

	return nil
}
