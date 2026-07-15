package utils

import (
	"context"
	crand "crypto/rand"
	"math/big"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/notifications"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GetLogger initializes a logger
func GetLogger() *zap.SugaredLogger {
	var consoleEncoder zapcore.Encoder
	stdout := zapcore.AddSync(os.Stdout)

	if config.Config.CreateDemoData {
		developmentCfg := zap.NewDevelopmentEncoderConfig()
		developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoder = zapcore.NewConsoleEncoder(developmentCfg)
	} else {
		productionCfg := zap.NewProductionEncoderConfig()
		productionCfg.TimeKey = "timestamp"
		productionCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoder = zapcore.NewConsoleEncoder(productionCfg)
	}

	notificationCfg := zap.NewDevelopmentEncoderConfig()
	notificationEncoder := zapcore.NewConsoleEncoder(notificationCfg)

	level := zap.NewAtomicLevelAt(zap.DebugLevel)

	consoleCore := zapcore.NewCore(consoleEncoder, stdout, level)
	notificationCore := zapcore.NewCore(notificationEncoder, notifications.NotificationsClient, zap.NewAtomicLevelAt(zap.ErrorLevel))

	core := zapcore.NewTee(
		&reqIDCore{core: consoleCore},
		&reqIDCore{core: notificationCore},
	)

	return zap.New(core).Sugar()
}

// context key type for storing request-scoped logger
type ctxKeyLogger struct{}

// WithLogger returns a new context that carries the provided logger.
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger{}, logger)
}

// LoggerFromContext extracts a request-scoped logger from the context.
// If no logger is present it falls back to the global logger from GetLogger().
func LoggerFromContext(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		return GetLogger()
	}
	if l, ok := ctx.Value(ctxKeyLogger{}).(*zap.SugaredLogger); ok && l != nil {
		return l
	}
	return GetLogger()
}

// --- goroutine-local request id support ---

var (
	gidMu  sync.RWMutex
	gidMap = make(map[int64]string)
)

func getGID() int64 {
	var buf [128]byte
	n := runtime.Stack(buf[:], false)
	if n <= 0 {
		return 0
	}
	// Stack header: "goroutine 12345 [running]:\n"
	fields := strings.Fields(string(buf[:n]))
	if len(fields) < 2 {
		return 0
	}
	gid, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return gid
}

func getRequestIDForGID(gid int64) string {
	gidMu.RLock()
	defer gidMu.RUnlock()
	return gidMap[gid]
}

// SetRequestID sets the request id for the current goroutine. Call this at the
// start of request handling so logs emitted with GetLogger() include the id.
func SetRequestID(id string) {
	gid := getGID()
	if gid == 0 {
		return
	}
	gidMu.Lock()
	gidMap[gid] = id
	gidMu.Unlock()
}

// ClearRequestID clears the request id for the current goroutine.
func ClearRequestID() {
	gid := getGID()
	if gid == 0 {
		return
	}
	gidMu.Lock()
	delete(gidMap, gid)
	gidMu.Unlock()
}

func currentRequestID() string {
	gid := getGID()
	if gid == 0 {
		return ""
	}
	return getRequestIDForGID(gid)
}

// reqIDCore wraps another zapcore.Core and injects the goroutine-local
// request_id field into logged entries when present.
type reqIDCore struct {
	core zapcore.Core
}

func (r *reqIDCore) Enabled(l zapcore.Level) bool { return r.core.Enabled(l) }
func (r *reqIDCore) With(fields []zapcore.Field) zapcore.Core {
	return &reqIDCore{core: r.core.With(fields)}
}
func (r *reqIDCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if r.Enabled(ent.Level) {
		return ce.AddCore(ent, r)
	}
	return ce
}
func (r *reqIDCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if id := currentRequestID(); id != "" {
		fields = append(fields, zap.String("request_id", id))
	}
	return r.core.Write(ent, fields)
}
func (r *reqIDCore) Sync() error { return r.core.Sync() }

// GetEnv returns the value of an env var, null value if var is not set yet or a default value if the environment variable is not set
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// RandomString returns a random string of length n
func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[RandomInt(0, len(letters))]
	}
	return string(b)
}

// RandomInt returns a random int between min and max
func RandomInt(min, max int) int {
	if max <= min {
		return min
	}
	span := int64(max - min)
	n, err := crand.Int(crand.Reader, big.NewInt(span))
	if err != nil {
		return min
	}
	return min + int(n.Int64())
}

// GetUnixTime returns the current unix time in seconds
func GetUnixTime() int64 {
	return time.Now().Unix()
}

// hostOnly returns the host portion of an address, stripping any port. It falls back to the
// input unchanged when there is no port to split.
func hostOnly(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// ReadUserIP returns the user's IP address.
//
// The X-Real-Ip and X-Forwarded-For headers are client-supplied and therefore spoofable.
// When config.TrustedProxies is empty, the legacy behavior is preserved (headers trusted
// unconditionally). When it is populated, those headers are only honored for requests that
// actually originate from a trusted proxy; otherwise the direct RemoteAddr is used. This
// prevents attackers from poisoning the IP blocklist or evading IP-based blocks/rate limits
// by forging these headers.
func ReadUserIP(r *http.Request) string {
	trustedProxies := config.Config.TrustedProxies

	// Legacy behavior: no configured proxies means trust the forwarding headers as-is.
	if len(trustedProxies) == 0 {
		if ip := r.Header.Get("X-Real-Ip"); ip != "" {
			return ip
		}
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			return ip
		}
		return r.RemoteAddr
	}

	remoteHost := hostOnly(r.RemoteAddr)

	trusted := make(map[string]struct{}, len(trustedProxies))
	for _, p := range trustedProxies {
		trusted[p] = struct{}{}
	}

	// Only honor forwarding headers when the immediate peer is a trusted proxy.
	if _, ok := trusted[remoteHost]; !ok {
		return remoteHost
	}

	// Walk X-Forwarded-For from right to left and return the first address that is not
	// itself a trusted proxy - that is the closest non-proxy client the chain saw.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if ip == "" {
				continue
			}
			if _, ok := trusted[ip]; !ok {
				return ip
			}
		}
	}

	// Fall back to X-Real-Ip, then to the trusted proxy address itself.
	if ip := strings.TrimSpace(r.Header.Get("X-Real-Ip")); ip != "" {
		return ip
	}
	return remoteHost
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func ToLower(s string) string {
	return strings.ToLower(s)
}

func GenerateRandomNumber() int {
	// Generate a random number between 100000 and 999999
	n, err := crand.Int(crand.Reader, big.NewInt(900000))
	if err != nil {
		// fallback to timestamp-based value (not crypto secure)
		return int((time.Now().UnixNano()/1e6)%900000) + 100000
	}
	return int(n.Int64()) + 100000
}
