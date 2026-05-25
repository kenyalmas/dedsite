package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestGoogleOAuthConfigNormalizesAllowedEmailHashes(t *testing.T) {
	config := newGoogleOAuthConfig(GoogleOAuthConfig{
		ClientID:           "client",
		ClientSecret:       "secret",
		AllowedEmailHashes: []string{" 258D8DC916DB8CEA2CAFB6C3CD0CB0246EFE061421DBD83EC3A350428CABDA4F ", "", "5b71ed5f946240dc76f3b7c24bdcbbc3528284ec5f4519249fb702686f0df5b8"},
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
			ClientID:           "client",
			ClientSecret:       "secret",
			AllowedEmailHashes: []string{"258d8dc916db8cea2cafb6c3cd0cb0246efe061421dbd83ec3a350428cabda4f"},
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

func TestGoogleRedirectURIUsesConfiguredPublicOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "http://internal.example/admin/login", nil)
	handler := Handler{
		googleOAuth: newGoogleOAuthConfig(GoogleOAuthConfig{
			PublicOrigin: "https://portfolio.example/",
		}),
	}

	if got := handler.googleRedirectURI(req); got != "https://portfolio.example/admin/login/google/callback" {
		t.Fatalf("googleRedirectURI() with configured origin = %q", got)
	}
}

func TestGoogleRedirectURIAddsHTTPSForBarePublicOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "http://internal.example/admin/login", nil)
	handler := Handler{
		googleOAuth: newGoogleOAuthConfig(GoogleOAuthConfig{
			PublicOrigin: "portfolio.example",
		}),
	}

	if got := handler.googleRedirectURI(req); got != "https://portfolio.example/admin/login/google/callback" {
		t.Fatalf("googleRedirectURI() with bare configured origin = %q", got)
	}
}
