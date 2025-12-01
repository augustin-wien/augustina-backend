package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIPBlocker(t *testing.T) {
	// Helper to reset the global blocker state
	resetBlocker := func() {
		GlobalBlocker.mu.Lock()
		GlobalBlocker.blockedIPs = make(map[string]time.Time)
		GlobalBlocker.mu.Unlock()
	}

	t.Run("BlockSuspiciousRequests blocks suspicious path", func(t *testing.T) {
		resetBlocker()

		handler := BlockSuspiciousRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/.env", nil)
		req.Header.Set("X-Real-Ip", "1.2.3.4")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, GlobalBlocker.IsBlocked("1.2.3.4"))
	})

	t.Run("BlockSuspiciousRequests allows valid path", func(t *testing.T) {
		resetBlocker()

		handler := BlockSuspiciousRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/valid-path", nil)
		req.Header.Set("X-Real-Ip", "1.2.3.5")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, GlobalBlocker.IsBlocked("1.2.3.5"))
	})

	t.Run("BlockSuspiciousRequests blocks exact match path", func(t *testing.T) {
		resetBlocker()

		handler := BlockSuspiciousRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/api", nil)
		req.Header.Set("X-Real-Ip", "1.2.3.6")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, GlobalBlocker.IsBlocked("1.2.3.6"))
	})

	t.Run("BlockSuspiciousRequests allows path starting with exact match prefix", func(t *testing.T) {
		resetBlocker()

		handler := BlockSuspiciousRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/api/valid", nil)
		req.Header.Set("X-Real-Ip", "1.2.3.7")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, GlobalBlocker.IsBlocked("1.2.3.7"))
	})

	t.Run("FilterBlockedIPs blocks previously blocked IP", func(t *testing.T) {
		resetBlocker()
		GlobalBlocker.BlockIP("1.2.3.8", 1*time.Hour)

		handler := FilterBlockedIPs(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/valid-path", nil)
		req.Header.Set("X-Real-Ip", "1.2.3.8")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("FilterBlockedIPs allows unblocked IP", func(t *testing.T) {
		resetBlocker()

		handler := FilterBlockedIPs(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/valid-path", nil)
		req.Header.Set("X-Real-Ip", "1.2.3.9")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Integration: Suspicious request blocks subsequent valid requests", func(t *testing.T) {
		resetBlocker()

		// Chain middlewares: FilterBlockedIPs -> BlockSuspiciousRequests -> Handler
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		chain := FilterBlockedIPs(BlockSuspiciousRequests(finalHandler))

		ip := "1.2.3.10"

		// 1. Suspicious request
		req1 := httptest.NewRequest("GET", "/.env", nil)
		req1.Header.Set("X-Real-Ip", ip)
		w1 := httptest.NewRecorder()
		chain.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusForbidden, w1.Code)

		// 2. Valid request from same IP should now be blocked
		req2 := httptest.NewRequest("GET", "/valid", nil)
		req2.Header.Set("X-Real-Ip", ip)
		w2 := httptest.NewRecorder()
		chain.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusForbidden, w2.Code)
	})

	t.Run("BlockBadUserAgents blocks bad user agent", func(t *testing.T) {
		resetBlocker()

		handler := BlockBadUserAgents(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/valid-path", nil)
		req.Header.Set("User-Agent", "curl/7.64.1")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("BlockBadUserAgents allows good user agent", func(t *testing.T) {
		resetBlocker()

		handler := BlockBadUserAgents(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/valid-path", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("BlockBadUserAgents allows health check paths with bad user agent", func(t *testing.T) {
		resetBlocker()

		handler := BlockBadUserAgents(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/healthz", nil)
		req.Header.Set("User-Agent", "curl/7.64.1")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		req2 := httptest.NewRequest("GET", "/readyz", nil)
		req2.Header.Set("User-Agent", "curl/7.64.1")
		w2 := httptest.NewRecorder()

		handler.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
	})

	t.Run("BlockFakeBrowsers blocks fake browser", func(t *testing.T) {
		resetBlocker()

		handler := BlockFakeBrowsers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/valid-path", nil)
		// Claim to be Mozilla but no Accept-Language
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("BlockFakeBrowsers allows real browser", func(t *testing.T) {
		resetBlocker()

		handler := BlockFakeBrowsers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/valid-path", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("IsBlocked returns false for expired block", func(t *testing.T) {
		resetBlocker()
		// Block for a negative duration to simulate expiration
		GlobalBlocker.BlockIP("1.2.3.11", -1*time.Hour)

		assert.False(t, GlobalBlocker.IsBlocked("1.2.3.11"))
	})

	t.Run("IsBlocked returns false for unknown IP", func(t *testing.T) {
		resetBlocker()
		assert.False(t, GlobalBlocker.IsBlocked("9.9.9.9"))
	})

	t.Run("BlockMaliciousPatterns blocks SQLi", func(t *testing.T) {
		resetBlocker()

		handler := BlockMaliciousPatterns(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/search?q=union+select", nil)
		req.Header.Set("X-Real-Ip", "1.2.3.12")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, GlobalBlocker.IsBlocked("1.2.3.12"))
	})

	t.Run("BlockMaliciousPatterns blocks XSS", func(t *testing.T) {
		resetBlocker()

		handler := BlockMaliciousPatterns(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/search?q=<script>alert(1)</script>", nil)
		req.Header.Set("X-Real-Ip", "1.2.3.13")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, GlobalBlocker.IsBlocked("1.2.3.13"))
	})

	t.Run("AddStrike blocks IP after threshold", func(t *testing.T) {
		resetBlocker()
		ip := "1.2.3.14"

		// 9 strikes - not blocked
		for i := 0; i < 9; i++ {
			GlobalBlocker.AddStrike(ip)
		}
		assert.False(t, GlobalBlocker.IsBlocked(ip))

		// 10th strike - blocked
		GlobalBlocker.AddStrike(ip)
		assert.True(t, GlobalBlocker.IsBlocked(ip))
	})

	t.Run("Integration: Bad User-Agent accumulates strikes", func(t *testing.T) {
		resetBlocker()
		ip := "1.2.3.15"

		handler := BlockBadUserAgents(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// 9 requests - should be forbidden but not blocked globally yet
		for i := 0; i < 9; i++ {
			req := httptest.NewRequest("GET", "/valid", nil)
			req.Header.Set("X-Real-Ip", ip)
			req.Header.Set("User-Agent", "curl/7.64.1")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		}
		assert.False(t, GlobalBlocker.IsBlocked(ip))

		// 10th request - should trigger block
		req := httptest.NewRequest("GET", "/valid", nil)
		req.Header.Set("X-Real-Ip", ip)
		req.Header.Set("User-Agent", "curl/7.64.1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, GlobalBlocker.IsBlocked(ip))
	})
}
