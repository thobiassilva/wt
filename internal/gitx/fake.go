package gitx

import (
	"context"
	"strings"
)

// WorktreeRecord captures a single worktree-add invocation made on the fake.
type WorktreeRecord struct {
	Dest   string
	Branch string
}

// FakeGit is an in-memory implementation of Git used by tests in this package
// and by other packages (e.g. the worktree package). It records calls and
// returns values driven by its public fields.
type FakeGit struct {
	// State / configurable return values.
	RepoRootValue      string
	CurrentBranchValue string
	Branches           map[string]bool

	// LsIgnoredOutput is keyed by the joined paths argument (paths joined with
	// "\x00"). Use the literal key "*" to provide a default that is returned
	// when the caller passes an empty paths slice.
	LsIgnoredOutput map[string][]string

	// RefFormatErrors maps a branch name to the error CheckRefFormat should
	// return for it. A nil entry (or missing key) means "valid".
	RefFormatErrors map[string]error

	// Recorded calls.
	Created   []string
	Worktrees []WorktreeRecord

	// Forced errors for mutation methods.
	CreateBranchErr error
	WorktreeAddErr  error
}

// NewFake returns an initialized FakeGit with all maps and slices ready to use.
func NewFake() *FakeGit {
	return &FakeGit{
		Branches:        map[string]bool{},
		LsIgnoredOutput: map[string][]string{},
		RefFormatErrors: map[string]error{},
		Created:         []string{},
		Worktrees:       []WorktreeRecord{},
	}
}

// RepoRoot returns the configured RepoRootValue.
func (f *FakeGit) RepoRoot(_ context.Context) (string, error) {
	return f.RepoRootValue, nil
}

// CurrentBranch returns the configured CurrentBranchValue.
func (f *FakeGit) CurrentBranch(_ context.Context) (string, error) {
	return f.CurrentBranchValue, nil
}

// BranchExists returns whether name is present (and true) in the Branches map.
func (f *FakeGit) BranchExists(_ context.Context, name string) (bool, error) {
	return f.Branches[name], nil
}

// CreateBranch records the creation and marks the branch as existing. If
// CreateBranchErr is non-nil it is returned and no state is mutated.
func (f *FakeGit) CreateBranch(_ context.Context, name, _ string) error {
	if f.CreateBranchErr != nil {
		return f.CreateBranchErr
	}
	f.Created = append(f.Created, name)
	if f.Branches == nil {
		f.Branches = map[string]bool{}
	}
	f.Branches[name] = true
	return nil
}

// WorktreeAdd records the worktree request. If WorktreeAddErr is non-nil it
// is returned and no state is mutated.
func (f *FakeGit) WorktreeAdd(_ context.Context, dest, branch string) error {
	if f.WorktreeAddErr != nil {
		return f.WorktreeAddErr
	}
	f.Worktrees = append(f.Worktrees, WorktreeRecord{Dest: dest, Branch: branch})
	return nil
}

// LsIgnored looks up the configured output by joined paths key, falling back
// to the "*" key when paths is empty.
func (f *FakeGit) LsIgnored(_ context.Context, _ string, paths []string) ([]string, error) {
	key := lsIgnoredKey(paths)
	if out, ok := f.LsIgnoredOutput[key]; ok {
		return out, nil
	}
	return nil, nil
}

// CheckRefFormat consults RefFormatErrors for the branch name.
func (f *FakeGit) CheckRefFormat(_ context.Context, branch string) error {
	if err, ok := f.RefFormatErrors[branch]; ok {
		return err
	}
	return nil
}

func lsIgnoredKey(paths []string) string {
	if len(paths) == 0 {
		return "*"
	}
	return strings.Join(paths, "\x00")
}

// Ensure compile-time interface satisfaction.
var _ Git = (*FakeGit)(nil)
