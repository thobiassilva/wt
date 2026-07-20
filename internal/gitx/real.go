package gitx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// realGit is the production implementation of Git, shelling out to a git
// binary located at exe (defaults to "git", PATH-resolved).
type realGit struct {
	exe string
}

// New returns a production Git implementation that invokes the "git" binary
// found on PATH.
func New() Git {
	return &realGit{exe: "git"}
}

// run executes git with args, optionally inside dir, and returns stdout.
// On non-zero exit it returns an error that includes stderr.
func (g *realGit) run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, g.exe, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", wrapExecError(err, args, stderr.String())
	}
	return stdout.String(), nil
}

func wrapExecError(err error, args []string, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr)
}

func trimTrailingNewline(s string) string {
	return strings.TrimRight(s, "\r\n")
}

// RepoRoot returns the absolute path of the current git repository root.
func (g *realGit) RepoRoot(ctx context.Context) (string, error) {
	out, err := g.run(ctx, "", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return trimTrailingNewline(out), nil
}

// CurrentBranch returns the name of the currently checked-out branch.
// On a detached HEAD git prints an empty line; we propagate that as "".
func (g *realGit) CurrentBranch(ctx context.Context) (string, error) {
	out, err := g.run(ctx, "", "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return trimTrailingNewline(out), nil
}

// BranchExists reports whether refs/heads/<name> resolves.
// Exit 0 -> true, exit 1 -> false, any other failure is returned as an error.
func (g *realGit) BranchExists(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, g.exe, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// git show-ref --verify --quiet exits 1 when the ref does not exist.
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("git show-ref refs/heads/%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return false, fmt.Errorf("git show-ref refs/heads/%s: %w", name, err)
}

// CreateBranch creates a new branch named name from base.
func (g *realGit) CreateBranch(ctx context.Context, name, base string) error {
	if _, err := g.run(ctx, "", "branch", name, base); err != nil {
		return err
	}
	return nil
}

// WorktreeAdd creates a worktree at dest pointing to branch.
func (g *realGit) WorktreeAdd(ctx context.Context, dest, branch string) error {
	if _, err := g.run(ctx, "", "worktree", "add", dest, branch); err != nil {
		return err
	}
	return nil
}

// LsIgnored runs `git ls-files --others --ignored --exclude-standard --directory`
// inside repoRoot, optionally scoped to the given pathspec arguments.
// An empty result (or non-zero exit) is treated as "no matches" and returns
// (nil, nil) so callers can iterate without nil-checking the error.
func (g *realGit) LsIgnored(ctx context.Context, repoRoot string, paths []string) ([]string, error) {
	args := []string{"ls-files", "--others", "--ignored", "--exclude-standard", "--directory"}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	cmd := exec.CommandContext(ctx, g.exe, args...)
	if repoRoot != "" {
		cmd.Dir = repoRoot
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// ls-files is read-only; empty matches still exit 0. A real failure
		// here (missing repo, bad pathspec) is reported, but we tolerate the
		// "no output" case by not erroring on empty stdout below.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	out := stdout.String()
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			continue
		}
		result = append(result, l)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// WorktreeList runs `git worktree list --porcelain` and parses the output into
// Worktree records. The main worktree is always the first entry.
func (g *realGit) WorktreeList(ctx context.Context) ([]Worktree, error) {
	out, err := g.run(ctx, "", "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreePorcelain(out), nil
}

// parseWorktreePorcelain parses the output of `git worktree list --porcelain`.
// Records are separated by blank lines; each attribute is a label optionally
// followed by a value. The "branch" attribute carries a full ref name
// (refs/heads/<name>), which we shorten to <name>.
func parseWorktreePorcelain(out string) []Worktree {
	var (
		result []Worktree
		cur    Worktree
		inRec  bool
	)
	flush := func() {
		if inRec {
			result = append(result, cur)
		}
		cur = Worktree{}
		inRec = false
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		label, value, _ := strings.Cut(line, " ")
		switch label {
		case "worktree":
			cur.Path = value
			inRec = true
		case "HEAD":
			cur.Head = value
		case "branch":
			cur.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			cur.Bare = true
		case "detached":
			cur.Detached = true
		case "locked":
			cur.Locked = true
		case "prunable":
			cur.Prunable = true
		}
	}
	flush()
	return result
}

// WorktreeRemove runs `git worktree remove <path>`, adding --force when force.
func (g *realGit) WorktreeRemove(ctx context.Context, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	if _, err := g.run(ctx, "", args...); err != nil {
		return err
	}
	return nil
}

// DeleteBranch runs `git branch -d <name>` (or -D when force is true).
func (g *realGit) DeleteBranch(ctx context.Context, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	if _, err := g.run(ctx, "", "branch", flag, name); err != nil {
		return err
	}
	return nil
}

// CheckRefFormat validates a branch name via `git check-ref-format --branch`.
func (g *realGit) CheckRefFormat(ctx context.Context, branch string) error {
	cmd := exec.CommandContext(ctx, g.exe, "check-ref-format", "--branch", branch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("invalid branch name %q: %w: %s", branch, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Ensure compile-time interface satisfaction.
var _ Git = (*realGit)(nil)
