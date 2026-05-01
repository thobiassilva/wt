package worktree

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/thobiassilva/wt/internal/fsx"
	"github.com/thobiassilva/wt/internal/gitx"
)

// dirEntry is an alias for fs.DirEntry used in walkDir to keep the signature
// compatible with filepath.WalkDir.
type dirEntry = fs.DirEntry

// resolveIncludeFiles reads .worktreeinclude from repoRoot and returns the
// list of repo-relative file paths that should be copied. It supports the full
// gitignore spec (including negation via go-gitignore), matching only files
// that are both gitignored and untracked in the working tree.
//
// The function calls git.LsIgnored for each pattern to get the candidate set,
// then applies the full include pattern list as a second-pass filter so that
// negation patterns (e.g. "!.env.example") remove previously matched entries.
func resolveIncludeFiles(ctx context.Context, repoRoot string, git gitx.Git, fs fsx.FS) ([]string, error) {
	includeFile := filepath.Join(repoRoot, ".worktreeinclude")
	if !fs.Exists(includeFile) {
		return nil, nil
	}

	data, err := fs.ReadFile(includeFile)
	if err != nil {
		return nil, fmt.Errorf("read .worktreeinclude: %w", err)
	}

	patterns := parsePatterns(data)
	if len(patterns) == 0 {
		return nil, nil
	}

	// Build a combined gitignore matcher from all patterns. go-gitignore
	// processes rules in order and supports negation natively.
	matcher := ignore.CompileIgnoreLines(patterns...)

	// Gather positive (non-negation) patterns to scope the git ls-files call.
	// Passing only positive patterns avoids asking git about files we'll
	// exclude anyway, without breaking the negation logic (which is handled
	// by the matcher above, not by git).
	positivePatterns := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if !strings.HasPrefix(p, "!") {
			positivePatterns = append(positivePatterns, p)
		}
	}

	// Ask git for all gitignored+untracked files that could match our patterns.
	// We use the positive patterns as scope hints; git may return more than we
	// want, which is fine — the matcher below filters the final set.
	candidates, err := git.LsIgnored(ctx, repoRoot, positivePatterns)
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	// Expand collapsed directory entries (git --directory flag returns "dir/"
	// when the whole directory is ignored). We need individual file paths.
	expanded, err := expandDirectories(candidates, repoRoot, fs)
	if err != nil {
		return nil, err
	}

	// Apply the pattern matcher to the expanded set. This is where negation
	// takes effect: a file matched by "*.env" will be excluded if it also
	// matches a subsequent "!.env.example" rule.
	var result []string
	for _, rel := range expanded {
		// Normalize: strip trailing slash from directory entries that slipped through.
		rel = strings.TrimSuffix(rel, "/")
		if rel == "" {
			continue
		}
		if matcher.MatchesPath(rel) {
			result = append(result, rel)
		}
	}

	return result, nil
}

// parsePatterns scans raw .worktreeinclude bytes line by line, trimming
// whitespace and discarding blank lines and comments.
func parsePatterns(data []byte) []string {
	var out []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// expandDirectories resolves paths that end with "/" (collapsed directory
// entries produced by git ls-files --directory) into individual file paths
// by walking the real or in-memory filesystem.
func expandDirectories(paths []string, repoRoot string, fs fsx.FS) ([]string, error) {
	var result []string
	for _, p := range paths {
		if !strings.HasSuffix(p, "/") {
			result = append(result, p)
			continue
		}
		// p is a directory: walk it and collect all files.
		dir := filepath.Join(repoRoot, p)
		files, err := walkDir(dir, repoRoot, fs)
		if err != nil {
			// Directory may have been cleaned up; skip gracefully.
			continue
		}
		result = append(result, files...)
	}
	return result, nil
}

// walkDir returns repo-relative paths for all files found under dir.
func walkDir(dir, repoRoot string, fs fsx.FS) ([]string, error) {
	// We use the OS via fs.Stat to check existence. For full recursive walk we
	// fall back to filepath.WalkDir (real FS) or a limited afero walk.
	// Since fsx.FS does not expose a Walk method, we rely on filepath.WalkDir
	// for production paths. In tests, collapsed directories are not returned by
	// FakeGit (it returns individual file paths), so this code path is
	// effectively only exercised in integration tests against a real git repo.
	var out []string

	// Probe the path to see if it exists at all before walking.
	if !fs.Exists(dir) {
		return nil, nil
	}

	err := filepath.WalkDir(dir, func(path string, d dirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if !d.IsDir() {
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr == nil {
				out = append(out, rel)
			}
		}
		return nil
	})
	return out, err
}
