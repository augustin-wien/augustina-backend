package utils

import (
	"augustin/config"
	"os"

	"go.uber.org/zap"
)

// GetLogger initializes a logger
func GetLogger() *zap.SugaredLogger {
	if config.Config.Development {
		return zap.Must(zap.NewDevelopment()).Sugar()
	}
	return zap.Must(zap.NewProduction()).Sugar()
}

// GetEnv returns the value of an environment variable or a default value if the environment variable is not set
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}
