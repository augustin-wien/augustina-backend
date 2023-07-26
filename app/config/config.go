package config

import "os"

// Config is the global configuration variable
var Config = config{
	Version: "0.0.1",
	Port: getEnv("PORT", "3000"),
	Development: (getEnv("DEVELOPMENT", "false") == "true"),
}
type config struct{
	Version string
	Port string
	Development bool
}

// Local copy of utils.GetEnv to avoid circular dependency
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}
