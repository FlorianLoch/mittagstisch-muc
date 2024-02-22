package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/securecookie"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func TestSessionStorage(t *testing.T) {
	r := chi.NewRouter()

	sessionStorage := NewSessionStorage[session](
		"session",
		securecookie.GenerateRandomKey(32),
		securecookie.GenerateRandomKey(32),
		true)

	r.Use(sessionStorage.Middleware())

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		sess := sessionStorage.SessionFromContext(r.Context())

		sess.LoggedIn = true
		sess.Username = "me"

		w.WriteHeader(http.StatusNoContent)
	})

	var (
		loggedIn bool
		username string
	)

	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		sess := sessionStorage.SessionFromContext(r.Context())

		loggedIn = sess.LoggedIn
		username = sess.Username
	})

	server := httptest.NewServer(r)
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := server.Client()
	client.Jar = jar

	res, err := client.Post(server.URL+"/login", "", nil)
	require.NoError(t, err)

	require.Equal(t, http.StatusNoContent, res.StatusCode)

	res, err = client.Get(server.URL + "/status")
	require.NoError(t, err)

	require.True(t, loggedIn)
	require.Equal(t, "me", username)
}
