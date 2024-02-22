package handler

import (
	"context"
	"github.com/felixge/httpsnoop"
	"github.com/gorilla/securecookie"
	"github.com/rs/zerolog/log"
	"net/http"
)

type ctxKey string

var ctxKeySession ctxKey = "session"

type SessionStorage[T any] struct {
	cookieName      string
	insecureAllowed bool
	sc              *securecookie.SecureCookie
}

func NewSessionStorage[T any](cookieName string, hashKey, blockKey []byte, insecureAllowed bool) *SessionStorage[T] {
	return &SessionStorage[T]{
		cookieName:      cookieName,
		insecureAllowed: insecureAllowed,
		sc:              securecookie.New(hashKey, blockKey),
	}
}

func (s *SessionStorage[T]) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := new(T)

			cookie, err := r.Cookie(s.cookieName)
			if err == nil {
				if err := s.sc.Decode(s.cookieName, cookie.Value, &session); err != nil {
					log.Error().Err(err).Msg("decoding session cookie")
				}
			}

			r = r.WithContext(context.WithValue(r.Context(), ctxKeySession, session))

			// We want to store the session in an encrypted cookie on the client, we do not want to keep the state on
			// the server.
			// Therefore, to solve this elegantly using middlewares, we need to wrap the response writer to set the
			// cookie after the response has been written/the handler has been executed.

			wrappedW := httpsnoop.Wrap(w, httpsnoop.Hooks{
				WriteHeader: func(wrappedFn httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
					encoded, err := s.sc.Encode(s.cookieName, session)
					if err != nil {
						log.Error().Err(err).Msg("encoding session cookie")
					}

					cookie := &http.Cookie{
						Name:     s.cookieName,
						Secure:   !s.insecureAllowed,
						Path:     "/",
						HttpOnly: true, // Client code should not mess with this cookie
						Value:    encoded,
					}

					http.SetCookie(w, cookie)

					return wrappedFn
				}})

			next.ServeHTTP(wrappedW, r)
		})
	}
}

func (s *SessionStorage[T]) SessionFromContext(ctx context.Context) *T {
	val, ok := ctx.Value(ctxKeySession).(*T)
	if !ok {
		log.Panic().Msg("retrieving session from context")
	}

	return val
}
