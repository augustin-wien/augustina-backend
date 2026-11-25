package paymentprovider

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/stretchr/testify/require"
)

// fakeVivaWallet points the VivaWallet configuration at a test server that
// answers token, order and transaction requests with the given handlers.
func fakeVivaWallet(t *testing.T, token, orders, transaction http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/connect/token":
			token(w, r)
		case r.URL.Path == "/checkout/v2/orders":
			orders(w, r)
		case strings.HasPrefix(r.URL.Path, "/checkout/v2/transactions/"):
			transaction(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	orig := config.Config
	t.Cleanup(func() { config.Config = orig })
	config.Config.VivaWalletAPIURL = server.URL
	config.Config.VivaWalletAccountsURL = server.URL
	config.Config.VivaWalletSmartCheckoutClientID = "id"
	config.Config.VivaWalletSmartCheckoutClientKey = "key"
	config.Config.VivaWalletSourceCode = "1234"
}

func respond(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestAuthenticateToVivaWallet(t *testing.T) {
	var gotAuth string
	fakeVivaWallet(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"access_token":"tok"}`))
	}, nil, nil)

	token, err := AuthenticateToVivaWallet()
	require.NoError(t, err)
	require.Equal(t, "tok", token)
	require.Equal(t, "Basic aWQ6a2V5", gotAuth)

	config.Config.VivaWalletSmartCheckoutClientKey = ""
	_, err = AuthenticateToVivaWallet()
	require.Error(t, err)

	config.Config.VivaWalletAccountsURL = "not a url"
	_, err = AuthenticateToVivaWallet()
	require.Error(t, err)

	config.Config.VivaWalletAccountsURL = ""
	_, err = AuthenticateToVivaWallet()
	require.Error(t, err)
}

func TestAuthenticateToVivaWalletInvalidResponse(t *testing.T) {
	fakeVivaWallet(t, respond(200, "not json"), nil, nil)
	_, err := AuthenticateToVivaWallet()
	require.Error(t, err)

	config.Config.VivaWalletAccountsURL = "http://127.0.0.1:1"
	_, err = AuthenticateToVivaWallet()
	require.Error(t, err)
}

func TestCreatePaymentOrderResponses(t *testing.T) {
	order := database.Order{}
	cases := []struct {
		name    string
		handler http.HandlerFunc
		code    string
		wantErr bool
	}{
		{"success", respond(200, `{"orderCode": 1234567890123456}`), "1234567890123456", false},
		{"error status", respond(500, `{"message":"down"}`), "", true},
		{"invalid json", respond(200, `nope`), "", true},
		{"empty order code", respond(200, `{"orderCode": 0}`), "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeVivaWallet(t, nil, tc.handler, nil)
			code, err := CreatePaymentOrder("tok", order, "lic-1")
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.code, code)
		})
	}

	t.Run("missing configuration", func(t *testing.T) {
		fakeVivaWallet(t, nil, respond(200, `{"orderCode": 1}`), nil)
		config.Config.VivaWalletSourceCode = ""
		_, err := CreatePaymentOrder("tok", order, "lic-1")
		require.Error(t, err)
		config.Config.VivaWalletAPIURL = ""
		_, err = CreatePaymentOrder("tok", order, "lic-1")
		require.Error(t, err)
		config.Config.VivaWalletAPIURL = "http://127.0.0.1:1"
		config.Config.VivaWalletSourceCode = "1234"
		_, err = CreatePaymentOrder("tok", order, "lic-1")
		require.Error(t, err)
	})
}

func TestVerifyTransactionIDResponses(t *testing.T) {
	token := respond(200, `{"access_token":"tok"}`)
	cases := []struct {
		name    string
		token   http.HandlerFunc
		tx      http.HandlerFunc
		wantErr error
	}{
		{"finished", token, respond(200, `{"orderCode": 1234567890, "statusId": "F", "amount": 5}`), nil},
		{"waiting", token, respond(200, `{"orderCode": 1234567890, "statusId": "MW"}`), nil},
		{"not successful", token, respond(200, `{"orderCode": 1234567890, "statusId": "E"}`), ErrTransactionNotSuccessful},
		{"error status", token, respond(404, `missing`), errors.New("any")},
		{"invalid json", token, respond(200, `nope`), errors.New("any")},
		{"auth fails", respond(200, `nope`), respond(200, `{}`), errors.New("any")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeVivaWallet(t, tc.token, nil, tc.tx)
			res, err := VerifyTransactionID("0b6d4b6c-1234-4c0e-9d2a-abcdefabcdef", false)
			switch {
			case tc.wantErr == nil:
				require.NoError(t, err)
				require.Equal(t, "1234567890", res.OrderCode.String())
			case errors.Is(tc.wantErr, ErrTransactionNotSuccessful):
				require.ErrorIs(t, err, ErrTransactionNotSuccessful)
			default:
				require.Error(t, err)
			}
		})
	}

	t.Run("no api url", func(t *testing.T) {
		fakeVivaWallet(t, token, nil, nil)
		config.Config.VivaWalletAPIURL = ""
		_, err := VerifyTransactionID("0b6d4b6c-1234-4c0e-9d2a-abcdefabcdef", false)
		require.Error(t, err)
	})
}

func TestPaymentFailureAndPaypalStubs(t *testing.T) {
	require.NoError(t, HandlePaymentFailureResponse(TransactionSuccessRequest{}))

	orig := config.Config.VivaWalletTransactionTypeIDPaypal
	t.Cleanup(func() { config.Config.VivaWalletTransactionTypeIDPaypal = orig })
	config.Config.VivaWalletTransactionTypeIDPaypal = 0
	require.Error(t, CreatePaypalTransactionCosts(TransactionSuccessRequest{}, database.Order{}))
	config.Config.VivaWalletTransactionTypeIDPaypal = 48
	require.NoError(t, CreatePaypalTransactionCosts(TransactionSuccessRequest{}, database.Order{}))
}
