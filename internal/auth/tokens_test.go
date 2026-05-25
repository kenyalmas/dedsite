package auth

import "testing"

func TestRandomTokenAndHashToken(t *testing.T) {
	token, err := RandomToken()
	if err != nil {
		t.Fatalf("RandomToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected random token")
	}
	if HashToken(token) == "" {
		t.Fatal("expected token hash")
	}
	if HashToken(token) == HashToken(token+"x") {
		t.Fatal("expected different tokens to hash differently")
	}
}
