package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// executeRequest, creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
func executeRequest(req *http.Request, router *chi.Mux) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

// checkResponseCode is a simple utility to check the response code
// of the response
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}


func writeJSON(w http.ResponseWriter, status int, data interface{}, wrap ...string) error {
	// out will hold the final version of the json to send to the client
	var out []byte

	// decide if we wrap the json payload in an overall json tag
	if len(wrap) > 0 {
		// wrapper
		wrapper := make(map[string]interface{})
		wrapper[wrap[0]] = data
		jsonBytes, err := json.Marshal(wrapper)
		if err != nil {
			return err
		}
		out = jsonBytes
	} else {
		// wrapper
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		out = jsonBytes
	}

	// set the content type & status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// write the json out
	_, err := w.Write(out)
	if err != nil {
		return err
	}
	return nil
}

func errorJSON(w http.ResponseWriter, err error, status ...int) {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	type jsonError struct {
		Message string `json:"message"`
	}

	theError := jsonError{
		Message: err.Error(),
	}

	_ = writeJSON(w, statusCode, theError, "error")
}

func readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1024 * 1024 // one megabyte
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// attempt to decode the data
	err := dec.Decode(data)
	if err != nil {
		return err
	}

	// make sure only one JSON value in payload
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}
