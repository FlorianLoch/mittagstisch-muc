package handler

import (
	"encoding/json"
	"github.com/go-chi/render"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"net/http"
	"time"
)

func NewLoginRequiredMiddleware(isSessionAuthenticatedFn func(r *http.Request) bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isSessionAuthenticatedFn(r) {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Controller) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Very naive throttling to make brute force attacks a little harder
	select {
	case <-r.Context().Done():
		return
	case <-time.After(1 * time.Second):
		break
	}

	creds := &loginRequest{}

	if err := json.NewDecoder(r.Body).Decode(creds); err != nil {
		// We do not log the actual error to ensure that we do not leak any sensitive information
		c.logger.Warn().Msg("failed to decode login request")

		w.WriteHeader(http.StatusBadRequest)

		render.PlainText(w, r, "failed to decode login request")

		return
	}

	// For now there is only one user
	if creds.Username != c.cfg.AdminUsername {
		c.logger.Warn().Str("username", creds.Username).Msg("invalid username")

		// This is so naive that it still allows for timing attacks as we return early, but so be it
		time.Sleep((time.Duration(rand.Intn(1000)) + 200) * time.Millisecond)

		w.WriteHeader(http.StatusUnauthorized)

		render.PlainText(w, r, "invalid credentials")

		return
	}

	// CompareHashAndPassword performs its checks in constant time to avoid timing attacks
	if err := bcrypt.CompareHashAndPassword([]byte(c.cfg.AdminPassword), []byte(creds.Password)); err != nil {
		time.Sleep((time.Duration(rand.Intn(1000)) + 200) * time.Millisecond)

		w.WriteHeader(http.StatusUnauthorized)

		render.PlainText(w, r, "invalid credentials")

		return
	}

	sess := c.sessionStorage.SessionFromContext(r.Context())
	sess.LoggedIn = true
	sess.Username = creds.Username

	w.WriteHeader(http.StatusOK)

	render.PlainText(w, r, "auth successful")
}
