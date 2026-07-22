// Package crypto provides AES-256-GCM encryption, both as simple
// single-key functions and as a versioned Keyring that supports key
// rotation.
//
// The Keyring is what makes rotation safe: every ciphertext it produces is
// tagged with the key version that encrypted it, so Decrypt can look up the
// matching key by version instead of assuming one global key. Data written
// under a retired key version keeps decrypting correctly after rotation.
//
// This package is deliberately unaware of persistence: callers own storing
// (ciphertext, key_version) pairs and handing the right version back in on
// read.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
)

// KeySize is the AES-256 key length in bytes.
const KeySize = 32

var (
	// ErrKeyNotFound is returned when Decrypt is asked for a key version the
	// Keyring does not hold.
	ErrKeyNotFound = errors.New("crypto: key version not found")

	// ErrCiphertextTooShort is returned when a ciphertext is smaller than
	// the GCM nonce, and therefore cannot be a value this package produced.
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")

	// ErrInvalidKeySize is returned when a key is not exactly KeySize bytes.
	ErrInvalidKeySize = errors.New("crypto: key must be 32 bytes for AES-256")
)

// GenerateKey returns a new cryptographically random 32-byte AES-256 key.
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("crypto: generate key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext with a single AES-256 key using GCM. The
// returned ciphertext is nonce || sealed data, so Decrypt only needs the
// key and the ciphertext to recover the plaintext. Intended for single-key
// use cases (e.g. sealing a data key under a master key in envelope
// encryption); see Keyring for versioned, rotation-safe encryption.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	return seal(key, plaintext)
}

// Decrypt decrypts a ciphertext produced by Encrypt using the same key.
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	return open(key, ciphertext)
}

// Encrypted is a ciphertext together with the key version that produced it.
type Encrypted struct {
	Ciphertext []byte
	KeyVersion int
}

// Keyring holds a set of versioned AES-256 keys. Encrypt always uses the
// current (highest) version; Decrypt looks up whichever version the
// ciphertext was tagged with, so data encrypted under a retired key version
// remains readable after rotation.
//
// Keyring is safe for concurrent use.
type Keyring struct {
	mu      sync.RWMutex
	keys    map[int][]byte
	current int
}

// NewKeyring builds a Keyring seeded with a single key at version 1.
func NewKeyring(initialKey []byte) (*Keyring, error) {
	if len(initialKey) != KeySize {
		return nil, ErrInvalidKeySize
	}
	return &Keyring{
		keys:    map[int][]byte{1: append([]byte(nil), initialKey...)},
		current: 1,
	}, nil
}

// CurrentVersion returns the key version used for new encryptions.
func (k *Keyring) CurrentVersion() int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.current
}

// RotateKey generates a fresh AES-256 key, registers it under the next key
// version, and makes it the current version for future Encrypt calls. Keys
// from prior versions are retained so existing ciphertexts keep decrypting.
// Returns the new version number.
func (k *Keyring) RotateKey() (int, error) {
	newKey, err := GenerateKey()
	if err != nil {
		return 0, err
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	next := k.current + 1
	k.keys[next] = newKey
	k.current = next
	return next, nil
}

// AddKeyVersion registers an explicit key under a specific version, without
// changing which version is current unless version is greater than the
// current version. Useful for seeding a Keyring from persisted key material
// (e.g. loaded from a secrets provider) or for tests.
func (k *Keyring) AddKeyVersion(version int, key []byte) error {
	if len(key) != KeySize {
		return ErrInvalidKeySize
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	k.keys[version] = append([]byte(nil), key...)
	if version > k.current {
		k.current = version
	}
	return nil
}

// Encrypt encrypts plaintext with the current key version using AES-256-GCM.
func (k *Keyring) Encrypt(plaintext []byte) (Encrypted, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	key, ok := k.keys[k.current]
	if !ok {
		return Encrypted{}, fmt.Errorf("crypto: %w: version %d", ErrKeyNotFound, k.current)
	}

	ciphertext, err := seal(key, plaintext)
	if err != nil {
		return Encrypted{}, err
	}

	return Encrypted{Ciphertext: ciphertext, KeyVersion: k.current}, nil
}

// Decrypt decrypts a ciphertext that was encrypted under the given key
// version. Returns ErrKeyNotFound if that version is not present in the
// Keyring.
func (k *Keyring) Decrypt(ciphertext []byte, keyVersion int) ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	key, ok := k.keys[keyVersion]
	if !ok {
		return nil, fmt.Errorf("crypto: %w: version %d", ErrKeyNotFound, keyVersion)
	}
	return open(key, ciphertext)
}

// seal encrypts plaintext with AES-256-GCM under key, prefixing the result
// with a randomly generated nonce.
func seal(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// open reverses seal: it splits the nonce off the front of ciphertext and
// decrypts+authenticates the remainder under key.
func open(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, sealed := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}

	return plaintext, nil
}
