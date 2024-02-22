package handler

import (
	"net/http"
)

type requestFunc func(r *http.Request) (*http.Response, error)

//func login(t testing.TB, reqFn requestFunc) {
//	loginData := loginRequest{
//		Email:    "user@example.com",
//		Password: "1234",
//	}
//
//	loginDataAsJSON, err := json.Marshal(loginData)
//	require.NoError(t, err)
//
//	req, err := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginDataAsJSON))
//	require.NoError(t, err)
//
//	res, err := reqFn(req)
//	require.NoError(t, err)
//
//	require.Equal(t, http.StatusOK, res.StatusCode)
//}
