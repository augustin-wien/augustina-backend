package middlewares

import (
    "net/http"
    "time"

    "github.com/augustin-wien/augustina-backend/utils"
    chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// statusRecorder wraps http.ResponseWriter to capture status code and size
type statusRecorder struct {
    http.ResponseWriter
    status int
    size   int
}

func (r *statusRecorder) WriteHeader(status int) {
    r.status = status
    r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
    if r.status == 0 {
        r.status = http.StatusOK
    }
    n, err := r.ResponseWriter.Write(b)
    r.size += n
    return n, err
}

// RequestLogger logs incoming HTTP requests with request-id and timing information.
func RequestLogger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        reqID := chimiddleware.GetReqID(r.Context())
        start := time.Now()
        sr := &statusRecorder{ResponseWriter: w}
        next.ServeHTTP(sr, r)
        duration := time.Since(start)

        log := utils.GetLogger()
        log.Infow("http_request",
            "id", reqID,
            "method", r.Method,
            "path", r.URL.Path,
            "remote", r.RemoteAddr,
            "status", sr.status,
            "bytes", sr.size,
            "duration_ms", duration.Milliseconds(),
        )
    })
}
