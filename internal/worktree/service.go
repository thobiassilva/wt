package worktree

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/thobiassilva/wt/internal/fsx"
	"github.com/thobiassilva/wt/internal/gitx"
	"github.com/thobiassilva/wt/internal/naming"
)

// Service orchestrates worktree creation. It is intentionally thin: it
// delegates git operations to gitx.Git, filesystem operations to fsx.FS,
// and name derivation to the naming package.
type Service struct {
	Git gitx.Git
	FS  fsx.FS
}

// NewService creates a Service wired to real git and filesystem implementations.
func NewService(git gitx.Git, fs fsx.FS) *Service {
	return &Service{Git: git, FS: fs}
}

// BuildPlan derives a Plan from the user Options without performing any
// side effects. Both dry-run and real execution share this path.
func (s *Service) BuildPlan(ctx context.Context, opts Options) (Plan, error) {
	repoRoot, err := s.Git.RepoRoot(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("git repo root: %w", err)
	}

	currentBranch, err := s.Git.CurrentBranch(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("current branch: %w", err)
	}

	if err := s.Git.CheckRefFormat(ctx, opts.Branch); err != nil {
		return Plan{}, fmt.Errorf("invalid branch name %q: %w", opts.Branch, err)
	}

	exists, err := s.Git.BranchExists(ctx, opts.Branch)
	if err != nil {
		return Plan{}, fmt.Errorf("check branch: %w", err)
	}

	// Derive worktree directory name.
	worktreeName := opts.Name
	if worktreeName == "" {
		worktreeName = naming.Derive(opts.Branch)
	}

	if err := naming.ValidateWorktreeName(worktreeName); err != nil {
		return Plan{}, fmt.Errorf("invalid worktree name %q: %w", worktreeName, err)
	}

	pathPrefix := opts.PathPrefix
	if pathPrefix == "" {
		pathPrefix = ".."
	}
	dest := filepath.Join(pathPrefix, worktreeName)

	// Reject destination that already exists.
	if s.FS.Exists(dest) {
		return Plan{}, fmt.Errorf("destination already exists: %s", dest)
	}

	base := opts.Base
	if base == "" {
		base = currentBranch
	}

	plan := Plan{
		Branch:       opts.Branch,
		Base:         base,
		Dest:         dest,
		BranchExists: exists,
		NoInclude:    opts.NoInclude,
		RepoRoot:     repoRoot,
	}

	if !opts.NoInclude {
		files, err := resolveIncludeFiles(ctx, repoRoot, s.Git, s.FS)
		if err != nil {
			return Plan{}, fmt.Errorf(".worktreeinclude: %w", err)
		}
		plan.FilesToCopy = files
	}

	return plan, nil
}

// Execute carries out the Plan: creates the branch if needed, adds the
// worktree, and copies .worktreeinclude files into the new worktree.
func (s *Service) Execute(ctx context.Context, plan Plan) error {
	if !plan.BranchExists {
		if err := s.Git.CreateBranch(ctx, plan.Branch, plan.Base); err != nil {
			return fmt.Errorf("create branch %q from %q: %w", plan.Branch, plan.Base, err)
		}
	}

	if err := s.Git.WorktreeAdd(ctx, plan.Dest, plan.Branch); err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}

	if !plan.NoInclude {
		if err := s.copyFiles(plan); err != nil {
			return err
		}
	}

	return nil
}

// copyFiles copies each file in plan.FilesToCopy from repoRoot to plan.Dest,
// preserving relative paths. Missing parent directories are created on demand.
func (s *Service) copyFiles(plan Plan) error {
	for _, rel := range plan.FilesToCopy {
		src := filepath.Join(plan.RepoRoot, rel)
		dst := filepath.Join(plan.Dest, rel)

		if err := s.FS.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("mkdir for %s: %w", dst, err)
		}
		if err := s.FS.CopyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", rel, err)
		}
	}
	return nil
}
