// Package gitx provides a thin abstraction over git CLI operations.
package gitx

import "context"

// Worktree is a single entry parsed from `git worktree list --porcelain`.
type Worktree struct {
	Path     string // absolute path
	Branch   string // short branch name (refs/heads/ stripped); empty if detached/bare
	Head     string // commit sha
	Bare     bool
	Detached bool
	Locked   bool
	Prunable bool
}

// Git is the contract for all git operations needed by the wt CLI.
// Implementations: realGit (os/exec) for production, FakeGit for tests.
type Git interface {
	RepoRoot(ctx context.Context) (string, error)
	CurrentBranch(ctx context.Context) (string, error)
	BranchExists(ctx context.Context, name string) (bool, error)
	CreateBranch(ctx context.Context, name, base string) error
	WorktreeAdd(ctx context.Context, dest, branch string) error
	// LsIgnored runs `git ls-files --others --ignored --exclude-standard --directory`
	// optionally scoped to specific paths. Used by .worktreeinclude resolution.
	LsIgnored(ctx context.Context, repoRoot string, paths []string) ([]string, error)
	CheckRefFormat(ctx context.Context, branch string) error
	// WorktreeList runs `git worktree list --porcelain` and parses the result.
	WorktreeList(ctx context.Context) ([]Worktree, error)
	// WorktreeRemove runs `git worktree remove [--force] <path>`.
	WorktreeRemove(ctx context.Context, path string, force bool) error
	// DeleteBranch runs `git branch -d <name>` (or -D when force is true).
	DeleteBranch(ctx context.Context, name string, force bool) error
}
