package config

import "os"

type config struct {
	Version                     string
	Port                        string
	CreateDemoData              bool
	PaymentServiceProvider      string
	VivaWalletSourceCode        string
	VivaWalletClientCredentials string
	VivaWalletVerificationKey   string
	VivaWalletApiURL            string
	VivaWalletAccountsURL       string
}

// Config is the global configuration variable
var Config = config{
	Version:                     "0.0.1",
	Port:                        getEnv("PORT", "3000"),
	CreateDemoData:              (getEnv("CREATE_DEMO_DATA", "false") == "true"),
	PaymentServiceProvider:      getEnv("PAYMENT_SERVICE_PROVIDER", ""),
	VivaWalletSourceCode:        getEnv("VIVA_WALLET_SOURCE_CODE", ""),
	VivaWalletClientCredentials: getEnv("VIVA_WALLET_CLIENT_CREDENTIALS", ""),
	VivaWalletVerificationKey:   getEnv("VIVA_WALLET_VERIFICATION_KEY", ""),
	VivaWalletApiURL:            getEnv("VIVA_WALLET_API_URL", ""),
	VivaWalletAccountsURL:       getEnv("VIVA_WALLET_ACCOUNTS_URL", ""),
}

// Local copy of utils.GetEnv to avoid circular dependency
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
