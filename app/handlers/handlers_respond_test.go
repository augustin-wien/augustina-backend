package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// failingWriter fails every write with err
type failingWriter struct {
	http.ResponseWriter
	err error
}

func (f failingWriter) Write([]byte) (int, error) { return 0, f.err }

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestRespond(t *testing.T) {
	rr := httptest.NewRecorder()
	respond(rr, errors.New("bad"), nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	// Write errors are only logged
	require.NotPanics(t, func() {
		respond(failingWriter{httptest.NewRecorder(), syscall.EPIPE}, nil, map[string]string{"a": "b"})
		respond(failingWriter{httptest.NewRecorder(), errors.New("disk full")}, nil, map[string]string{"a": "b"})
	})
}

func TestIsClientGone(t *testing.T) {
	require.True(t, isClientGone(syscall.EPIPE))
	require.True(t, isClientGone(syscall.ECONNRESET))
	require.True(t, isClientGone(timeoutError{}))
	require.False(t, isClientGone(errors.New("other")))
}
