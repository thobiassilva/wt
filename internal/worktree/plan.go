// Package worktree contains the domain logic for creating git worktrees
// and copying files specified in .worktreeinclude.
package worktree

// Plan is the resolved intent of a `wt` invocation, shared by dry-run and execution.
// It is built by Service.BuildPlan and consumed by Service.Execute.
type Plan struct {
	Branch         string   // target branch name
	Base           string   // base branch (used only if Branch must be created)
	Dest           string   // destination directory for the worktree
	BranchExists   bool     // true if Branch already exists in the repo
	NoInclude      bool     // skip .worktreeinclude copy
	FilesToCopy    []string // resolved files relative to repo root, ordered
	RepoRoot       string   // absolute path of the repo root
}

// Options is the user-facing input from CLI flags.
type Options struct {
	Branch     string
	Name       string
	Base       string
	PathPrefix string
	NoInclude  bool
	DryRun     bool
}
