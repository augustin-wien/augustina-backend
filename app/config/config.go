package config

import "os"

type config struct {
	Version                     string
	Port                        string
	Development                 bool
	PaymentServiceProvider      string
	VivaWalletSourceCode        string
	VivaWalletClientCredentials string
	VivaWalletVerificationKey   string
	VivaWalletURL               string
}

// Config is the global configuration variable
var Config = config{
	Version:                     "0.0.1",
	Port:                        getEnv("PORT", "3000"),
	Development:                 (getEnv("DEVELOPMENT", "false") == "true"),
	PaymentServiceProvider:      getEnv("PAYMENT_SERVICE_PROVIDER", ""),
	VivaWalletSourceCode:        getEnv("VIVA_WALLET_SOURCE_CODE", ""),
	VivaWalletClientCredentials: getEnv("VIVA_WALLET_CLIENT_CREDENTIALS", ""),
	VivaWalletVerificationKey:   getEnv("VIVA_WALLET_VERIFICATION_KEY", ""),
	VivaWalletURL:               getEnv("VIVA_WALLET_URL", ""),
}

// Local copy of utils.GetEnv to avoid circular dependency
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}
