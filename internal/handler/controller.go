package handler

import (
	"crypto/sha256"
	"github.com/florianloch/mittagstisch/internal/config"
	"github.com/florianloch/mittagstisch/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
	"github.com/rs/zerolog"
	"net/http"
)

type session struct {
	LoggedIn bool
	Username string
}

type Controller struct {
	chi.Router
	logger         *zerolog.Logger
	cfg            config.Config
	db             *database.DB
	sessionStorage *SessionStorage[session]
}

func NewController(cfg config.Config, db *database.DB, logger *zerolog.Logger) *Controller {
	r := chi.NewRouter()

	sessionStorage := NewSessionStorage[session](
		"session",
		make32Bytes(cfg.SessionHashKey),
		make32Bytes(cfg.SessionBlockKey),
		cfg.SessionInsecureAllowed)

	c := &Controller{
		Router:         r,
		logger:         logger,
		cfg:            cfg,
		db:             db,
		sessionStorage: sessionStorage,
	}

	r.Use(middleware.Recoverer, middleware.Compress(5), middleware.RealIP, middleware.Logger)

	loginRequired := NewLoginRequiredMiddleware(func(r *http.Request) bool {
		ctx := r.Context()

		sess := sessionStorage.SessionFromContext(ctx)

		return sess.LoggedIn
	})

	sessionMiddleware := sessionStorage.Middleware()
	csrfMiddleware := csrf.Protect(
		make32Bytes(cfg.CSRFKey),
		csrf.Secure(!cfg.SessionInsecureAllowed),
		csrf.Path("/api/v1"))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Use(sessionMiddleware)
			r.Use(csrfMiddleware)

			r.Get("/csrf", handleCSRF)
			r.Post("/login", c.handleLogin)
		})

		r.With(loginRequired).Route("/admin", func(r chi.Router) {
			r.Use(sessionMiddleware)
			r.Use(csrfMiddleware)

			// TODO
		})
	})

	return c
}

func handleCSRF(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("X-CSRF-Token", csrf.Token(r))

	rw.WriteHeader(http.StatusOK)
}

func make32Bytes(key string) []byte {
	if len(key) == 32 {
		return []byte(key)
	}

	hashed := sha256.Sum256([]byte(key))

	return hashed[:]
}
