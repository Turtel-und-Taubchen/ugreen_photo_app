package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"time"
)

const (
	// SessionTTL is the idle lifetime of a session.
	SessionTTL = 30 * 24 * time.Hour
	// sessionRenewAfter throttles sliding renewal so a busy client does not write on every request.
	sessionRenewAfter = 24 * time.Hour
)

// SessionStore keeps sessions in app.db. Only sha256(token) is persisted, so a leaked
// database cannot be replayed as cookies.
type SessionStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewSessionStore(db *sql.DB, now func() time.Time) *SessionStore {
	if now == nil {
		now = time.Now
	}
	return &SessionStore{db: db, now: now}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Create starts a session and returns the opaque token and its expiry.
func (s *SessionStore) Create(user string) (string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := s.now()
	expires := now.Add(SessionTTL)
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now.Unix()); err != nil {
		return "", time.Time{}, err
	}
	if _, err := s.db.Exec(`INSERT INTO sessions (id_hash, username, expires_at, renewed_at) VALUES (?, ?, ?, ?)`,
		hashToken(token), user, expires.Unix(), now.Unix()); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// Lookup resolves a token to its user. newExpires is non-zero only when this call renewed
// the session (at most once per 24 h), so the caller re-issues the cookie only then.
func (s *SessionStore) Lookup(token string) (user string, newExpires time.Time, ok bool) {
	now := s.now()
	h := hashToken(token)
	var expires, renewed int64
	err := s.db.QueryRow(`SELECT username, expires_at, renewed_at FROM sessions WHERE id_hash = ?`, h).
		Scan(&user, &expires, &renewed)
	if err != nil {
		return "", time.Time{}, false
	}
	if expires <= now.Unix() {
		s.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now.Unix())
		return "", time.Time{}, false
	}
	if now.Unix()-renewed >= int64(sessionRenewAfter/time.Second) {
		newExpires = now.Add(SessionTTL)
		if _, err := s.db.Exec(`UPDATE sessions SET expires_at = ?, renewed_at = ? WHERE id_hash = ?`,
			newExpires.Unix(), now.Unix(), h); err != nil {
			newExpires = time.Time{}
		}
	}
	return user, newExpires, true
}

// Delete ends a session; an unknown token is not an error.
func (s *SessionStore) Delete(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id_hash = ?`, hashToken(token))
	return err
}
