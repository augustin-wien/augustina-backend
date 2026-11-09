package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestReportPaymentFailure(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	tests := []struct {
		name     string
		body     any
		verified bool
		status   int
		stored   bool
	}{
		{name: "stores the EventId", body: reportPaymentFailureRequest{OrderCode: "5555555555", EventID: "12005"}, status: http.StatusNoContent, stored: true},
		{name: "ignores a verified order", body: reportPaymentFailureRequest{OrderCode: "5555555555", EventID: "12005"}, verified: true, status: http.StatusNotFound},
		{name: "unknown order code", body: reportPaymentFailureRequest{OrderCode: "4444444444", EventID: "12005"}, status: http.StatusNotFound},
		{name: "non numeric EventId", body: reportPaymentFailureRequest{OrderCode: "5555555555", EventID: "12005x"}, status: http.StatusBadRequest},
		{name: "missing order code", body: reportPaymentFailureRequest{EventID: "12005"}, status: http.StatusBadRequest},
		{name: "invalid body", body: "not an object", status: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, database.Db.InitEmptyTestDb())

			createTestVendor(t, "testpaymentfailure")
			vendor, err := database.Db.GetVendorByLicenseID("testpaymentfailure")
			require.NoError(t, err)
			orderID, err := database.Db.CreateOrder(database.Order{
				OrderCode: null.StringFrom("5555555555"),
				Vendor:    vendor.ID,
				Timestamp: time.Now(),
			})
			require.NoError(t, err)
			if tt.verified {
				require.NoError(t, database.Db.VerifyOrderAndCreatePayments(orderID, 0))
			}

			utils.TestRequest(t, r, "POST", "/api/orders/failure/", tt.body, tt.status)

			order, err := database.Db.GetOrderByOrderCode("5555555555")
			require.NoError(t, err)
			if tt.stored {
				require.Equal(t, null.IntFrom(12005), order.PaymentEventID)
				require.True(t, order.PaymentFailedAt.Valid)
			} else {
				require.False(t, order.PaymentEventID.Valid)
				require.False(t, order.PaymentFailedAt.Valid)
			}
		})
	}
}
