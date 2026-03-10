package crypto

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	testCases := []string{
		"ghp_abc123def456",
		"a-very-long-token-with-special-chars!@#$%^&*()",
		"short",
		"",
	}

	for _, token := range testCases {
		t.Run(token, func(t *testing.T) {
			encrypted, err := Encrypt(token)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if token != "" && encrypted == token {
				t.Error("Encrypt() returned plaintext")
			}

			decrypted, err := Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != token {
				t.Errorf("Decrypt() = %q, want %q", decrypted, token)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	token := "ghp_test_token_123"

	enc1, err := Encrypt(token)
	if err != nil {
		t.Fatalf("first Encrypt() error = %v", err)
	}

	enc2, err := Encrypt(token)
	if err != nil {
		t.Fatalf("second Encrypt() error = %v", err)
	}

	// Due to random nonce, same plaintext should produce different ciphertexts
	if enc1 == enc2 {
		t.Error("Encrypt() produced identical ciphertexts for same input (nonce not random?)")
	}

	// Both should decrypt correctly
	dec1, _ := Decrypt(enc1)
	dec2, _ := Decrypt(enc2)
	if dec1 != token || dec2 != token {
		t.Error("different ciphertexts did not decrypt to same plaintext")
	}
}

func TestDeriveKeyConsistency(t *testing.T) {
	// deriveKey should return the same key on repeated calls
	key1, err := deriveKey()
	if err != nil {
		t.Fatalf("first deriveKey() error = %v", err)
	}

	key2, err := deriveKey()
	if err != nil {
		t.Fatalf("second deriveKey() error = %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("key length = %d, want 32 (AES-256)", len(key1))
	}

	for i := range key1 {
		if key1[i] != key2[i] {
			t.Fatal("deriveKey() returned different keys on repeated calls")
		}
	}
}

func TestGetMachineID(t *testing.T) {
	id, err := getMachineID()
	if err != nil {
		t.Fatalf("getMachineID() error = %v", err)
	}
	if id == "" {
		t.Error("getMachineID() returned empty string")
	}
}

func TestFallbackMachineID(t *testing.T) {
	id, err := fallbackMachineID()
	if err != nil {
		t.Fatalf("fallbackMachineID() error = %v", err)
	}
	if id == "" {
		t.Error("fallbackMachineID() returned empty string")
	}
}

func TestEncryptEmptyString(t *testing.T) {
	encrypted, err := Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt(\"\") error = %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if decrypted != "" {
		t.Errorf("Decrypt() = %q, want empty", decrypted)
	}
}

func TestDecryptInvalidData(t *testing.T) {
	_, err := Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Error("Decrypt() should error on invalid base64")
	}

	_, err = Decrypt("aGVsbG8=") // valid base64 but not valid ciphertext
	if err == nil {
		t.Error("Decrypt() should error on invalid ciphertext")
	}
}
