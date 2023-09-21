package config

import "os"

type config struct {
	Version                          string
	Port                             string
	CreateDemoData                   bool
	PaymentServiceProvider           string
	VivaWalletSourceCode             string
	VivaWalletVerificationKey        string
	VivaWalletApiURL                 string
	VivaWalletAccountsURL            string
	VivaWalletSmartCheckoutURL       string
	VivaWalletSmartCheckoutClientID  string
	VivaWalletSmartCheckoutClientKey string
}

// Config is the global configuration variable
var Config = config{
	Version:                          "0.0.1",
	Port:                             getEnv("PORT", "3000"),
	CreateDemoData:                   (getEnv("CREATE_DEMO_DATA", "false") == "true"),
	PaymentServiceProvider:           getEnv("PAYMENT_SERVICE_PROVIDER", ""),
	VivaWalletSourceCode:             getEnv("VIVA_WALLET_SOURCE_CODE", ""),
	VivaWalletVerificationKey:        getEnv("VIVA_WALLET_VERIFICATION_KEY", ""),
	VivaWalletApiURL:                 getEnv("VIVA_WALLET_API_URL", ""),
	VivaWalletAccountsURL:            getEnv("VIVA_WALLET_ACCOUNTS_URL", ""),
	VivaWalletSmartCheckoutURL:       getEnv("VIVA_WALLET_SMART_CHECKOUT_URL", ""),
	VivaWalletSmartCheckoutClientID:  getEnv("VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID", ""),
	VivaWalletSmartCheckoutClientKey: getEnv("VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY", ""),
}

// Local copy of utils.GetEnv to avoid circular dependency
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
