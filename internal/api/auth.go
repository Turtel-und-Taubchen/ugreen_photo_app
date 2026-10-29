package api

import (
	"encoding/json"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"photocurator/internal/auth"
)

const (
	sessionCookie = "pc_session"
	loginFailed   = "Anmeldung fehlgeschlagen. Name oder Passwort falsch."
	maxLoginBody  = 8 << 10
)

// AuthDeps is what the session gate and the login endpoints need.
type AuthDeps struct {
	Users    auth.Users
	Sessions *auth.SessionStore
	Limiter  *auth.LoginLimiter
	// Verify defaults to auth.VerifyPassword; tests wrap it to count password checks.
	Verify func(phc, pw string) (bool, error)
}

type authAPI struct {
	AuthDeps
	// dummyHash is checked for unknown user names so they cost exactly one Argon2 run, like a
	// wrong password; computed eagerly so the first such login is not slower.
	dummyHash string
}

func newDummyHash() string {
	if h, err := auth.HashPassword("photocurator-dummy"); err == nil {
		return h
	}
	return "$argon2id$v=19$m=19456,t=2,p=1$c29tZXNhbHRzb21lc2FsdA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
}

func newAuthAPI(d AuthDeps) *authAPI {
	if d.Verify == nil {
		d.Verify = auth.VerifyPassword
	}
	return &authAPI{AuthDeps: d, dummyHash: newDummyHash()}
}

// clientIP uses only the TCP peer address: X-Forwarded-For is client-controlled and there is
// no trusted proxy in front of the app, so honouring it would let a client reset its lockout.
func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// Secure is deliberately off: the app is served over plain HTTP inside the home network.
func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/",
		MaxAge:   int(auth.SessionTTL / time.Second),
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
}

// sessionUser resolves the cookie to a user still listed in users.conf and re-issues the
// cookie when the store slid the expiry forward.
func (a *authAPI) sessionUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return "", false
	}
	user, newExpires, ok := a.Sessions.Lookup(c.Value)
	if !ok {
		return "", false
	}
	if _, listed := a.Users[user]; !listed {
		_ = a.Sessions.Delete(c.Value)
		return "", false
	}
	if !newExpires.IsZero() {
		setSessionCookie(w, c.Value)
	}
	return user, true
}

func (a *authAPI) login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if ok, wait := a.Limiter.Allowed(ip); !ok {
		secs := max(int(math.Ceil(wait.Seconds())), 1)
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "Zu viele Fehlversuche. Bitte in " + strconv.Itoa(secs) + " Sekunden erneut versuchen.",
		})
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxLoginBody)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	hash, known := a.Users[req.Username]
	if !known {
		hash = a.dummyHash
	}
	ok, err := a.Verify(hash, req.Password)
	if err != nil {
		slog.Error("password verification failed", "err", err)
	}
	if !known || !ok || err != nil {
		a.Limiter.Fail(ip)
		slog.Info("login failed", "ip", ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": loginFailed})
		return
	}
	a.Limiter.Success(ip)
	token, _, err := a.Sessions.Create(req.Username)
	if err != nil {
		slog.Error("session create", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]string{"username": req.Username})
}

func (a *authAPI) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := a.Sessions.Delete(c.Value); err != nil {
			slog.Error("session delete", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *authAPI) me(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"username": user})
}
