package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/paymentprovider"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// sweepViva is a VivaWallet test server that confirms the registered transactions
type sweepViva struct {
	mu           sync.Mutex
	transactions map[string]paymentprovider.TransactionVerificationResponse
}

func (v *sweepViva) register(transactionID string, orderCode int64, amount float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.transactions[transactionID] = paymentprovider.TransactionVerificationResponse{
		OrderCode: paymentprovider.VivaOrderCode(orderCode), Amount: amount, StatusID: "F",
		MerchantTrns: "Ein gutes Leben für alle!",
	}
}

func startSweepViva(t *testing.T) *sweepViva {
	t.Helper()
	v := &sweepViva{transactions: map[string]paymentprovider.TransactionVerificationResponse{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/connect/token":
			_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":3600}`))
		case r.URL.Path == "/checkout/v2/orders":
			_, _ = fmt.Fprintf(w, `{"orderCode":%d}`, 8200000000+sweepSeq.Add(1))
		case strings.HasPrefix(r.URL.Path, "/checkout/v2/transactions/"):
			v.mu.Lock()
			tx, ok := v.transactions[strings.TrimPrefix(r.URL.Path, "/checkout/v2/transactions/")]
			v.mu.Unlock()
			if !ok {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(tx)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	orig := config.Config
	origProd := database.Db.IsProduction
	t.Cleanup(func() {
		config.Config = orig
		database.Db.IsProduction = origProd
	})
	config.Config.VivaWalletAPIURL = srv.URL
	config.Config.VivaWalletAccountsURL = srv.URL
	config.Config.VivaWalletSmartCheckoutURL = srv.URL + "/checkout?s="
	config.Config.VivaWalletSmartCheckoutClientID = "id"
	config.Config.VivaWalletSmartCheckoutClientKey = "key"
	config.Config.VivaWalletSourceCode = "1234"
	config.Config.DEBUG_payments = false
	config.Config.Development = true
	database.Db.IsProduction = true
	return v
}

// sweepUnverifiedOrder creates an unverified order and returns its id and order code
func sweepUnverifiedOrder(t *testing.T, v database.Vendor, a database.Account, item int) (int, int64) {
	t.Helper()
	buyer, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	code := 8300000000 + sweepSeq.Add(1)
	id, err := database.Db.CreateOrder(database.Order{
		Vendor: v.ID, OrderCode: null.StringFrom(fmt.Sprintf("%d", code)),
		CustomerEmail: null.StringFrom(fmt.Sprintf("buyer-%d@example.com", code)),
		Entries:       []database.OrderEntry{{Item: item, Quantity: 1, Sender: buyer, Receiver: a.ID, IsSale: true}},
	})
	require.NoError(t, err)
	return id, code
}

func multipartBody(fields map[string]string) (string, string) {
	var b strings.Builder
	for k, v := range fields {
		fmt.Fprintf(&b, "--b\r\nContent-Disposition: form-data; name=%q\r\n\r\n%s\r\n", k, v)
	}
	b.WriteString("--b--\r\n")
	return b.String(), "multipart/form-data; boundary=b"
}

func TestHandlerFlowsSurviveEachDatabaseFailure(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	origSend := mailer.Send
	t.Cleanup(func() { mailer.Send = origSend })
	mailer.Send = func(*mailer.EmailRequest) (bool, error) { return true, nil }

	viva := startSweepViva(t)
	_, err := database.Db.GetItemByName(config.Config.TransactionCostsName)
	require.NoError(t, err, "the test database has the transaction costs item")
	require.NoError(t, database.Db.CreateOrUpdateMailTemplate("PDFLicenceItemTemplate.html", "Dein PDF", `<a href="{{.URL}}">PDF</a>`))

	cases := []struct {
		name  string
		want  int
		setup func(t *testing.T, run int) sweepRequest
	}{
		{"create order", 200, func(t *testing.T, _ int) sweepRequest {
			v, _ := sweepVendor(t)
			return sweepRequest{method: "POST", path: "/api/orders/", body: fmt.Sprintf(`{"Entries":[{"Item":%d,"Quantity":1}],"VendorLicenseID":%q,"CustomerEmail":"c@example.com"}`, sweepItem(t, "normal_item"), v.LicenseID.String)}
		}},
		{"success webhook", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			_, code := sweepUnverifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			tx := fmt.Sprintf("sweep-tx-%d", code)
			viva.register(tx, code, 3)
			var req paymentprovider.TransactionSuccessRequest
			req.EventData.OrderCode = paymentprovider.VivaOrderCode(code)
			req.EventData.TransactionID = tx
			req.EventData.Amount = 3
			req.EventData.StatusID = "F"
			body, _ := json.Marshal(req)
			return sweepRequest{method: "POST", path: "/api/webhooks/vivawallet/success/", body: string(body)}
		}},
		{"price webhook", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			orderID, code := sweepUnverifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			require.NoError(t, database.Db.VerifyOrderAndCreatePayments(orderID, 0))
			tx := fmt.Sprintf("sweep-price-%d", code)
			viva.register(tx, code, 3)
			var req paymentprovider.TransactionPriceRequest
			req.EventData.OrderCode = paymentprovider.VivaOrderCode(code)
			req.EventData.TransactionID = tx
			req.EventData.TotalCommission = 0.15
			body, _ := json.Marshal(req)
			return sweepRequest{method: "POST", path: "/api/webhooks/vivawallet/price/", body: string(body)}
		}},
		{"admin verify by code", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			orderID, code := sweepUnverifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			tx := fmt.Sprintf("sweep-admin-%d", code)
			require.NoError(t, database.Db.SetOrderTransactionID(orderID, tx))
			viva.register(tx, code, 3)
			return sweepRequest{method: "GET", path: fmt.Sprintf("/api/orders/unverified/code/%d/verify/", code)}
		}},
		{"verify payment order", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			orderID, code := sweepUnverifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			tx := fmt.Sprintf("sweep-verify-%d", code)
			require.NoError(t, database.Db.SetOrderTransactionID(orderID, tx))
			require.NoError(t, database.Db.VerifyOrderAndCreatePayments(orderID, 0))
			viva.register(tx, code, 3)
			return sweepRequest{method: "GET", path: fmt.Sprintf("/api/orders/verify/?s=%d&t=%s", code, tx)}
		}},
		{"resend order mail", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			pdfID, err := database.Db.CreatePDF(database.PDF{Path: fmt.Sprintf("/tmp/sweep-%d.pdf", sweepSeq.Add(1)), Timestamp: time.Now()})
			require.NoError(t, err)
			license, err := database.Db.CreateItem(database.Item{Name: fmt.Sprintf("Sweep license %d", sweepSeq.Add(1)), Description: "Test item", Price: 100, Type: "license_item", IsLicenseItem: true})
			require.NoError(t, err)
			pdfItem, err := database.Db.CreateItem(database.Item{Name: fmt.Sprintf("Sweep pdf %d", sweepSeq.Add(1)), Description: "Test item", Price: 300, Type: "issue", IsPDFItem: true, PDF: null.IntFrom(pdfID), LicenseItem: null.IntFrom(int64(license))})
			require.NoError(t, err)
			orderID := sweepVerifiedOrder(t, v, a, pdfItem)
			return sweepRequest{method: "POST", path: fmt.Sprintf("/api/orders/%d/resend-mail/", orderID)}
		}},
		{"create online issue", 200, func(t *testing.T, run int) sweepRequest {
			body, ct := multipartBody(map[string]string{"Name": fmt.Sprintf("Sweep issue %d", sweepSeq.Add(1)), "Description": "Issue", "Price": "300", "Type": "online_issue", "LicenseGroup": "sweep_group", "license_cost": "100"})
			return sweepRequest{method: "POST", path: "/api/items/", body: body, contentType: ct}
		}},
		{"update item", 200, func(t *testing.T, _ int) sweepRequest {
			id := sweepItem(t, "normal_item")
			body, ct := multipartBody(map[string]string{"Name": fmt.Sprintf("Sweep renamed %d", sweepSeq.Add(1)), "Description": "Renamed", "Price": "400", "Type": "normal_item"})
			return sweepRequest{method: "PUT", path: fmt.Sprintf("/api/items/%d/", id), body: body, contentType: ct}
		}},
		{"customer get update delete", 204, func(t *testing.T, _ int) sweepRequest {
			c, err := database.Db.CreateCustomer(&database.Customer{Email: fmt.Sprintf("sweep-cust-%d@example.com", sweepSeq.Add(1))})
			require.NoError(t, err)
			return sweepRequest{method: "DELETE", path: fmt.Sprintf("/api/customers/%d/", c.ID)}
		}},
		{"update customer", 200, func(t *testing.T, _ int) sweepRequest {
			c, err := database.Db.CreateCustomer(&database.Customer{Email: fmt.Sprintf("sweep-upd-%d@example.com", sweepSeq.Add(1))})
			require.NoError(t, err)
			return sweepRequest{method: "PUT", path: fmt.Sprintf("/api/customers/%d/", c.ID), body: fmt.Sprintf(`{"email":%q,"firstname":"Neu","newsletter":true}`, c.Email)}
		}},
		{"update abonement", 200, func(t *testing.T, _ int) sweepRequest {
			c, err := database.Db.CreateCustomer(&database.Customer{Email: fmt.Sprintf("sweep-aboupd-%d@example.com", sweepSeq.Add(1))})
			require.NoError(t, err)
			abo, err := database.Db.CreateAbonement(&database.Abonement{CustomerID: c.ID, FromDate: time.Now(), ToDate: time.Now().AddDate(1, 0, 0), Status: "active"})
			require.NoError(t, err)
			return sweepRequest{method: "PUT", path: fmt.Sprintf("/api/abonements/%d/", abo.ID), body: fmt.Sprintf(`{"customer_id":%d,"from_date":"2026-01-01","to_date":"2027-01-01","status":"active"}`, c.ID)}
		}},
		{"campaign update", 200, func(t *testing.T, _ int) sweepRequest {
			item := sweepItem(t, "normal_item")
			body := fmt.Sprintf(`{"name":"sweep-c-%d","item_id":%d,"title":"t","enabled":true}`, sweepSeq.Add(1), item)
			res := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/campaigns/", strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+adminUserToken.AccessToken)
			r.ServeHTTP(res, req)
			var c Campaign
			require.NoError(t, json.Unmarshal(res.Body.Bytes(), &c))
			return sweepRequest{method: "PUT", path: fmt.Sprintf("/api/campaigns/%d/", c.ID), body: body}
		}},
		{"unverified orders", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			sweepUnverifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			return sweepRequest{method: "GET", path: "/api/orders/unverified/"}
		}},
		{"verified orders", 200, func(t *testing.T, _ int) sweepRequest {
			return sweepRequest{method: "GET", path: "/api/orders/verified/"}
		}},
		{"payments", 200, func(t *testing.T, _ int) sweepRequest {
			return sweepRequest{method: "GET", path: "/api/payments/"}
		}},
		{"mail tasks", 200, func(t *testing.T, _ int) sweepRequest {
			return sweepRequest{method: "GET", path: "/api/mail-tasks/"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Development mode simulates the webhook of a new order in the background
			config.Config.Development = tc.name != "create order"
			sweepHandler(t, tc.want, tc.setup)
		})
	}
}
