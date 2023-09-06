package utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

var log = GetLogger()

// executeRequest, creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
func SubmitRequest(req *http.Request, router *chi.Mux) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func SubmitRequestAndCheckResponse(t *testing.T, req *http.Request, router *chi.Mux, expectedResponseCode int) (res *httptest.ResponseRecorder) {
	res = SubmitRequest(req, router)
	if res.Code != expectedResponseCode {
		log.Error(req.URL.Path, " expected status code ", expectedResponseCode, ", got ", res.Code, ": ", res.Body.String())
		t.FailNow()
	}
	return res
}

// checkResponseCode is a simple utility to check the response code of the response
func CheckResponse(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func CheckError(t *testing.T, err error) {
	if err != nil {
		log.Error(err)
		t.Error(err)
		t.FailNow()
	}
}



func TestRequest(t *testing.T, router *chi.Mux, method string, url string, body any, expectedResponseCode int) (res *httptest.ResponseRecorder) {
	var bodyJSON bytes.Buffer
	err := json.NewEncoder(&bodyJSON).Encode(body)
	CheckError(t, err)
	req, err := http.NewRequest(method, url, &bodyJSON)
	CheckError(t, err)
	res = SubmitRequestAndCheckResponse(t, req, router, expectedResponseCode)
	return res
}

func TestRequestMultiPart(
	t *testing.T,
	router *chi.Mux,
	method string,
	url string,
	body *bytes.Buffer,
	contentType string,
	expectedResponseCode int,
) (res *httptest.ResponseRecorder) {
	req, err := http.NewRequest(method, url, body)
	CheckError(t, err)
	req.Header.Set("Content-Type", contentType)
	res = SubmitRequestAndCheckResponse(t, req, router, expectedResponseCode)
	return res
}


func TestRequestStr(t *testing.T, router *chi.Mux, method string, url string, body string, expectedResponseCode int) (res *httptest.ResponseRecorder) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
	CheckError(t, err)
	res = SubmitRequestAndCheckResponse(t, req, router, expectedResponseCode)
	return res
}
