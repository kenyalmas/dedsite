package db

import (
	"database/sql"
	"errors"
	"time"

	"dedsite/internal/auth"
)

type AdminUser struct {
	ID       int64
	Username string
}

type AdminSession struct {
	User      AdminUser
	CSRFToken string
	ExpiresAt time.Time
}

func (s Store) EnsureDefaultAdmin() error {
	var count int
	if err := s.conn.QueryRow(`SELECT COUNT(*) FROM admin_users WHERE username = ?`, "admin").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := auth.HashPassword("password")
	if err != nil {
		return err
	}

	_, err = s.conn.Exec(`INSERT INTO admin_users (username, password_hash) VALUES (?, ?)`, "admin", hash)
	return err
}

func (s Store) AuthenticateAdmin(username string, password string) (AdminUser, bool, error) {
	var user AdminUser
	var hash string
	err := s.conn.QueryRow(`
		SELECT id, username, password_hash
		FROM admin_users
		WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AdminUser{}, false, nil
		}
		return AdminUser{}, false, err
	}

	return user, auth.VerifyPassword(hash, password), nil
}

func (s Store) CreateAdminSession(userID int64, duration time.Duration) (string, string, time.Time, error) {
	token, err := auth.RandomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	csrfToken, err := auth.RandomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}

	expires := time.Now().Add(duration).UTC()
	_, err = s.conn.Exec(`
		INSERT INTO admin_sessions (user_id, token_hash, csrf_hash, expires_at)
		VALUES (?, ?, ?, ?)
	`, userID, auth.HashToken(token), auth.HashToken(csrfToken), expires.Format(time.RFC3339))
	if err != nil {
		return "", "", time.Time{}, err
	}

	return token, csrfToken, expires, nil
}

func (s Store) AdminSessionForToken(token string) (AdminSession, bool, error) {
	var session AdminSession
	var expiresAt string
	err := s.conn.QueryRow(`
		SELECT admin_users.id, admin_users.username, admin_sessions.csrf_hash, admin_sessions.expires_at
		FROM admin_sessions
		JOIN admin_users ON admin_users.id = admin_sessions.user_id
		WHERE admin_sessions.token_hash = ?
			AND admin_sessions.expires_at > ?
	`, auth.HashToken(token), time.Now().UTC().Format(time.RFC3339)).Scan(&session.User.ID, &session.User.Username, &session.CSRFToken, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AdminSession{}, false, nil
		}
		return AdminSession{}, false, err
	}
	session.ExpiresAt, err = time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return AdminSession{}, false, err
	}

	return session, true, nil
}

func (s Store) DeleteAdminSession(token string) error {
	_, err := s.conn.Exec(`DELETE FROM admin_sessions WHERE token_hash = ?`, auth.HashToken(token))
	return err
}

func (s Store) SetAdminSessionCSRF(token string, csrfToken string) error {
	_, err := s.conn.Exec(`
		UPDATE admin_sessions
		SET csrf_hash = ?
		WHERE token_hash = ?
	`, auth.HashToken(csrfToken), auth.HashToken(token))
	return err
}
