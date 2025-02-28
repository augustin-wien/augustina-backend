package utils

import (
	"math/rand"
	"net/http"
	"os"
	"strings"
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
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdout, level),
		zapcore.NewCore(notificationEncoder, notifications.NotificationsClient, level),
	)

	return zap.New(core).Sugar()
}

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
	return min + rand.Intn(max-min)
}

// GetUnixTime returns the current unix time in seconds
func GetUnixTime() int64 {
	return time.Now().Unix()
}

// ReadUserIP returns the user's IP address
func ReadUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
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
