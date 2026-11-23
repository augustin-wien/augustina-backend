package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/go-chi/chi/v5"
)

var log = GetLogger()

// testClientCounter numbers the client addresses of test requests
var testClientCounter atomic.Uint32

// SubmitRequest creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
// Requests without an explicit client address each get their own, so the
// per-IP rate limit of the router does not add up over a whole test run.
func SubmitRequest(req *http.Request, router *chi.Mux) *httptest.ResponseRecorder {
	if req.RemoteAddr == "" || req.RemoteAddr == "192.0.2.1:1234" {
		n := testClientCounter.Add(1)
		req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", n>>16&0xff, n>>8&0xff, n&0xff)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// SubmitRequestAndCheckResponse is a simple utility to check the response code of the response
func SubmitRequestAndCheckResponse(t *testing.T, req *http.Request, router *chi.Mux, expectedResponseCode int) (res *httptest.ResponseRecorder) {
	res = SubmitRequest(req, router)
	if res.Code != expectedResponseCode {
		log.Error(req.URL.Path, " expected status code ", expectedResponseCode, ", got ", res.Code, ": ", res.Body.String())
		t.FailNow()
	}
	return res
}

// CheckResponse is a simple utility to check the response code of the response
func CheckResponse(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

// CheckError is a simple utility to check the error
func CheckError(t *testing.T, err error) {
	if err != nil {
		log.Error("CheckError: ", err)
		t.Error(err)
		t.FailNow()
	}
}

// TestRequest is a simple utility to test a request
func TestRequest(t *testing.T, router *chi.Mux, method string, url string, body any, expectedResponseCode int) (res *httptest.ResponseRecorder) {
	var bodyJSON bytes.Buffer
	err := json.NewEncoder(&bodyJSON).Encode(body)
	CheckError(t, err)
	req, err := http.NewRequest(method, url, &bodyJSON)
	CheckError(t, err)
	res = SubmitRequestAndCheckResponse(t, req, router, expectedResponseCode)
	return res
}

// TestRequestWithAuth is a simple utility to test a request including auth
func TestRequestWithAuth(t *testing.T, router *chi.Mux, method string, url string, body any, expectedResponseCode int, auth *gocloak.JWT) (res *httptest.ResponseRecorder) {
	var bodyJSON bytes.Buffer
	err := json.NewEncoder(&bodyJSON).Encode(body)
	CheckError(t, err)
	req, err := http.NewRequest(method, url, &bodyJSON)
	CheckError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	res = SubmitRequestAndCheckResponse(t, req, router, expectedResponseCode)
	return res
}

// TestRequestMultiPartWithAuth is a simple utility to test a request with multipart/form-data including auth
func TestRequestMultiPartWithAuth(
	t *testing.T,
	router *chi.Mux,
	method string,
	url string,
	body *bytes.Buffer,
	contentType string,
	expectedResponseCode int,
	auth *gocloak.JWT,
) (res *httptest.ResponseRecorder) {
	req, err := http.NewRequest(method, url, body)
	CheckError(t, err)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	res = SubmitRequestAndCheckResponse(t, req, router, expectedResponseCode)
	return res
}

// TestRequestStr is a simple utility to test a request with a string body
func TestRequestStr(t *testing.T, router *chi.Mux, method string, url string, body string, expectedResponseCode int) (res *httptest.ResponseRecorder) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
	CheckError(t, err)
	res = SubmitRequestAndCheckResponse(t, req, router, expectedResponseCode)
	return res
}

// TestRequestStrWithAuth is a simple utility to test a request with a string body
func TestRequestStrWithAuth(t *testing.T, router *chi.Mux, method string, url string, body string, expectedResponseCode int, auth *gocloak.JWT) (res *httptest.ResponseRecorder) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
	CheckError(t, err)
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	res = SubmitRequestAndCheckResponse(t, req, router, expectedResponseCode)
	return res
}
