package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// TestRequestLogger ensures the RequestLogger middleware forwards requests to
// the next handler and that a request-id is available in the request context.
func TestRequestLogger(t *testing.T) {
	r := chi.NewRouter()
	// add chi's request id middleware so our RequestLogger can read an id
	r.Use(chimiddleware.RequestID)
	r.Use(RequestLogger)

	r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		id := chimiddleware.GetReqID(r.Context())
		if id == "" {
			t.Error("expected request id to be present in context")
		}
		w.Header().Set("X-ECHO-REQID", id)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello"))
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusCreated)
	}
	if body := rr.Body.String(); body != "hello" {
		t.Fatalf("unexpected body: %q", body)
	}
	if rid := rr.Header().Get("X-ECHO-REQID"); rid == "" {
		t.Fatalf("expected X-ECHO-REQID header to be set")
	}
}
