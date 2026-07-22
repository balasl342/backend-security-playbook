package crypto

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey_ReturnsCorrectSizeAndIsRandom(t *testing.T) {
	k1, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, k1, KeySize)

	k2, err := GenerateKey()
	require.NoError(t, err)
	assert.NotEqual(t, k1, k2, "two generated keys should not collide")
}

func TestEncrypt_Decrypt_RoundTrip(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	plaintext := []byte("123-45-6789")

	ciphertext, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := Decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_ProducesDifferentCiphertextEachTime(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	plaintext := []byte("same input")

	c1, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	c2, err := Encrypt(key, plaintext)
	require.NoError(t, err)

	assert.False(t, bytes.Equal(c1, c2), "nonce reuse would be a critical GCM vulnerability")
}

func TestEncrypt_RejectsWrongKeySize(t *testing.T) {
	_, err := Encrypt([]byte("too-short"), []byte("data"))
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestDecrypt_RejectsWrongKeySize(t *testing.T) {
	_, err := Decrypt([]byte("too-short"), []byte("data"))
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestDecrypt_RejectsTamperedCiphertext(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	ciphertext, err := Encrypt(key, []byte("secret"))
	require.NoError(t, err)

	tampered := append([]byte(nil), ciphertext...)
	tampered[len(tampered)-1] ^= 0xFF

	_, err = Decrypt(key, tampered)
	assert.Error(t, err)
}

func TestDecrypt_RejectsWrongKey(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	ciphertext, err := Encrypt(key1, []byte("secret"))
	require.NoError(t, err)

	_, err = Decrypt(key2, ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_RejectsTooShortCiphertext(t *testing.T) {
	key, _ := GenerateKey()
	_, err := Decrypt(key, []byte("short"))
	assert.ErrorIs(t, err, ErrCiphertextTooShort)
}

func TestNewKeyring_RejectsWrongKeySize(t *testing.T) {
	_, err := NewKeyring([]byte("too-short"))
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestKeyring_EncryptUsesCurrentVersion(t *testing.T) {
	key, _ := GenerateKey()
	kr, err := NewKeyring(key)
	require.NoError(t, err)
	assert.Equal(t, 1, kr.CurrentVersion())

	enc, err := kr.Encrypt([]byte("payload"))
	require.NoError(t, err)
	assert.Equal(t, 1, enc.KeyVersion)
}

func TestKeyring_Decrypt_RoundTrip(t *testing.T) {
	key, _ := GenerateKey()
	kr, err := NewKeyring(key)
	require.NoError(t, err)

	plaintext := []byte("4111111111111111")
	enc, err := kr.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := kr.Decrypt(enc.Ciphertext, enc.KeyVersion)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestKeyring_RotateKey_OldCiphertextsStillDecrypt(t *testing.T) {
	key, _ := GenerateKey()
	kr, err := NewKeyring(key)
	require.NoError(t, err)

	oldPlaintext := []byte("encrypted under v1")
	oldEnc, err := kr.Encrypt(oldPlaintext)
	require.NoError(t, err)
	require.Equal(t, 1, oldEnc.KeyVersion)

	newVersion, err := kr.RotateKey()
	require.NoError(t, err)
	assert.Equal(t, 2, newVersion)
	assert.Equal(t, 2, kr.CurrentVersion())

	newPlaintext := []byte("encrypted under v2")
	newEnc, err := kr.Encrypt(newPlaintext)
	require.NoError(t, err)
	assert.Equal(t, 2, newEnc.KeyVersion)

	// Old ciphertext, encrypted before rotation, must still decrypt using
	// its original key version.
	decryptedOld, err := kr.Decrypt(oldEnc.Ciphertext, oldEnc.KeyVersion)
	require.NoError(t, err)
	assert.Equal(t, oldPlaintext, decryptedOld)

	decryptedNew, err := kr.Decrypt(newEnc.Ciphertext, newEnc.KeyVersion)
	require.NoError(t, err)
	assert.Equal(t, newPlaintext, decryptedNew)
}

func TestKeyring_Decrypt_UnknownVersionFails(t *testing.T) {
	key, _ := GenerateKey()
	kr, err := NewKeyring(key)
	require.NoError(t, err)

	_, err = kr.Decrypt([]byte("whatever"), 99)
	assert.True(t, errors.Is(err, ErrKeyNotFound))
}

func TestKeyring_AddKeyVersion_RejectsWrongKeySize(t *testing.T) {
	key, _ := GenerateKey()
	kr, err := NewKeyring(key)
	require.NoError(t, err)

	err = kr.AddKeyVersion(5, []byte("too-short"))
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestKeyring_AddKeyVersion_AdvancesCurrentOnlyIfHigher(t *testing.T) {
	key, _ := GenerateKey()
	kr, err := NewKeyring(key)
	require.NoError(t, err)

	lowerKey, _ := GenerateKey()
	require.NoError(t, kr.AddKeyVersion(0, lowerKey))
	assert.Equal(t, 1, kr.CurrentVersion(), "adding a lower version must not change current")

	higherKey, _ := GenerateKey()
	require.NoError(t, kr.AddKeyVersion(5, higherKey))
	assert.Equal(t, 5, kr.CurrentVersion())
}

func TestKeyring_ConcurrentEncryptAndRotate(t *testing.T) {
	key, _ := GenerateKey()
	kr, err := NewKeyring(key)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			_, _ = kr.Encrypt([]byte("concurrent"))
		}
	}()

	for i := 0; i < 3; i++ {
		_, err := kr.RotateKey()
		require.NoError(t, err)
	}

	<-done
}
