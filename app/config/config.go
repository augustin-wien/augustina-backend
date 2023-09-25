package config

import (
	"os"
	"strconv"
)

type config struct {
	Version                          string
	Port                             string
	CreateDemoData                   bool
	OrgaCoversTransactionCosts       bool
	TransactionCostsName             string
	PaymentServiceProvider           string
	VivaWalletSourceCode             string
	VivaWalletVerificationKey        string
	VivaWalletApiURL                 string
	VivaWalletAccountsURL            string
	VivaWalletSmartCheckoutURL       string
	VivaWalletSmartCheckoutClientID  string
	VivaWalletSmartCheckoutClientKey string
	PaypalFixCosts                   float64
	PaypalPercentageCosts            float64
}

// Config is the global configuration variable
var Config = config{
	Version:                          "0.0.1",
	Port:                             getEnv("PORT", "3000"),
	CreateDemoData:                   (getEnv("CREATE_DEMO_DATA", "false") == "true"),
	OrgaCoversTransactionCosts:       (getEnv("ORGA_COVERS_TRANSACTION_COSTS", "false") == "true"),
	TransactionCostsName:             getEnv("TRANSACTION_COSTS_NAME", "transactionCosts"),
	PaymentServiceProvider:           getEnv("PAYMENT_SERVICE_PROVIDER", ""),
	VivaWalletSourceCode:             getEnv("VIVA_WALLET_SOURCE_CODE", ""),
	VivaWalletVerificationKey:        getEnv("VIVA_WALLET_VERIFICATION_KEY", ""),
	VivaWalletApiURL:                 getEnv("VIVA_WALLET_API_URL", ""),
	VivaWalletAccountsURL:            getEnv("VIVA_WALLET_ACCOUNTS_URL", ""),
	VivaWalletSmartCheckoutURL:       getEnv("VIVA_WALLET_SMART_CHECKOUT_URL", ""),
	VivaWalletSmartCheckoutClientID:  getEnv("VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID", ""),
	VivaWalletSmartCheckoutClientKey: getEnv("VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY", ""),
	PaypalFixCosts:                   getEnvFloat("PAYPAL_FIX_COSTS", 0.00),
	PaypalPercentageCosts:            getEnvFloat("PAYPAL_PERCENTAGE_COSTS", 0.00),
}

// Local copy of utils.GetEnv to avoid circular dependency
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

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
