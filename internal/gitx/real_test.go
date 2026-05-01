//go:build integration

package gitx

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRepo initializes a fresh git repo in a temp dir, configures a user,
// makes one initial commit, and returns the repo path.
func setupRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	mustRun(t, dir, "git", "init", "-q", "-b", "main")
	mustRun(t, dir, "git", "config", "user.email", "test@example.com")
	mustRun(t, dir, "git", "config", "user.name", "Test User")
	mustRun(t, dir, "git", "config", "commit.gpgsign", "false")

	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# test\n"), 0o644))
	mustRun(t, dir, "git", "add", "README.md")
	mustRun(t, dir, "git", "commit", "-q", "-m", "initial")
	return dir
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "command %s %v failed: %s", name, args, string(out))
}

func newRealForTest(t *testing.T) *realGit {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	return &realGit{exe: "git"}
}

func TestRealGit_RepoRoot(t *testing.T) {
	g := newRealForTest(t)
	dir := setupRepo(t)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	got, err := g.RepoRoot(context.Background())
	require.NoError(t, err)
	// On macOS /tmp is a symlink to /private/tmp; resolve both for comparison.
	gotResolved, err := filepath.EvalSymlinks(got)
	require.NoError(t, err)
	dirResolved, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	assert.Equal(t, dirResolved, gotResolved)
}

func TestRealGit_CurrentBranch(t *testing.T) {
	g := newRealForTest(t)
	dir := setupRepo(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	got, err := g.CurrentBranch(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "main", got)
}

func TestRealGit_BranchExists(t *testing.T) {
	g := newRealForTest(t)
	dir := setupRepo(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	exists, err := g.BranchExists(context.Background(), "main")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = g.BranchExists(context.Background(), "does-not-exist")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRealGit_CreateBranch(t *testing.T) {
	g := newRealForTest(t)
	dir := setupRepo(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, g.CreateBranch(context.Background(), "feature/x", "main"))

	exists, err := g.BranchExists(context.Background(), "feature/x")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRealGit_CreateBranch_BadBase(t *testing.T) {
	g := newRealForTest(t)
	dir := setupRepo(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	err = g.CreateBranch(context.Background(), "feature/x", "no-such-base")
	assert.Error(t, err)
}

func TestRealGit_WorktreeAdd(t *testing.T) {
	g := newRealForTest(t)
	dir := setupRepo(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, g.CreateBranch(context.Background(), "feature/y", "main"))

	dest := filepath.Join(t.TempDir(), "wt-y")
	require.NoError(t, g.WorktreeAdd(context.Background(), dest, "feature/y"))
	_, err = os.Stat(dest)
	assert.NoError(t, err)
}

func TestRealGit_LsIgnored(t *testing.T) {
	g := newRealForTest(t)
	dir := setupRepo(t)

	// Add a .gitignore that ignores .env, then create one.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".env\n"), 0o644))
	mustRun(t, dir, "git", "add", ".gitignore")
	mustRun(t, dir, "git", "commit", "-q", "-m", "ignore env")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o644))

	got, err := g.LsIgnored(context.Background(), dir, nil)
	require.NoError(t, err)
	assert.Contains(t, got, ".env")
}

func TestRealGit_LsIgnored_Scoped(t *testing.T) {
	g := newRealForTest(t)
	dir := setupRepo(t)

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("secrets/\n"), 0o644))
	mustRun(t, dir, "git", "add", ".gitignore")
	mustRun(t, dir, "git", "commit", "-q", "-m", "ignore secrets")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "secrets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets", "key"), []byte("k"), 0o644))

	got, err := g.LsIgnored(context.Background(), dir, []string{"secrets"})
	require.NoError(t, err)
	require.NotEmpty(t, got)
	// The --directory flag collapses entire ignored directories to "secrets/".
	found := false
	for _, p := range got {
		if p == "secrets/" || p == "secrets/key" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 'secrets/' or 'secrets/key' in %v", got)
}

func TestRealGit_CheckRefFormat(t *testing.T) {
	g := newRealForTest(t)

	require.NoError(t, g.CheckRefFormat(context.Background(), "feature/ok"))
	assert.Error(t, g.CheckRefFormat(context.Background(), "bad..name"))
	assert.Error(t, g.CheckRefFormat(context.Background(), ""))
}

func TestRealGit_NewReturnsRealGit(t *testing.T) {
	g := New()
	rg, ok := g.(*realGit)
	require.True(t, ok)
	assert.Equal(t, "git", rg.exe)
}
