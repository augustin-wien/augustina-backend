package utils

import (
	"os"

	"go.uber.org/zap"
)


func InitLog() *zap.SugaredLogger {
	return zap.Must(zap.NewDevelopment()).Sugar()
}

var log = InitLog()

// GetEnv returns the value of an environment variable or a default value if the environment variable is not set
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}
