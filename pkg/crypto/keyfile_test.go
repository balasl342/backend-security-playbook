package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOrCreateKeyFile_CreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "master.key")

	key, err := LoadOrCreateKeyFile(path)
	require.NoError(t, err)
	assert.Len(t, key, KeySize)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}

func TestLoadOrCreateKeyFile_ReloadsSameKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "master.key")

	first, err := LoadOrCreateKeyFile(path)
	require.NoError(t, err)

	second, err := LoadOrCreateKeyFile(path)
	require.NoError(t, err)

	assert.Equal(t, first, second)
}

func TestLoadOrCreateKeyFile_RejectsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "master.key")
	require.NoError(t, os.WriteFile(path, []byte("not-valid-base64!!!"), 0o600))

	_, err := LoadOrCreateKeyFile(path)
	assert.Error(t, err)
}

func TestLoadOrCreateKeyFile_RejectsWrongLengthKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "master.key")
	require.NoError(t, os.WriteFile(path, []byte("c2hvcnQ="), 0o600)) // base64("short")

	_, err := LoadOrCreateKeyFile(path)
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestLoadOrCreateKeyFile_RoundTripsThroughEncrypt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "master.key")

	key, err := LoadOrCreateKeyFile(path)
	require.NoError(t, err)

	ciphertext, err := Encrypt(key, []byte("hello"))
	require.NoError(t, err)

	reloaded, err := LoadOrCreateKeyFile(path)
	require.NoError(t, err)

	plaintext, err := Decrypt(reloaded, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(plaintext))
}
