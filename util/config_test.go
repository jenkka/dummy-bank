package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	content := []byte("DB_DRIVER=postgres\nDB_SOURCE=postgres://test\nSERVER_ADDRESS=0.0.0.0:8080\n")
	err := os.WriteFile(tmpDir+"/app.env", content, 0644)
	require.NoError(t, err)

	config, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	require.Equal(t, "postgres", config.DbDriver)
	require.Equal(t, "postgres://test", config.DbSource)
	require.Equal(t, "0.0.0.0:8080", config.ServerAddress)
}

func TestLoadConfigInvalidPath(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path")
	require.Error(t, err)
}
