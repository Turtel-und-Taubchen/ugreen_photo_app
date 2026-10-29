package api

import (
	"context"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

type userKey struct{}

// UserFromContext returns the signed-in user that RequireSession attached to the request.
func UserFromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(userKey{}).(string)
	return u, ok && u != ""
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// isAPIPath classifies by the cleaned path, so `/api//x` or `/x/../api/y` still count as API
// and get JSON 401 / CSRF treatment instead of a page redirect.
func isAPIPath(p string) bool {
	c := path.Clean("/" + p)
	return c == "/api" || strings.HasPrefix(c, "/api/")
}

// isOpen is the complete list of what may be reached without a session. Matching is exact on
// the request path as sent: a path that is not canonical (`..`, `//`, `.`, trailing slash,
// non-canonical percent-encoding) is never open, even if it would resolve to an open target.
func isOpen(r *http.Request, static fs.FS) bool {
	p := r.URL.Path
	if r.URL.RawPath != "" || p == "" || path.Clean(p) != p {
		return false
	}
	getLike := r.Method == http.MethodGet || r.Method == http.MethodHead
	switch {
	case p == "/api/login":
		return r.Method == http.MethodPost
	case p == "/api/health", p == "/login":
		return getLike
	case !getLike || p == "/" || isAPIPath(p):
		return false
	}
	// index.html is the protected app shell; every other embedded regular file is a public asset.
	name := p[1:]
	if name == "index.html" || !fs.ValidPath(name) {
		return false
	}
	info, err := fs.Stat(static, name)
	return err == nil && info.Mode().IsRegular()
}

// RequireSession denies by default. Without a valid session: open paths pass, API paths get
// 401 JSON, page GET/HEAD requests are redirected to /login, and any other method on a page
// path gets 401 JSON too (a redirect would invite clients to replay a write as GET).
func (a *authAPI) RequireSession(static fs.FS, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, ok := a.sessionUser(w, r); ok {
			if r.URL.Path == "/login" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
			return
		}
		switch {
		case isOpen(r, static):
			next.ServeHTTP(w, r)
		case !isAPIPath(r.URL.Path) && (r.Method == http.MethodGet || r.Method == http.MethodHead):
			http.Redirect(w, r, "/login", http.StatusFound)
		default:
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		}
	})
}

// CSRF refuses state-changing API requests that lack X-Requested-With: a cross-site form
// cannot set that header, and a cross-site fetch that sets it needs a CORS preflight we never
// grant. It runs inside RequireSession, so an unauthenticated write is a 401, not a 403.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isSafeMethod(r.Method) && isAPIPath(r.URL.Path) && r.Header.Get("X-Requested-With") == "" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "missing X-Requested-With header"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
