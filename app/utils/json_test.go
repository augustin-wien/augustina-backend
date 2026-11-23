package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

type failingResponseWriter struct{ *httptest.ResponseRecorder }

func (failingResponseWriter) Write([]byte) (int, error) { return 0, errors.New("connection closed") }

func TestReadJSON(t *testing.T) {
	read := func(body string, into any) error {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		return ReadJSON(httptest.NewRecorder(), req, into)
	}
	var out struct{ Name string }

	require.NoError(t, read(`{"Name":"a","Unknown":1}`, &out))
	require.Equal(t, "a", out.Name)

	require.Error(t, read(`{not json`, &out))
	require.Error(t, read(`{"Name":1}`, &out), "wrong type")
	require.Error(t, read(`{"Name":"a"}{"Name":"b"}`, &out), "two JSON values")
	require.Error(t, read(`{"Name":"`+strings.Repeat("x", 2<<20)+`"}`, &out), "larger than one megabyte")

	orig := config.Config.Development
	t.Cleanup(func() { config.Config.Development = orig })
	config.Config.Development = true
	require.NoError(t, read(`{"Name":"dev"}`, &out))
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	require.NoError(t, WriteJSON(rr, http.StatusCreated, "<b>", "data"))
	require.Equal(t, http.StatusCreated, rr.Code)
	require.JSONEq(t, `{"data":"<b>"}`, rr.Body.String())

	require.Error(t, WriteJSON(httptest.NewRecorder(), http.StatusOK, make(chan int)))
	require.Error(t, WriteJSON(httptest.NewRecorder(), http.StatusOK, make(chan int), "data"))
	require.Error(t, WriteJSON(failingResponseWriter{httptest.NewRecorder()}, http.StatusOK, "x"))

	rr = httptest.NewRecorder()
	ErrorJSON(rr, errors.New("boom"))
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.NotPanics(t, func() { ErrorJSON(failingResponseWriter{httptest.NewRecorder()}, errors.New("boom"), 500) })
}
