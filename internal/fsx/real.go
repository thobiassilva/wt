package fsx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// realFS is the production implementation of FS backed by the standard library's os package.
type realFS struct{}

// New returns the production FS implementation.
func New() FS {
	return realFS{}
}

// Stat returns the FileInfo for the given path.
func (realFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// MkdirAll creates a directory along with any necessary parents.
func (realFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// ReadFile reads the named file and returns its contents.
func (realFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Exists returns true if path exists. It is best-effort: any error other than
// fs.ErrNotExist (e.g. permission denied) results in false.
func (realFS) Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return false
}

// CopyFile copies the contents of src to dst, preserving src's file mode.
func (realFS) CopyFile(src, dst string) (err error) {
	in, oerr := os.Open(src)
	if oerr != nil {
		return fmt.Errorf("open source %q: %w", src, oerr)
	}
	defer in.Close()

	info, serr := in.Stat()
	if serr != nil {
		return fmt.Errorf("stat source %q: %w", src, serr)
	}

	out, cerr := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if cerr != nil {
		return fmt.Errorf("create destination %q: %w", dst, cerr)
	}

	// The destination close error must surface if no other error occurred,
	// since data may not be flushed until close.
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
