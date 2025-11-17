package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var version = "1.0.21-0f82a52"

type config struct {
	Version                           string
	Port                              string
	CreateDemoData                    bool
	PaypalFixCosts                    float64
	PaypalPercentageCosts             float64
	TransactionCostsName              string
	DonationName                      string
	IntervalToDeletePDFsInWeeks       int
	VivaWalletVerificationKey         string
	VivaWalletAPIURL                  string
	VivaWalletAccountsURL             string
	VivaWalletSmartCheckoutURL        string
	VivaWalletSmartCheckoutClientID   string
	VivaWalletSmartCheckoutClientKey  string
	VivaWalletSourceCode              string
	VivaWalletTransactionTypeIDPaypal int
	KeycloakHostname                  string
	KeycloakRealm                     string
	KeycloakClientID                  string
	KeycloakClientSecret              string
	KeycloakVendorGroup               string
	KeycloakCustomerGroup             string
	KeycloakBackofficeGroup           string
	SendCustomerEmail                 bool
	OnlinePaperUrl                    string
	FrontendURL                       string
	Development                       bool
	SMTPServer                        string
	SMTPPort                          string
	SMTPUsername                      string
	SMTPPassword                      string
	SMTPSenderAddress                 string
	SMTPSsl                           bool
	SMTPInsecureSkipVerify            bool
	SentryDSN                         string
	FlourWebhookURL                   string
	DEBUG_payments                    bool // For debugging purposes, not used in production
}

// Config is the global configuration variable
var Config config

func InitConfig() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	_ = godotenv.Load(pwd + "/.env")

	Config = config{
		Version:                           version,
		Port:                              getEnv("PORT", "3000"),
		CreateDemoData:                    (getEnv("CREATE_DEMO_DATA", "false") == "true"),
		PaypalFixCosts:                    getEnvFloat("PAYPAL_FIX_COSTS", 0.00),
		PaypalPercentageCosts:             getEnvFloat("PAYPAL_PERCENTAGE_COSTS", 0.00),
		DonationName:                      getEnv("DONATION_NAME", "donation"),
		TransactionCostsName:              getEnv("TRANSACTION_COSTS_NAME", "transactionCosts"),
		IntervalToDeletePDFsInWeeks:       getEnvInt("INTERVAL_TO_DELETE_PDFS_IN_WEEKS", 0),
		VivaWalletVerificationKey:         getEnv("VIVA_WALLET_VERIFICATION_KEY", ""),
		VivaWalletAPIURL:                  getEnv("VIVA_WALLET_API_URL", ""),
		VivaWalletAccountsURL:             getEnv("VIVA_WALLET_ACCOUNTS_URL", ""),
		VivaWalletSmartCheckoutURL:        getEnv("VIVA_WALLET_SMART_CHECKOUT_URL", ""),
		VivaWalletSmartCheckoutClientID:   getEnv("VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID", ""),
		VivaWalletSmartCheckoutClientKey:  getEnv("VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY", ""),
		VivaWalletSourceCode:              getEnv("VIVA_WALLET_SOURCE_CODE", ""),
		VivaWalletTransactionTypeIDPaypal: getEnvInt("VIVA_WALLET_TRANSACTION_TYPE_ID_PAYPAL", 0),
		KeycloakVendorGroup:               getEnv("KEYCLOAK_VENDOR_GROUP", "vendors"),
		KeycloakCustomerGroup:             getEnv("KEYCLOAK_CUSTOMER_GROUP", "customer"),
		KeycloakBackofficeGroup:           getEnv("KEYCLOAK_BACKOFFICE_GROUP", "backoffice"),
		KeycloakHostname:                  getEnv("KEYCLOAK_HOST", ""),
		KeycloakRealm:                     getEnv("KEYCLOAK_REALM", ""),
		KeycloakClientID:                  getEnv("KEYCLOAK_CLIENT_ID", ""),
		KeycloakClientSecret:              getEnv("KEYCLOAK_CLIENT_SECRET", ""),
		SendCustomerEmail:                 (getEnv("SEND_CUSTOMER_EMAIL", "false") == "true"),
		OnlinePaperUrl:                    getEnv("ONLINE_PAPER_URL", ""),
		Development:                       (getEnv("DEVELOPMENT", "false") == "true"),
		SMTPServer:                        getEnv("SMTP_SERVER", ""),
		SMTPPort:                          getEnv("SMTP_PORT", ""),
		SMTPUsername:                      getEnv("SMTP_USERNAME", ""),
		SMTPPassword:                      getEnv("SMTP_PASSWORD", ""),
		SMTPSenderAddress:                 getEnv("SMTP_SENDER_ADDRESS", ""),
		SMTPSsl:                           (getEnv("SMTP_SSL", "false") == "true"),
		SMTPInsecureSkipVerify:            (getEnv("SMTP_INSECURE_SKIP_VERIFY", "false") == "true"),
		FrontendURL:                       getEnv("FRONTEND_URL", ""),
		SentryDSN:                         getEnv("SENTRY_DSN", ""),
		FlourWebhookURL:                   getEnv("FLOUR_WEBHOOK_URL", ""),
		DEBUG_payments:                    (getEnv("DEBUG_payments", "false") == "true"),
	}
	return nil
}

// Validate checks required configuration values and returns an error if any
// required value is missing or malformed.
func (c config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT must be set")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("FRONTEND_URL must be set")
	}
	return nil
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
		float64Value, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fallback
		}
		return float64Value
	}
	return fallback
}

// getEnvInt returns the value of the environment variable key as an int or the fallback value if not set
func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return fallback
		}
		return intValue
	}
	return fallback
}
