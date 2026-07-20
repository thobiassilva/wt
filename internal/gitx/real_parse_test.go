package gitx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorktreePorcelain_Basic(t *testing.T) {
	out := "worktree /path/to/main\n" +
		"HEAD abcd1234\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /path/to/wt-feature\n" +
		"HEAD ef567890\n" +
		"branch refs/heads/feature/x\n" +
		"\n"

	got := parseWorktreePorcelain(out)
	require.Len(t, got, 2)

	assert.Equal(t, "/path/to/main", got[0].Path)
	assert.Equal(t, "abcd1234", got[0].Head)
	assert.Equal(t, "main", got[0].Branch)
	assert.False(t, got[0].Detached)

	assert.Equal(t, "/path/to/wt-feature", got[1].Path)
	assert.Equal(t, "feature/x", got[1].Branch)
}

func TestParseWorktreePorcelain_Detached(t *testing.T) {
	out := "worktree /path/to/detached\n" +
		"HEAD 11223344\n" +
		"detached\n" +
		"\n"

	got := parseWorktreePorcelain(out)
	require.Len(t, got, 1)
	assert.True(t, got[0].Detached)
	assert.Empty(t, got[0].Branch)
}

func TestParseWorktreePorcelain_BareLockedPrunable(t *testing.T) {
	out := "worktree /path/to/bare\n" +
		"bare\n" +
		"\n" +
		"worktree /path/to/broken\n" +
		"HEAD 55667788\n" +
		"branch refs/heads/old\n" +
		"locked reason text\n" +
		"prunable gitdir file points to non-existent location\n" +
		"\n"

	got := parseWorktreePorcelain(out)
	require.Len(t, got, 2)

	assert.True(t, got[0].Bare)
	assert.Empty(t, got[0].Head)
	assert.Empty(t, got[0].Branch)

	assert.Equal(t, "old", got[1].Branch)
	assert.True(t, got[1].Locked)
	assert.True(t, got[1].Prunable)
}

func TestParseWorktreePorcelain_NoTrailingBlankLine(t *testing.T) {
	out := "worktree /path/to/main\n" +
		"HEAD abcd1234\n" +
		"branch refs/heads/main"

	got := parseWorktreePorcelain(out)
	require.Len(t, got, 1)
	assert.Equal(t, "main", got[0].Branch)
}

func TestParseWorktreePorcelain_Empty(t *testing.T) {
	assert.Empty(t, parseWorktreePorcelain(""))
}

func TestParseWorktreePorcelain_CRLF(t *testing.T) {
	out := "worktree /path/to/main\r\nHEAD abcd1234\r\nbranch refs/heads/main\r\n\r\n"
	got := parseWorktreePorcelain(out)
	require.Len(t, got, 1)
	assert.Equal(t, "/path/to/main", got[0].Path)
	assert.Equal(t, "main", got[0].Branch)
}
