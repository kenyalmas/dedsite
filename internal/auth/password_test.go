package auth

import "testing"

func TestPBKDF2KeyInvalidParams(t *testing.T) {
	if got := pbkdf2Key([]byte("pw"), []byte("salt"), 0, 32); got != nil {
		t.Fatal("expected nil key for invalid iteration count")
	}
	if got := pbkdf2Key([]byte("pw"), []byte("salt"), 1, 0); got != nil {
		t.Fatal("expected nil key for invalid key length")
	}
}
