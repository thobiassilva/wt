package naming

import (
	"errors"
	"strings"
)

// Validation errors returned by ValidateWorktreeName. They are exported so
// callers can use errors.Is for programmatic checks.
var (
	ErrEmptyName    = errors.New("worktree name cannot be empty")
	ErrDotDot       = errors.New("worktree name cannot contain '..'")
	ErrAbsolutePath = errors.New("worktree name cannot be an absolute path")
	ErrSpace        = errors.New("worktree name cannot contain spaces")
)

// ValidateWorktreeName checks that name is a safe worktree directory name.
//
// A name is rejected if it is empty, contains "..", starts with '/'
// (absolute path), or contains a space. Any other name is accepted; in
// particular a single '.' or non-leading '/' are allowed.
func ValidateWorktreeName(name string) error {
	if name == "" {
		return ErrEmptyName
	}
	if strings.Contains(name, "..") {
		return ErrDotDot
	}
	if strings.HasPrefix(name, "/") {
		return ErrAbsolutePath
	}
	if strings.Contains(name, " ") {
		return ErrSpace
	}
	return nil
}
