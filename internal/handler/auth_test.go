package handler

import (
	"bytes"
	"encoding/json"
	"github.com/florianloch/mittagstisch/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/securecookie"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func TestAuth_LoginRequiredMiddleware(t *testing.T) {
	r := chi.NewRouter()

	sessionStorage := NewSessionStorage[session](
		"session",
		securecookie.GenerateRandomKey(32),
		securecookie.GenerateRandomKey(32),
		true)

	loginRequired := NewLoginRequiredMiddleware(func(r *http.Request) bool {
		return sessionStorage.SessionFromContext(r.Context()).LoggedIn
	})

	r.Use(sessionStorage.Middleware())

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		sess := sessionStorage.SessionFromContext(r.Context())

		sess.LoggedIn = true

		w.WriteHeader(http.StatusOK)
	})

	r.With(loginRequired).Get("/only_for_logged_in_users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(r)
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := server.Client()
	client.Jar = jar

	res, err := client.Get(server.URL + "/only_for_logged_in_users")
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)

	res, err = client.Post(server.URL+"/login", "", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	res, err = client.Get(server.URL + "/only_for_logged_in_users")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
}

func TestController_auth(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.DefaultCost)
	require.NoError(t, err)

	logger := zerolog.New(zerolog.NewTestWriter(t))

	c := NewController(config.Config{
		AdminUsername:          "admin",
		AdminPassword:          string(hashedPassword),
		SessionInsecureAllowed: true,
	}, nil, &logger)

	server := httptest.NewServer(c)
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := server.Client()
	client.Jar = jar

	res, err := client.Get(server.URL + "/api/v1/auth/csrf?page=page1")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	csrfToken := res.Header.Get("X-CSRF-Token")
	require.NotEmpty(t, csrfToken)

	wrongLoginData := loginRequest{
		Username: "admin",
		Password: "4321",
	}

	wrongLoginDataAsJSON, err := json.Marshal(wrongLoginData)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/auth/login?page=page1", bytes.NewReader(wrongLoginDataAsJSON))
	require.NoError(t, err)
	req.Header.Set("X-CSRF-Token", csrfToken)

	res, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)

	correctLoginData := loginRequest{
		Username: "admin",
		Password: "1234",
	}

	correctLoginDataAsJSON, err := json.Marshal(correctLoginData)
	require.NoError(t, err)

	req, err = http.NewRequest(http.MethodPost, server.URL+"/api/v1/auth/login?page=page1", bytes.NewReader(correctLoginDataAsJSON))
	require.NoError(t, err)
	req.Header.Set("X-CSRF-Token", csrfToken)

	res, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	sessionCookie := sessionCookie(c.sessionStorage.cookieName, res)
	require.NotNil(t, sessionCookie)

	var sess session

	require.NoError(t, c.sessionStorage.sc.Decode(c.sessionStorage.cookieName, sessionCookie.Value, &sess))

	require.True(t, sess.LoggedIn)
	require.Equal(t, "admin", sess.Username)
}

func sessionCookie(cookieName string, res *http.Response) *http.Cookie {
	for _, cookie := range res.Cookies() {
		if cookie.Name == cookieName {
			return cookie
		}
	}

	return nil
}
