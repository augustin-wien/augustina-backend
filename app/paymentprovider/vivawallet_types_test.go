package paymentprovider

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// VivaWallet sends orderCode as a JSON number. On 2025-11-28 the Go type was changed to
// string and every transaction lookup failed to unmarshal, leaving paid orders unverified.
// These payloads mirror the shape VivaWallet actually sends.

func TestTransactionVerificationResponse_NumericOrderCode(t *testing.T) {
	body := `{"email":"a@example.com","amount":5.0,"orderCode":2295536933107359,"statusId":"F",` +
		`"fullName":"A B","insDate":"2025-11-28T18:36:40.123+01:00","transactionTypeId":5}`

	var resp TransactionVerificationResponse
	require.NoError(t, json.Unmarshal([]byte(body), &resp))
	require.Equal(t, VivaOrderCode(2295536933107359), resp.OrderCode)
	require.Equal(t, "2295536933107359", resp.OrderCode.String())
}

func TestVivaOrderCode_AcceptsQuotedString(t *testing.T) {
	var resp TransactionVerificationResponse
	require.NoError(t, json.Unmarshal([]byte(`{"orderCode":"2295536933107359"}`), &resp))
	require.Equal(t, "2295536933107359", resp.OrderCode.String())
}

func TestVivaOrderCode_NullIsZero(t *testing.T) {
	var resp PaymentOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"orderCode":null}`), &resp))
	require.Zero(t, resp.OrderCode)
}

func TestVivaOrderCode_RejectsGarbage(t *testing.T) {
	var resp PaymentOrderResponse
	require.Error(t, json.Unmarshal([]byte(`{"orderCode":"abc"}`), &resp))
}

func TestVivaOrderCode_MarshalsAsNumber(t *testing.T) {
	b, err := json.Marshal(PaymentOrderResponse{OrderCode: 2295536933107359})
	require.NoError(t, err)
	require.JSONEq(t, `{"orderCode":2295536933107359}`, string(b))
}

// The success webhook goes through utils.ReadJSON, which decodes into a map with UseNumber
// and re-marshals before unmarshalling into the struct - the order code must survive that
// without losing precision.
func TestTransactionSuccessRequest_NumericOrderCodeThroughReadJSON(t *testing.T) {
	body := `{"EventTypeId":1796,"EventData":{"OrderCode":2295536933107359,` +
		`"TransactionId":"8f620cda-dd80-4f45-9212-65080d881b12","StatusId":"F","Amount":5.0}}`
	r := httptest.NewRequest("POST", "/api/webhooks/vivawallet/success/", strings.NewReader(body))
	w := httptest.NewRecorder()

	var req TransactionSuccessRequest
	require.NoError(t, utils.ReadJSON(w, r, &req))
	require.Equal(t, "2295536933107359", req.EventData.OrderCode.String())
	require.Equal(t, "8f620cda-dd80-4f45-9212-65080d881b12", req.EventData.TransactionID)
}

func TestTransactionPriceRequest_NumericOrderCode(t *testing.T) {
	var req TransactionPriceRequest
	require.NoError(t, json.Unmarshal([]byte(`{"EventData":{"OrderCode":2295536933107359,"TransactionId":"x"}}`), &req))
	require.Equal(t, "2295536933107359", req.EventData.OrderCode.String())
}
