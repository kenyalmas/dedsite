package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestGoogleOAuthConfigNormalizesAllowedEmails(t *testing.T) {
	config := newGoogleOAuthConfig(GoogleOAuthConfig{
		ClientID:      "client",
		ClientSecret:  "secret",
		AllowedEmails: []string{" Admin@Example.COM ", "", "other@example.com"},
	})

	if !config.enabled() {
		t.Fatal("expected config to be enabled")
	}
	if !config.allows("admin@example.com") {
		t.Fatal("expected lowercase trimmed email to be allowed")
	}
	if config.allows("missing@example.com") {
		t.Fatal("did not expect unlisted email to be allowed")
	}
}

func TestAllowsGoogleUser(t *testing.T) {
	handler := Handler{
		googleOAuth: newGoogleOAuthConfig(GoogleOAuthConfig{
			ClientID:      "client",
			ClientSecret:  "secret",
			AllowedEmails: []string{"admin@example.com"},
		}),
	}

	tests := []struct {
		name     string
		userInfo googleUserInfo
		want     bool
	}{
		{
			name: "allows verified allowlisted email",
			userInfo: googleUserInfo{
				Email:         " Admin@Example.COM ",
				EmailVerified: true,
			},
			want: true,
		},
		{
			name: "allows legacy verified field",
			userInfo: googleUserInfo{
				Email:               "admin@example.com",
				LegacyEmailVerified: true,
			},
			want: true,
		},
		{
			name: "rejects unverified email",
			userInfo: googleUserInfo{
				Email: "admin@example.com",
			},
		},
		{
			name: "rejects non-allowlisted email",
			userInfo: googleUserInfo{
				Email:         "guest@example.com",
				EmailVerified: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.allowsGoogleUser(tt.userInfo); got != tt.want {
				t.Fatalf("allowsGoogleUser() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestGoogleRedirectURI(t *testing.T) {
	req := httptest.NewRequest("GET", "http://localhost:8090/admin/login", nil)
	handler := Handler{}
	if got := handler.googleRedirectURI(req); got != "http://localhost:8090/admin/login/google/callback" {
		t.Fatalf("googleRedirectURI() = %q", got)
	}

	req.Header.Set("X-Forwarded-Proto", "https")
	handler.trustProxyHeaders = true
	if got := handler.googleRedirectURI(req); got != "https://localhost:8090/admin/login/google/callback" {
		t.Fatalf("googleRedirectURI() with trusted proxy = %q", got)
	}
}
