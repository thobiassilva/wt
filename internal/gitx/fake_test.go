package gitx

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeGit_NewFakeInitializesMaps(t *testing.T) {
	f := NewFake()
	require.NotNil(t, f)
	assert.NotNil(t, f.Branches)
	assert.NotNil(t, f.LsIgnoredOutput)
	assert.NotNil(t, f.RefFormatErrors)
	assert.NotNil(t, f.Created)
	assert.NotNil(t, f.Worktrees)
}

func TestFakeGit_RepoRoot(t *testing.T) {
	f := NewFake()
	f.RepoRootValue = "/tmp/repo"
	got, err := f.RepoRoot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/tmp/repo", got)
}

func TestFakeGit_CurrentBranch(t *testing.T) {
	f := NewFake()
	f.CurrentBranchValue = "main"
	got, err := f.CurrentBranch(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "main", got)
}

func TestFakeGit_BranchExists(t *testing.T) {
	f := NewFake()
	f.Branches["main"] = true

	exists, err := f.BranchExists(context.Background(), "main")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = f.BranchExists(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestFakeGit_CreateBranch_RecordsAndMarksExisting(t *testing.T) {
	f := NewFake()
	require.NoError(t, f.CreateBranch(context.Background(), "feature/x", "main"))
	assert.Equal(t, []string{"feature/x"}, f.Created)
	assert.True(t, f.Branches["feature/x"])
}

func TestFakeGit_CreateBranch_ForcedError(t *testing.T) {
	f := NewFake()
	boom := errors.New("boom")
	f.CreateBranchErr = boom
	err := f.CreateBranch(context.Background(), "feature/x", "main")
	assert.ErrorIs(t, err, boom)
	assert.Empty(t, f.Created)
	assert.False(t, f.Branches["feature/x"])
}

func TestFakeGit_WorktreeAdd_RecordsCalls(t *testing.T) {
	f := NewFake()
	require.NoError(t, f.WorktreeAdd(context.Background(), "../wt-feature", "feature/x"))
	require.NoError(t, f.WorktreeAdd(context.Background(), "../wt-bug", "bugfix/y"))
	assert.Equal(t, []WorktreeRecord{
		{Dest: "../wt-feature", Branch: "feature/x"},
		{Dest: "../wt-bug", Branch: "bugfix/y"},
	}, f.Worktrees)
}

func TestFakeGit_WorktreeAdd_ForcedError(t *testing.T) {
	f := NewFake()
	boom := errors.New("boom")
	f.WorktreeAddErr = boom
	err := f.WorktreeAdd(context.Background(), "../x", "x")
	assert.ErrorIs(t, err, boom)
	assert.Empty(t, f.Worktrees)
}

func TestFakeGit_LsIgnored_DefaultStarKey(t *testing.T) {
	f := NewFake()
	f.LsIgnoredOutput["*"] = []string{".env", "secrets/key"}
	got, err := f.LsIgnored(context.Background(), "/repo", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{".env", "secrets/key"}, got)
}

func TestFakeGit_LsIgnored_PathScoped(t *testing.T) {
	f := NewFake()
	f.LsIgnoredOutput["secrets"] = []string{"secrets/key"}
	got, err := f.LsIgnored(context.Background(), "/repo", []string{"secrets"})
	require.NoError(t, err)
	assert.Equal(t, []string{"secrets/key"}, got)
}

func TestFakeGit_LsIgnored_MissingKeyReturnsNil(t *testing.T) {
	f := NewFake()
	got, err := f.LsIgnored(context.Background(), "/repo", []string{"unknown"})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFakeGit_CheckRefFormat(t *testing.T) {
	f := NewFake()
	bad := errors.New("invalid")
	f.RefFormatErrors["bad..name"] = bad

	assert.ErrorIs(t, f.CheckRefFormat(context.Background(), "bad..name"), bad)
	assert.NoError(t, f.CheckRefFormat(context.Background(), "feature/ok"))
}

func TestFakeGit_ImplementsGit(t *testing.T) {
	var _ Git = NewFake()
}
