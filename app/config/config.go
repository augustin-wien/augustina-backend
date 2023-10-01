package config

import (
	"os"
	"strconv"
)

type config struct {
	Version                           string
	Port                              string
	CreateDemoData                    bool
	TransactionCostsName              string
	VivaWalletVerificationKey         string
	VivaWalletApiURL                  string
	VivaWalletAccountsURL             string
	VivaWalletSmartCheckoutURL        string
	VivaWalletSmartCheckoutClientID   string
	VivaWalletSmartCheckoutClientKey  string
	PaypalFixCosts                    float64
	PaypalPercentageCosts             float64
	VivaWalletTransactionTypeIDPaypal int
}

// Config is the global configuration variable
var Config = config{
	Version:                           "0.0.1",
	Port:                              getEnv("PORT", "3000"),
	CreateDemoData:                    (getEnv("CREATE_DEMO_DATA", "false") == "true"),
	TransactionCostsName:              getEnv("TRANSACTION_COSTS_NAME", "transactionCosts"),
	VivaWalletVerificationKey:         getEnv("VIVA_WALLET_VERIFICATION_KEY", ""),
	VivaWalletApiURL:                  getEnv("VIVA_WALLET_API_URL", ""),
	VivaWalletAccountsURL:             getEnv("VIVA_WALLET_ACCOUNTS_URL", ""),
	VivaWalletSmartCheckoutURL:        getEnv("VIVA_WALLET_SMART_CHECKOUT_URL", ""),
	VivaWalletSmartCheckoutClientID:   getEnv("VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID", ""),
	VivaWalletSmartCheckoutClientKey:  getEnv("VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY", ""),
	VivaWalletTransactionTypeIDPaypal: getEnvInt("VIVA_WALLET_TRANSACTION_TYPE_ID_PAYPAL", 0),
	PaypalFixCosts:                    getEnvFloat("PAYPAL_FIX_COSTS", 0.00),
	PaypalPercentageCosts:             getEnvFloat("PAYPAL_PERCENTAGE_COSTS", 0.00),
}

// Local copy of utils.GetEnv to avoid circular dependency
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// getEnvFloat returns the value of the environment variable key as a float64 or the fallback value if not set
func getEnvFloat(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		float64_value, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fallback
		}
		return float64_value
	}
	return fallback
}

// getEnvInt returns the value of the environment variable key as an int or the fallback value if not set
func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		int_value, err := strconv.Atoi(value)
		if err != nil {
			return fallback
		}
		return int_value
	}
	return fallback
}
