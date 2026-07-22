package crypto

import (
	"encoding/base64"
	"fmt"
	"os"
)

// LoadOrCreateKeyFile reads a base64-encoded AES-256 key from path. If the
// file does not exist, it generates a new key, writes it to path (mode
// 0600), and returns it. This exists purely for local development
// convenience (crypto.kms_provider=local) - it is never appropriate for
// production, where keys must come from a real KMS.
func LoadOrCreateKeyFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		key, decodeErr := base64.StdEncoding.DecodeString(string(data))
		if decodeErr != nil {
			return nil, fmt.Errorf("crypto: decode key file %s: %w", path, decodeErr)
		}
		if len(key) != KeySize {
			return nil, fmt.Errorf("crypto: key file %s: %w", path, ErrInvalidKeySize)
		}
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("crypto: read key file %s: %w", path, err)
	}

	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		return nil, fmt.Errorf("crypto: write key file %s: %w", path, err)
	}

	return key, nil
}
