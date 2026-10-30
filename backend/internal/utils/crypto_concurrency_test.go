package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptionKeyProcessHelper(t *testing.T) {
	path := os.Getenv("CLAIMCTL_KEY_TEST_PATH")
	if path == "" {
		t.Skip("subprocess helper")
	}
	key, err := LoadOrGenerateKey("CLAIMCTL_SYNTHETIC_KEY", path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(os.Getenv("CLAIMCTL_KEY_TEST_RESULT"), []byte(key), 0600))
}

func TestEncryptionKeyConcurrentProcesses(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "keys", "app.key")
	t.Setenv("CLAIMCTL_SYNTHETIC_KEY", "")
	var wg sync.WaitGroup
	errors := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			command := exec.Command(os.Args[0], "-test.run=^TestEncryptionKeyProcessHelper$")
			command.Env = append(os.Environ(), "CLAIMCTL_KEY_TEST_PATH="+keyPath, "CLAIMCTL_KEY_TEST_RESULT="+filepath.Join(dir, string(rune('a'+i))))
			errors <- command.Run()
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	stable, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	for i := 0; i < 8; i++ {
		key, err := os.ReadFile(filepath.Join(dir, string(rune('a'+i))))
		require.NoError(t, err)
		require.Equal(t, stable, key)
	}
	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	require.NoError(t, os.WriteFile(keyPath, []byte("corrupt"), 0600))
	_, err = LoadOrGenerateKey("CLAIMCTL_SYNTHETIC_KEY", keyPath)
	require.Error(t, err)
	unchanged, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	require.Equal(t, "corrupt", string(unchanged))
	require.NoError(t, os.Remove(keyPath))
	missing := filepath.Join(dir, "missing-key")
	require.NoError(t, os.Symlink(missing, keyPath))
	_, err = LoadOrGenerateKey("CLAIMCTL_SYNTHETIC_KEY", keyPath)
	require.Error(t, err)
	target, err := os.Readlink(keyPath)
	require.NoError(t, err)
	require.Equal(t, missing, target)
}
