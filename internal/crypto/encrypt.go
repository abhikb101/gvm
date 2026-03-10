package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Encrypt encrypts a token using AES-256-GCM with a machine-derived key.
func Encrypt(plaintext string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", fmt.Errorf("deriving encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a token that was encrypted with Encrypt.
func Decrypt(encoded string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", fmt.Errorf("deriving encryption key: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decoding token: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting token: %w", err)
	}

	return string(plaintext), nil
}

// deriveKey creates a deterministic 256-bit key from machine-specific data.
func deriveKey() ([]byte, error) {
	machineID, err := getMachineID()
	if err != nil {
		return nil, err
	}

	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME") // Windows fallback
	}

	// SHA-256 produces exactly 32 bytes = AES-256 key
	hash := sha256.Sum256([]byte("gvm:" + machineID + ":" + user))
	return hash[:], nil
}

func getMachineID() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
		if err != nil {
			return fallbackMachineID()
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "IOPlatformUUID") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					return strings.Trim(strings.TrimSpace(parts[1]), "\""), nil
				}
			}
		}
		return fallbackMachineID()

	case "linux":
		// Try /etc/machine-id first (systemd), then /var/lib/dbus/machine-id
		for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
			data, err := os.ReadFile(path)
			if err == nil {
				id := strings.TrimSpace(string(data))
				if id != "" {
					return id, nil
				}
			}
		}
		return fallbackMachineID()

	default:
		return fallbackMachineID()
	}
}

func fallbackMachineID() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("cannot determine machine identity: %w", err)
	}
	home, _ := os.UserHomeDir()
	return hostname + ":" + home, nil
}
