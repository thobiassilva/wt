// Package fsx provides a thin filesystem abstraction for testability.
package fsx

import "os"

// FS is the contract for filesystem operations needed by wt.
// Implementations: realFS (os) for production, afero.MemMapFs adapter for tests.
type FS interface {
	Stat(path string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	CopyFile(src, dst string) error
	ReadFile(path string) ([]byte, error)
	Exists(path string) bool
}
