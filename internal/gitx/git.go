// Package gitx provides a thin abstraction over git CLI operations.
package gitx

import "context"

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
}
