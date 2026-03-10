package platform

import (
	"runtime"
	"testing"
)

func TestCopyToClipboardEmpty(t *testing.T) {
	// Should not panic even if clipboard tool is available
	err := CopyToClipboard("")
	// May succeed or fail depending on environment, just don't panic
	_ = err
}

func TestOpenBrowser(t *testing.T) {
	// Just verify it doesn't panic — don't actually open a browser in tests
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		// We can't really test this without opening a browser,
		// but verify the function exists and handles the URL
		_ = OpenBrowser
	}
}

func TestKeychainAvailable(t *testing.T) {
	available := KeychainAvailable()
	// Result depends on environment, just verify no panic
	_ = available
}

func TestKeychainDeleteNonexistent(t *testing.T) {
	// Should not error when deleting nonexistent entry
	err := KeychainDelete("gvm-test-nonexistent-profile-xyz")
	if err != nil {
		t.Errorf("KeychainDelete() for nonexistent entry should return nil, got %v", err)
	}
}
