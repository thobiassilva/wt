//go:build integration

package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wtBinary returns the path to the compiled wt binary. It compiles it once
// into t.TempDir() and caches the path across the test binary run.
var wtBin string

func TestMain(m *testing.M) {
	bin, err := buildBinary()
	if err != nil {
		panic("failed to build wt binary: " + err.Error())
	}
	wtBin = bin
	os.Exit(m.Run())
}

func buildBinary() (string, error) {
	dir, err := os.MkdirTemp("", "wt-integration-*")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, "wt")
	cmd := exec.Command("go", "build", "-o", bin, "../cmd/wt")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return bin, cmd.Run()
}

// setupRepo initializes a fresh git repo with one commit in a temp dir.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main")
	mustGit(t, dir, "config", "user.email", "test@example.com")
	mustGit(t, dir, "config", "user.name", "Test")
	mustGit(t, dir, "config", "commit.gpgsign", "false")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644))
	mustGit(t, dir, "add", "README.md")
	mustGit(t, dir, "commit", "-q", "-m", "init")
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, out)
}

func runWt(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(wtBin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		}
	}
	return string(out), code
}

func TestIntegration_DryRun_NewBranch(t *testing.T) {
	repo := setupRepo(t)
	out, code := runWt(t, repo, "feature/testBranch", "--dry-run")
	assert.Equal(t, 0, code, "output: %s", out)
	assert.Contains(t, out, "feature/testBranch")
	assert.Contains(t, out, "feature-test-branch")
	assert.Contains(t, out, "dry-run")

	// No worktree should have been created.
	_, err := os.Stat(filepath.Join(filepath.Dir(repo), "feature-test-branch"))
	assert.True(t, os.IsNotExist(err), "worktree directory must not be created in dry-run")
}

func TestIntegration_CreateWorktree(t *testing.T) {
	repo := setupRepo(t)
	dest := filepath.Join(t.TempDir(), "feature-my-work")

	out, code := runWt(t, repo, "feature/myWork", "--path", filepath.Dir(dest))
	assert.Equal(t, 0, code, "output: %s", out)

	// Worktree directory must exist.
	info, err := os.Stat(dest)
	require.NoError(t, err, "worktree directory must be created")
	assert.True(t, info.IsDir())

	// Branch must exist in the original repo.
	cmd := exec.Command("git", "branch", "--list", "feature/myWork")
	cmd.Dir = repo
	branchOut, _ := cmd.Output()
	assert.Contains(t, string(branchOut), "feature/myWork")
}

func TestIntegration_WorktreeInclude_WithNegation(t *testing.T) {
	repo := setupRepo(t)

	// Create .gitignore that ignores *.env files.
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("*.env\n"), 0o644))
	mustGit(t, repo, "add", ".gitignore")
	mustGit(t, repo, "commit", "-q", "-m", "add gitignore")

	// Create .worktreeinclude with negation: copy *.env but NOT prod.env.
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".worktreeinclude"), []byte("*.env\n!prod.env\n"), 0o644))

	// Create ignored files.
	require.NoError(t, os.WriteFile(filepath.Join(repo, "dev.env"), []byte("DEV=1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "prod.env"), []byte("PROD=1\n"), 0o644))

	dest := filepath.Join(t.TempDir(), "feature-env")
	out, code := runWt(t, repo, "feature/env", "--path", filepath.Dir(dest))
	assert.Equal(t, 0, code, "output: %s", out)

	// dev.env must be copied.
	devEnv, err := os.ReadFile(filepath.Join(dest, "dev.env"))
	require.NoError(t, err, "dev.env must be copied")
	assert.Equal(t, "DEV=1\n", string(devEnv))

	// prod.env must NOT be copied (negation rule).
	_, err = os.Stat(filepath.Join(dest, "prod.env"))
	assert.True(t, os.IsNotExist(err), "prod.env must be excluded by negation")
}

func TestIntegration_DestAlreadyExists_Error(t *testing.T) {
	repo := setupRepo(t)
	destParent := t.TempDir()
	dest := filepath.Join(destParent, "feature-x")
	require.NoError(t, os.MkdirAll(dest, 0o755))

	_, code := runWt(t, repo, "feature/x", "--path", destParent)
	assert.NotEqual(t, 0, code, "should fail when dest already exists")
}

func TestIntegration_InvalidBranchName_Error(t *testing.T) {
	repo := setupRepo(t)
	_, code := runWt(t, repo, "bad..name", "--path", t.TempDir())
	assert.NotEqual(t, 0, code)
}

func TestIntegration_NoInclude(t *testing.T) {
	repo := setupRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".env\n"), 0o644))
	mustGit(t, repo, "add", ".gitignore")
	mustGit(t, repo, "commit", "-q", "-m", "ignore")
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".worktreeinclude"), []byte(".env\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("SECRET=1\n"), 0o644))

	dest := filepath.Join(t.TempDir(), "feature-no-inc")
	out, code := runWt(t, repo, "feature/noInc", "--path", filepath.Dir(dest), "--no-include")
	assert.Equal(t, 0, code, "output: %s", out)

	_, err := os.Stat(filepath.Join(dest, ".env"))
	assert.True(t, os.IsNotExist(err), ".env must not be copied with --no-include")
}
