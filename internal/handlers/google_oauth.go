package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dedsite/internal/auth"
)

const googleOAuthStateCookie = "dedsite_google_oauth_state"

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type googleUserInfo struct {
	Email               string `json:"email"`
	EmailVerified       bool   `json:"email_verified"`
	LegacyEmailVerified bool   `json:"verified_email"`
}

func (u googleUserInfo) verified() bool {
	return u.EmailVerified || u.LegacyEmailVerified
}

func (h Handler) AdminGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.googleOAuth.enabled() {
		http.NotFound(w, r)
		return
	}

	state, err := auth.RandomToken()
	if err != nil {
		http.Error(w, "Could not prepare Google login", http.StatusInternalServerError)
		return
	}

	secureCookie := isSecureRequest(r, h.trustProxyHeaders)
	http.SetCookie(w, &http.Cookie{
		Name:     googleOAuthStateCookie,
		Value:    state,
		Path:     "/admin/login/google",
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	params := url.Values{}
	params.Set("client_id", h.googleOAuth.clientID)
	params.Set("redirect_uri", h.googleRedirectURI(r))
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	http.Redirect(w, r, "https://accounts.google.com/o/oauth2/v2/auth?"+params.Encode(), http.StatusFound)
}

func (h Handler) AdminGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !h.googleOAuth.enabled() {
		http.NotFound(w, r)
		return
	}

	stateCookie, err := r.Cookie(googleOAuthStateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "Invalid Google login state", http.StatusBadRequest)
		return
	}
	expireCookieAtPath(w, googleOAuthStateCookie, true, isSecureRequest(r, h.trustProxyHeaders), "/admin/login/google")

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Error(w, "Missing Google login code", http.StatusBadRequest)
		return
	}

	userInfo, err := h.fetchGoogleUserInfo(r, code)
	if err != nil {
		http.Error(w, "Could not verify Google login", http.StatusBadGateway)
		return
	}
	email := strings.ToLower(strings.TrimSpace(userInfo.Email))
	if !h.allowsGoogleUser(userInfo) {
		http.Error(w, "Google account is not allowed", http.StatusForbidden)
		return
	}

	user, err := h.store.EnsureOAuthAdmin(email)
	if err != nil {
		http.Error(w, "Could not prepare admin account", http.StatusInternalServerError)
		return
	}
	if err := h.startAdminSession(w, r, user); err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h Handler) allowsGoogleUser(userInfo googleUserInfo) bool {
	email := strings.ToLower(strings.TrimSpace(userInfo.Email))
	return email != "" && userInfo.verified() && h.googleOAuth.allows(email)
}

func (h Handler) fetchGoogleUserInfo(r *http.Request, code string) (googleUserInfo, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", h.googleOAuth.clientID)
	form.Set("client_secret", h.googleOAuth.clientSecret)
	form.Set("redirect_uri", h.googleRedirectURI(r))
	form.Set("grant_type", "authorization_code")

	tokenResponse, err := http.PostForm("https://oauth2.googleapis.com/token", form)
	if err != nil {
		return googleUserInfo{}, err
	}
	defer tokenResponse.Body.Close()
	if tokenResponse.StatusCode != http.StatusOK {
		return googleUserInfo{}, fmt.Errorf("google token endpoint returned %s", tokenResponse.Status)
	}

	var token googleTokenResponse
	if err := json.NewDecoder(tokenResponse.Body).Decode(&token); err != nil {
		return googleUserInfo{}, err
	}
	if token.AccessToken == "" {
		return googleUserInfo{}, errors.New("google token response missing access token")
	}

	userRequest, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return googleUserInfo{}, err
	}
	userRequest.Header.Set("Authorization", "Bearer "+token.AccessToken)

	userResponse, err := http.DefaultClient.Do(userRequest)
	if err != nil {
		return googleUserInfo{}, err
	}
	defer userResponse.Body.Close()
	if userResponse.StatusCode != http.StatusOK {
		return googleUserInfo{}, fmt.Errorf("google userinfo endpoint returned %s", userResponse.Status)
	}

	var userInfo googleUserInfo
	if err := json.NewDecoder(userResponse.Body).Decode(&userInfo); err != nil {
		return googleUserInfo{}, err
	}
	return userInfo, nil
}

func (h Handler) googleRedirectURI(r *http.Request) string {
	scheme := "http"
	if isSecureRequest(r, h.trustProxyHeaders) {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/admin/login/google/callback"
}

func expireCookieAtPath(w http.ResponseWriter, name string, httpOnly bool, secure bool, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
