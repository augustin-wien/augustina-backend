package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
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

func TestRequestLogger_LogsRequestId(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Restore stdout at the end
	defer func() {
		os.Stdout = oldStdout
	}()

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(RequestLogger)

	router.Get("/log-test", func(rw http.ResponseWriter, req *http.Request) {
		// This log should contain the request_id
		utils.GetLogger().Info("test log inside handler")
		rw.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/log-test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Close the write end of the pipe to read from it
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check if the output contains "request_id" and the log message
	if !strings.Contains(output, "test log inside handler") {
		t.Error("expected log message not found in output")
	}
	if !strings.Contains(output, "request_id") {
		t.Error("expected request_id field not found in log output")
	}
}

func TestRequestLogger_LogsRequestId_WithDatabaseCall(t *testing.T) {
	// Setup environment and DB
	wd, _ := os.Getwd()
	// Assuming we are in app/middlewares, go up to app/
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	config.InitConfig()
	// Initialize test database
	if err := database.Db.InitEmptyTestDb(); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(RequestLogger)

	router.Get("/db-test", func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		// Perform a database operation
		// We use the Settings table as it is initialized in InitEmptyTestDb
		count, err := database.Db.EntClient.Settings.Query().Count(ctx)
		if err != nil {
			utils.GetLogger().Errorw("db error", "error", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		utils.GetLogger().Infow("db call success", "count", count)
		rw.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/db-test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "db call success") {
		t.Error("expected log message not found in output")
		t.Logf("Output: %s", output)
	}

	// Check if driver.Query log contains request_id
	lines := strings.Split(output, "\n")
	foundQueryLogWithReqID := false
	for _, line := range lines {
		if strings.Contains(line, "driver.Query") && strings.Contains(line, "request_id") {
			foundQueryLogWithReqID = true
			break
		}
	}

	if !foundQueryLogWithReqID {
		t.Error("expected driver.Query log to contain request_id")
		t.Logf("Output: %s", output)
	}
}
