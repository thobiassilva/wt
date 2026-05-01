package fsx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealFS_Stat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	require.NoError(t, os.WriteFile(f, []byte("hello"), 0o644))

	fsys := New()
	info, err := fsys.Stat(f)
	require.NoError(t, err)
	assert.Equal(t, "a.txt", info.Name())
	assert.Equal(t, int64(5), info.Size())
}

func TestRealFS_Stat_Missing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	fsys := New()
	_, err := fsys.Stat(filepath.Join(dir, "nope.txt"))
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestRealFS_MkdirAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")

	fsys := New()
	require.NoError(t, fsys.MkdirAll(nested, 0o755))

	info, err := os.Stat(nested)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestRealFS_ReadFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f := filepath.Join(dir, "data.txt")
	want := []byte("contents-123")
	require.NoError(t, os.WriteFile(f, want, 0o644))

	fsys := New()
	got, err := fsys.ReadFile(f)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestRealFS_ReadFile_Missing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	fsys := New()
	_, err := fsys.ReadFile(filepath.Join(dir, "missing"))
	require.Error(t, err)
}

func TestRealFS_Exists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f := filepath.Join(dir, "present.txt")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o644))

	fsys := New()
	assert.True(t, fsys.Exists(f))
	assert.True(t, fsys.Exists(dir))
	assert.False(t, fsys.Exists(filepath.Join(dir, "absent.txt")))
}

func TestRealFS_CopyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	want := []byte("payload\nwith newline\n")
	require.NoError(t, os.WriteFile(src, want, 0o644))

	fsys := New()
	require.NoError(t, fsys.CopyFile(src, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestRealFS_CopyFile_PreservesPermissions(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix file permissions")
	}
	dir := t.TempDir()

	cases := []struct {
		name string
		mode os.FileMode
	}{
		{"0644", 0o644},
		{"0600", 0o600},
		{"0755", 0o755},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := filepath.Join(dir, "src-"+tc.name)
			dst := filepath.Join(dir, "dst-"+tc.name)
			require.NoError(t, os.WriteFile(src, []byte("x"), tc.mode))

			fsys := New()
			require.NoError(t, fsys.CopyFile(src, dst))

			info, err := os.Stat(dst)
			require.NoError(t, err)
			assert.Equal(t, tc.mode, info.Mode().Perm())
		})
	}
}

func TestRealFS_CopyFile_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "empty.txt")
	dst := filepath.Join(dir, "empty-copy.txt")
	require.NoError(t, os.WriteFile(src, []byte{}, 0o644))

	fsys := New()
	require.NoError(t, fsys.CopyFile(src, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRealFS_CopyFile_SrcMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	fsys := New()
	err := fsys.CopyFile(filepath.Join(dir, "nope"), filepath.Join(dir, "dst"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open source")
}

func TestRealFS_CopyFile_DstDirMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o644))

	fsys := New()
	// Destination's parent directory doesn't exist.
	err := fsys.CopyFile(src, filepath.Join(dir, "missing-dir", "dst.txt"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create destination")
}
