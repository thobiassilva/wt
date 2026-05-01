package worktree

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thobiassilva/wt/internal/fsx"
	"github.com/thobiassilva/wt/internal/gitx"
)

func newTestService(git *gitx.FakeGit, fs fsx.FS) *Service {
	return &Service{Git: git, FS: fs}
}

func defaultFakeGit() *gitx.FakeGit {
	g := gitx.NewFake()
	g.RepoRootValue = "/repo"
	g.CurrentBranchValue = "main"
	return g
}

func TestBuildPlan_NewBranch(t *testing.T) {
	g := defaultFakeGit()
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	plan, err := svc.BuildPlan(context.Background(), Options{
		Branch:     "feature/x",
		PathPrefix: "/tmp",
	})
	require.NoError(t, err)

	assert.Equal(t, "feature/x", plan.Branch)
	assert.Equal(t, "main", plan.Base)
	assert.Equal(t, filepath.Join("/tmp", "feature-x"), plan.Dest)
	assert.False(t, plan.BranchExists)
	assert.Equal(t, "/repo", plan.RepoRoot)
}

func TestBuildPlan_ExistingBranch(t *testing.T) {
	g := defaultFakeGit()
	g.Branches["feature/x"] = true
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	plan, err := svc.BuildPlan(context.Background(), Options{
		Branch:     "feature/x",
		PathPrefix: "/tmp",
	})
	require.NoError(t, err)
	assert.True(t, plan.BranchExists)
}

func TestBuildPlan_CustomName(t *testing.T) {
	g := defaultFakeGit()
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	plan, err := svc.BuildPlan(context.Background(), Options{
		Branch:     "feature/x",
		Name:       "my-worktree",
		PathPrefix: "/tmp",
	})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/tmp", "my-worktree"), plan.Dest)
}

func TestBuildPlan_CustomBase(t *testing.T) {
	g := defaultFakeGit()
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	plan, err := svc.BuildPlan(context.Background(), Options{
		Branch:     "feature/x",
		Base:       "develop",
		PathPrefix: "/tmp",
	})
	require.NoError(t, err)
	assert.Equal(t, "develop", plan.Base)
}

func TestBuildPlan_DestExists_Error(t *testing.T) {
	g := defaultFakeGit()
	fsys, mem := makeFS(t, "/repo")
	require.NoError(t, mem.MkdirAll("/tmp/feature-x", 0o755))

	svc := newTestService(g, fsys)
	_, err := svc.BuildPlan(context.Background(), Options{
		Branch:     "feature/x",
		PathPrefix: "/tmp",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestBuildPlan_InvalidBranchName(t *testing.T) {
	g := defaultFakeGit()
	g.RefFormatErrors["bad..name"] = errors.New("invalid ref name")
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	_, err := svc.BuildPlan(context.Background(), Options{
		Branch:     "bad..name",
		PathPrefix: "/tmp",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch name")
}

func TestBuildPlan_NoInclude(t *testing.T) {
	g := defaultFakeGit()
	fs, mem := makeFS(t, "/repo")
	writeFile(t, mem, "/repo/.worktreeinclude", ".env\n")
	g.LsIgnoredOutput[".env"] = []string{".env"}

	svc := newTestService(g, fs)
	plan, err := svc.BuildPlan(context.Background(), Options{
		Branch:     "feature/x",
		PathPrefix: "/tmp",
		NoInclude:  true,
	})
	require.NoError(t, err)
	assert.Empty(t, plan.FilesToCopy)
}

func TestBuildPlan_WithInclude(t *testing.T) {
	g := defaultFakeGit()
	fs, mem := makeFS(t, "/repo")
	writeFile(t, mem, "/repo/.worktreeinclude", ".env\n")
	g.LsIgnoredOutput[".env"] = []string{".env"}

	svc := newTestService(g, fs)
	plan, err := svc.BuildPlan(context.Background(), Options{
		Branch:     "feature/x",
		PathPrefix: "/tmp",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{".env"}, plan.FilesToCopy)
}

func TestExecute_NewBranch(t *testing.T) {
	g := defaultFakeGit()
	fs, _ := makeFS(t, "/repo")
	dest := filepath.Join("/tmp", "feature-x")

	svc := newTestService(g, fs)
	plan := Plan{
		Branch:       "feature/x",
		Base:         "main",
		Dest:         dest,
		BranchExists: false,
		RepoRoot:     "/repo",
	}
	require.NoError(t, svc.Execute(context.Background(), plan))
	assert.Equal(t, []string{"feature/x"}, g.Created)
	require.Len(t, g.Worktrees, 1)
	assert.Equal(t, dest, g.Worktrees[0].Dest)
}

func TestExecute_ExistingBranch_NoBranchCreate(t *testing.T) {
	g := defaultFakeGit()
	g.Branches["feature/x"] = true
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	plan := Plan{
		Branch:       "feature/x",
		Base:         "main",
		Dest:         filepath.Join("/tmp", "feature-x"),
		BranchExists: true,
		RepoRoot:     "/repo",
	}
	require.NoError(t, svc.Execute(context.Background(), plan))
	assert.Empty(t, g.Created, "should not create branch when it already exists")
}

func TestExecute_CopiesFiles(t *testing.T) {
	g := defaultFakeGit()
	fsys, mem := makeFS(t, "/repo")
	require.NoError(t, afero.WriteFile(mem, "/repo/.env", []byte("SECRET=1\n"), 0o600))
	require.NoError(t, mem.MkdirAll("/dest/feature-x", 0o755))

	svc := newTestService(g, fsys)
	plan := Plan{
		Branch:       "feature/x",
		Base:         "main",
		Dest:         "/dest/feature-x",
		BranchExists: false,
		RepoRoot:     "/repo",
		FilesToCopy:  []string{".env"},
	}
	require.NoError(t, svc.Execute(context.Background(), plan))

	got, err := afero.ReadFile(mem, "/dest/feature-x/.env")
	require.NoError(t, err)
	assert.Equal(t, []byte("SECRET=1\n"), got)
}

func TestExecute_WorktreeAddError(t *testing.T) {
	g := defaultFakeGit()
	g.WorktreeAddErr = errors.New("worktree add failed")
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	plan := Plan{Branch: "feature/x", Base: "main", Dest: filepath.Join("/tmp", "x"), BranchExists: true, RepoRoot: "/repo"}
	err := svc.Execute(context.Background(), plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worktree add failed")
}
