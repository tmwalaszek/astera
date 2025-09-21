package sqlite3

import (
	"astera"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSqlite3(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "methox-")
	require.NoError(t, err)

	db, err := NewDB(path.Join(tempDir, "test.db"))
	require.NoError(t, err)

	defer os.RemoveAll(tempDir)

	modules := []*astera.Module{
		{
			Name:    "github.com/tmwalaszek/module1",
			Version: "v1.0.0",
			Info:    []byte("info"),
			Mod:     []byte("mod"),
			Zip:     []byte("zip"),
			ZipHash: "hash",
		},
		{
			Name:    "github.com/tmwalaszek/module2",
			Version: "v2.0.0",
			Info:    []byte("info"),
			Mod:     []byte("mod"),
			Zip:     []byte("zip"),
			ZipHash: "hash",
		},
		{
			Name:    "github.com/tmwalaszek/module1",
			Version: "v1.0.0",
			Info:    []byte("info"),
			Mod:     []byte("mod"),
			Zip:     []byte("zip"),
			ZipHash: "hash",
		},
	}

	for _, m := range modules {
		err = db.InsertModule(m)
		require.NoError(t, err)
	}

	versions, err := db.GetVersionList("github.com/tmwalaszek/module1")
	require.NoError(t, err)

	require.Equal(t, []string{"v1.0.0"}, versions)

	infoFile, err := db.GetVersionInfo("github.com/tmwalaszek/module1", "v1.0.0")
	require.NoError(t, err)
	require.Equal(t, []byte("info"), infoFile)

	_, err = db.GetVersionInfo("github.com/tmwalaszek/module1", "v2.0.0")
	require.Error(t, astera.ErrModuleNotFound)

	modFile, err := db.GetModFile("github.com/tmwalaszek/module1", "v1.0.0")
	require.NoError(t, err)
	require.Equal(t, []byte("mod"), modFile)

	_, err = db.GetModFile("github.com/tmwalaszek/module1", "v2.0.0")
	require.Error(t, astera.ErrModuleNotFound)

	zipFile, err := db.GetModuleZip("github.com/tmwalaszek/module1", "v1.0.0")
	require.NoError(t, err)
	require.Equal(t, []byte("zip"), zipFile)

	_, err = db.GetModuleZip("github.com/tmwalaszek/module1", "v2.0.0")
	require.Error(t, astera.ErrModuleNotFound)

	exists, err := db.ModuleExists("github.com/tmwalaszek/module1", "v1.0.0")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = db.ModuleExists("github.com/tmwalaszek/module1", "v2.0.0")
	require.NoError(t, err)
	require.False(t, exists)
}
