package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
)

var (
	orderCodePattern = regexp.MustCompile(`^\d{1,20}$`)
	eventIDPattern   = regexp.MustCompile(`^\d{1,9}$`)
)

type reportPaymentFailureRequest struct {
	OrderCode string `json:"orderCode"`
	EventID   string `json:"eventId"`
}

// ReportPaymentFailure godoc
//
//	 	@Summary 		Report a failed payment
//		@Description	Stores the EventId VivaWallet appended to the failure redirect on the unverified order
//		@Tags			Orders
//		@Accept			json
//		@Param			body body reportPaymentFailureRequest true "Order code and EventId"
//		@Success		204
//		@Router			/orders/failure/ [post]
func ReportPaymentFailure(w http.ResponseWriter, r *http.Request) {
	var req reportPaymentFailureRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req); err != nil {
		utils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}
	if !orderCodePattern.MatchString(req.OrderCode) || !eventIDPattern.MatchString(req.EventID) {
		utils.ErrorJSON(w, errors.New("invalid orderCode or eventId"), http.StatusBadRequest)
		return
	}
	eventID, _ := strconv.Atoi(req.EventID)

	updated, err := database.Db.SetOrderPaymentFailure(req.OrderCode, eventID)
	if err != nil {
		utils.ErrorJSON(w, errors.New("storing the payment failure failed"), http.StatusInternalServerError)
		return
	}
	if !updated {
		utils.ErrorJSON(w, errors.New("no unverified order with this order code"), http.StatusNotFound)
		return
	}

	log.Infow("ReportPaymentFailure: payment failed at VivaWallet", "order_code", req.OrderCode, "event_id", eventID)
	w.WriteHeader(http.StatusNoContent)
}
