package worktree

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thobiassilva/wt/internal/fsx"
	"github.com/thobiassilva/wt/internal/gitx"
)

// makeFS returns an fsx.FS backed by an in-memory afero filesystem with
// repoRoot pre-created as a directory. Caller can then write files into it.
func makeFS(t *testing.T, repoRoot string) (fsx.FS, afero.Fs) {
	t.Helper()
	mem := afero.NewMemMapFs()
	require.NoError(t, mem.MkdirAll(repoRoot, 0o755))
	return fsx.NewAfero(mem), mem
}

func writeFile(t *testing.T, mem afero.Fs, path, content string) {
	t.Helper()
	require.NoError(t, afero.WriteFile(mem, path, []byte(content), 0o644))
}

func TestResolveIncludeFiles_NoFile(t *testing.T) {
	fs, _ := makeFS(t, "/repo")
	git := gitx.NewFake()

	files, err := resolveIncludeFiles(context.Background(), "/repo", git, fs)
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestResolveIncludeFiles_EmptyFile(t *testing.T) {
	fs, mem := makeFS(t, "/repo")
	writeFile(t, mem, "/repo/.worktreeinclude", "\n# just a comment\n\n")
	git := gitx.NewFake()

	files, err := resolveIncludeFiles(context.Background(), "/repo", git, fs)
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestResolveIncludeFiles_SimplePattern(t *testing.T) {
	fs, mem := makeFS(t, "/repo")
	writeFile(t, mem, "/repo/.worktreeinclude", ".env\n")

	git := gitx.NewFake()
	git.LsIgnoredOutput[".env"] = []string{".env"}

	files, err := resolveIncludeFiles(context.Background(), "/repo", git, fs)
	require.NoError(t, err)
	assert.Equal(t, []string{".env"}, files)
}

func TestResolveIncludeFiles_NegationExcludesFile(t *testing.T) {
	// Patterns: "*.env" matches everything ending in .env.
	// "!.env.example" then un-matches .env.example.
	// git returns both; the matcher should only include .env.
	fs, mem := makeFS(t, "/repo")
	writeFile(t, mem, "/repo/.worktreeinclude", "*.env\n!.env.example\n")

	git := gitx.NewFake()
	// Positive pattern is "*.env"; git returns both files.
	git.LsIgnoredOutput["*.env"] = []string{".env", ".env.example"}

	files, err := resolveIncludeFiles(context.Background(), "/repo", git, fs)
	require.NoError(t, err)
	assert.Equal(t, []string{".env"}, files)
}

func TestResolveIncludeFiles_NegationKeepsBothWhenNoMatch(t *testing.T) {
	// "*.env" + "!prod.env" — only prod.env is negated.
	fs, mem := makeFS(t, "/repo")
	writeFile(t, mem, "/repo/.worktreeinclude", "*.env\n!prod.env\n")

	git := gitx.NewFake()
	git.LsIgnoredOutput["*.env"] = []string{".env", "staging.env", "prod.env"}

	files, err := resolveIncludeFiles(context.Background(), "/repo", git, fs)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{".env", "staging.env"}, files)
}

func TestResolveIncludeFiles_CommentsAndBlanks(t *testing.T) {
	fs, mem := makeFS(t, "/repo")
	writeFile(t, mem, "/repo/.worktreeinclude", `
# copy secrets
.env

# nested
secrets/key
`)

	git := gitx.NewFake()
	// Both patterns are sent in one LsIgnored call; key is NUL-joined.
	git.LsIgnoredOutput[".env\x00secrets/key"] = []string{".env", "secrets/key"}

	files, err := resolveIncludeFiles(context.Background(), "/repo", git, fs)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{".env", "secrets/key"}, files)
}

func TestResolveIncludeFiles_MultiplePositivePatterns(t *testing.T) {
	fs, mem := makeFS(t, "/repo")
	writeFile(t, mem, "/repo/.worktreeinclude", ".env\nsecrets/\n")

	git := gitx.NewFake()
	// When called with multiple positive patterns they are joined in the key.
	git.LsIgnoredOutput[".env\x00secrets/"] = []string{".env", "secrets/key"}

	files, err := resolveIncludeFiles(context.Background(), "/repo", git, fs)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{".env", "secrets/key"}, files)
}

func TestParsePatterns(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"blank lines", "\n\n\n", nil},
		{"comments only", "# one\n# two\n", nil},
		{"simple", ".env\n", []string{".env"}},
		{"mixed", "# comment\n.env\n\n!.env.example\n", []string{".env", "!.env.example"}},
		{"trim whitespace", "  .env  \n", []string{".env"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := parsePatterns([]byte(tc.input))
			assert.Equal(t, tc.want, got)
		})
	}
}
