package utils

import (
	"augustin/config"
	"encoding/base64"
	"math/rand"
	"os"

	"go.uber.org/zap"
)

// GetLogger initializes a logger
func GetLogger() *zap.SugaredLogger {
	if config.Config.CreateDemoData {
		return zap.Must(zap.NewDevelopment()).Sugar()
	}
	return zap.Must(zap.NewProduction()).Sugar()
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
	const letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[RandomInt(0, len(letters))]
	}
	return string(b)
}

func RandomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}