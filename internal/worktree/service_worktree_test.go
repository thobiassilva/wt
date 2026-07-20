package worktree

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thobiassilva/wt/internal/gitx"
)

func TestList_ReturnsWorktreesAndCurrent(t *testing.T) {
	g := defaultFakeGit()
	g.WorktreeListValue = []gitx.Worktree{
		{Path: "/repo", Branch: "main"},
		{Path: "/tmp/feature-x", Branch: "feature/x"},
	}
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	worktrees, current, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, worktrees, 2)
	// current is normalized to the platform's path form (e.g. "\\repo" on Windows).
	assert.Equal(t, resolvePath("/repo"), current)
}

func TestList_Error(t *testing.T) {
	g := defaultFakeGit()
	g.WorktreeListErr = errors.New("boom")
	fs, _ := makeFS(t, "/repo")

	svc := newTestService(g, fs)
	_, _, err := svc.List(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list worktrees")
}

func worktreeFakeWithList() *gitx.FakeGit {
	g := defaultFakeGit()
	g.WorktreeListValue = []gitx.Worktree{
		{Path: "/repo", Branch: "main"},
		{Path: "/tmp/feature-login-form", Branch: "feature/loginForm"},
	}
	return g
}

func TestResolveWorktree_ByBranch(t *testing.T) {
	g := worktreeFakeWithList()
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	wt, err := svc.ResolveWorktree(context.Background(), "feature/loginForm")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/feature-login-form", wt.Path)
}

func TestResolveWorktree_ByDerivedName(t *testing.T) {
	g := worktreeFakeWithList()
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	wt, err := svc.ResolveWorktree(context.Background(), "feature-login-form")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/feature-login-form", wt.Path)
}

func TestResolveWorktree_ByPath(t *testing.T) {
	g := worktreeFakeWithList()
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	wt, err := svc.ResolveWorktree(context.Background(), "/tmp/feature-login-form")
	require.NoError(t, err)
	assert.Equal(t, "feature/loginForm", wt.Branch)
}

func TestResolveWorktree_NotFound(t *testing.T) {
	g := worktreeFakeWithList()
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	_, err := svc.ResolveWorktree(context.Background(), "does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "não encontrada")
}

func TestResolveWorktree_Ambiguous(t *testing.T) {
	g := defaultFakeGit()
	// Two worktrees whose derived names collide with the argument.
	g.WorktreeListValue = []gitx.Worktree{
		{Path: "/tmp/dup", Branch: "feature/dup"},
		{Path: "/other/feature-dup", Branch: "other"},
	}
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	_, err := svc.ResolveWorktree(context.Background(), "feature-dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambíguo")
}

func TestRemoveWorktree_RemovesAndKeepsBranch(t *testing.T) {
	g := worktreeFakeWithList()
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	wt := gitx.Worktree{Path: "/tmp/feature-login-form", Branch: "feature/loginForm"}
	require.NoError(t, svc.RemoveWorktree(context.Background(), wt, false, false))
	assert.Equal(t, []string{"/tmp/feature-login-form"}, g.Removed)
	assert.Empty(t, g.DeletedBranches, "branch must be kept by default")
}

func TestRemoveWorktree_DeletesBranchWhenAsked(t *testing.T) {
	g := worktreeFakeWithList()
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	wt := gitx.Worktree{Path: "/tmp/feature-login-form", Branch: "feature/loginForm"}
	require.NoError(t, svc.RemoveWorktree(context.Background(), wt, false, true))
	assert.Equal(t, []string{"feature/loginForm"}, g.DeletedBranches)
}

func TestRemoveWorktree_RefusesCurrent(t *testing.T) {
	g := worktreeFakeWithList() // RepoRootValue == "/repo"
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	wt := gitx.Worktree{Path: "/repo", Branch: "main"}
	err := svc.RemoveWorktree(context.Background(), wt, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worktree atual")
	assert.Empty(t, g.Removed)
}

func TestRemoveWorktree_PropagatesRemoveError(t *testing.T) {
	g := worktreeFakeWithList()
	g.WorktreeRemoveErr = errors.New("dirty worktree")
	fs, _ := makeFS(t, "/repo")
	svc := newTestService(g, fs)

	wt := gitx.Worktree{Path: "/tmp/feature-login-form", Branch: "feature/loginForm"}
	err := svc.RemoveWorktree(context.Background(), wt, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dirty worktree")
}
