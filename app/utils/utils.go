package utils

import (
	"augustin/config"
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
