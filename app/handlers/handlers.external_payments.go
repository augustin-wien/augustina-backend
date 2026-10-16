package handlers

import (
	"net/http"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
)

// ExternalPayment is the backoffice view of a VivaWallet payment from another sales channel.
// ent's own struct tags use omitempty, which would drop zero amounts and fees.
type ExternalPayment struct {
	ID            int       `json:"id"`
	TransactionID string    `json:"transaction_id"`
	OrderCode     string    `json:"order_code"`
	Amount        int       `json:"amount"`
	Fee           int       `json:"fee"`
	StatusID      string    `json:"status_id"`
	SourceCode    string    `json:"source_code"`
	SourceName    string    `json:"source_name"`
	TerminalID    int       `json:"terminal_id"`
	MerchantTrns  string    `json:"merchant_trns"`
	PaidAt        time.Time `json:"paid_at"`
}

// ListExternalPayments godoc
//
//	@Summary		List payments from other sales channels
//	@Description	VivaWallet payments not made through this backend, such as donations at a card terminal. Recorded while donation tracking is enabled.
//	@Tags			Payments
//	@Produce		json
//	@Param			from	query		string	false	"Start of the period (RFC3339)"
//	@Param			to		query		string	false	"End of the period, exclusive (RFC3339)"
//	@Success		200		{array}		ExternalPayment
//	@Security		KeycloakAuth
//	@Router			/external-payments/ [get]
func ListExternalPayments(w http.ResponseWriter, r *http.Request) {
	var from, to time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		from = t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		to = t
	}

	payments, err := database.Db.ListExternalPayments(from, to)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	res := make([]ExternalPayment, 0, len(payments))
	for _, p := range payments {
		res = append(res, ExternalPayment{
			ID:            p.ID,
			TransactionID: p.TransactionID,
			OrderCode:     p.OrderCode,
			Amount:        p.Amount,
			Fee:           p.Fee,
			StatusID:      p.StatusID,
			SourceCode:    p.SourceCode,
			SourceName:    p.SourceName,
			TerminalID:    p.TerminalID,
			MerchantTrns:  p.MerchantTrns,
			PaidAt:        p.PaidAt,
		})
	}
	respond(w, nil, res)
}
