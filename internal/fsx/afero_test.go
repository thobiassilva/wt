package fsx

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeAfero(t *testing.T, fs afero.Fs, path string, data []byte, mode os.FileMode) {
	t.Helper()
	require.NoError(t, afero.WriteFile(fs, path, data, mode))
}

func TestAferoFS_Stat(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()
	writeAfero(t, mem, "/a.txt", []byte("hello"), 0o644)

	fsys := NewAfero(mem)
	info, err := fsys.Stat("/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "a.txt", info.Name())
	assert.Equal(t, int64(5), info.Size())
}

func TestAferoFS_Stat_Missing(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()

	fsys := NewAfero(mem)
	_, err := fsys.Stat("/nope")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestAferoFS_MkdirAll(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()

	fsys := NewAfero(mem)
	require.NoError(t, fsys.MkdirAll("/a/b/c", 0o755))

	info, err := mem.Stat("/a/b/c")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestAferoFS_ReadFile(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()
	want := []byte("contents-123")
	writeAfero(t, mem, "/data.txt", want, 0o644)

	fsys := NewAfero(mem)
	got, err := fsys.ReadFile("/data.txt")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAferoFS_ReadFile_Missing(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()

	fsys := NewAfero(mem)
	_, err := fsys.ReadFile("/missing")
	require.Error(t, err)
}

func TestAferoFS_Exists(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()
	writeAfero(t, mem, "/present.txt", []byte("x"), 0o644)
	require.NoError(t, mem.MkdirAll("/some-dir", 0o755))

	fsys := NewAfero(mem)
	assert.True(t, fsys.Exists("/present.txt"))
	assert.True(t, fsys.Exists("/some-dir"))
	assert.False(t, fsys.Exists("/absent.txt"))
}

func TestAferoFS_CopyFile(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()
	want := []byte("payload\nwith newline\n")
	writeAfero(t, mem, "/src.txt", want, 0o644)

	fsys := NewAfero(mem)
	require.NoError(t, fsys.CopyFile("/src.txt", "/dst.txt"))

	got, err := afero.ReadFile(mem, "/dst.txt")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAferoFS_CopyFile_PreservesPermissions(t *testing.T) {
	t.Parallel()

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
			mem := afero.NewMemMapFs()
			writeAfero(t, mem, "/src", []byte("x"), tc.mode)

			fsys := NewAfero(mem)
			require.NoError(t, fsys.CopyFile("/src", "/dst"))

			info, err := mem.Stat("/dst")
			require.NoError(t, err)
			assert.Equal(t, tc.mode, info.Mode().Perm())
		})
	}
}

func TestAferoFS_CopyFile_Empty(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()
	writeAfero(t, mem, "/empty.txt", []byte{}, 0o644)

	fsys := NewAfero(mem)
	require.NoError(t, fsys.CopyFile("/empty.txt", "/empty-copy.txt"))

	got, err := afero.ReadFile(mem, "/empty-copy.txt")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestAferoFS_CopyFile_SrcMissing(t *testing.T) {
	t.Parallel()
	mem := afero.NewMemMapFs()

	fsys := NewAfero(mem)
	err := fsys.CopyFile("/nope", "/dst")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open source")
}

// TestAferoFS_Isolation verifies that two parallel tests using independent
// MemMapFs instances do not see each other's files.
func TestAferoFS_Isolation(t *testing.T) {
	t.Parallel()

	t.Run("instance-a", func(t *testing.T) {
		t.Parallel()
		mem := afero.NewMemMapFs()
		writeAfero(t, mem, "/only-in-a.txt", []byte("a"), 0o644)

		fsys := NewAfero(mem)
		assert.True(t, fsys.Exists("/only-in-a.txt"))
		assert.False(t, fsys.Exists("/only-in-b.txt"))
	})

	t.Run("instance-b", func(t *testing.T) {
		t.Parallel()
		mem := afero.NewMemMapFs()
		writeAfero(t, mem, "/only-in-b.txt", []byte("b"), 0o644)

		fsys := NewAfero(mem)
		assert.True(t, fsys.Exists("/only-in-b.txt"))
		assert.False(t, fsys.Exists("/only-in-a.txt"))
	})
}
