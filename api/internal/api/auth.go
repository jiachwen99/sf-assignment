package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
	"github.com/jiachwen99/sf-assignment/api/internal/store"
)

const sessionCookie = "session"

type credentials struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

/*
 * HttpOnly so script cannot read the token, SameSite=Lax so another site cannot
 * make an authenticated write on the user's behalf, and Path=/ so it travels
 * with every request rather than only the ones under /api.
 *
 * Secure is set only when the request arrived over TLS. Setting it
 * unconditionally would mean the cookie is silently dropped over plain HTTP,
 * and this runs over plain HTTP in development.
 */
func (s *Server) setSession(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func (s *Server) clearSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	body, err := decode[credentials](r)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	fields := map[string]string{}
	if !strings.Contains(body.Email, "@") {
		fields["email"] = "That does not look like an email address"
	}
	if strings.TrimSpace(body.Name) == "" {
		fields["name"] = "Name must not be empty"
	}
	// Length rather than a composition rule. A long passphrase beats a short
	// one with a digit bolted on, and rules that forbid characters push people
	// towards worse passwords.
	if len(body.Password) < 8 {
		fields["password"] = "Password must be at least 8 characters"
	}
	if len(fields) > 0 {
		s.fail(w, r, &domain.ValidationError{Fields: fields})
		return
	}

	user, err := s.svc.Register(r.Context(), body.Email, body.Name, body.Password)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	// Signed in immediately. Registering and then being asked to log in is a
	// step that exists only because the two were built separately.
	token, expires, err := s.svc.StartSession(r.Context(), user.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.setSession(w, r, token, expires)
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	body, err := decode[credentials](r)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	user, err := s.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	token, expires, err := s.svc.StartSession(r.Context(), user.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.setSession(w, r, token, expires)
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if err := s.svc.EndSession(r.Context(), cookie.Value); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	s.clearSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// Null rather than an error when nobody is signed in. Not being signed in is a
// state the application supports, not a failure.
func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type userKey struct{}

func userFrom(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(userKey{}).(store.User)
	return u, ok
}

/*
 * Resolves the session on every request and carries the result forward. It
 * never refuses one.
 *
 * Accounts supply identity, never separation: everyone sees the same list, so
 * there is nothing here to protect by rejecting anonymous callers. What signing
 * in buys is that the history can say who did something. Guarding the routes
 * would contradict the requirement that users share one list.
 */
func (s *Server) withSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, err := s.svc.UserForSession(r.Context(), cookie.Value)
		if err != nil {
			// An expired or unknown token is treated as not signed in rather
			// than as an error, so a stale cookie does not break the app.
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userKey{}, user)
		ctx = store.WithActor(ctx, user.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
