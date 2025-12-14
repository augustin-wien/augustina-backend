package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/integrations"
	"github.com/go-chi/chi/v5"
)

type resendResponse struct {
	Status  string `json:"status"`
	OrderID int    `json:"order_id"`
	Message string `json:"message"`
	SentAt  string `json:"sent_at"`
}

// ResendFlourWebhook triggers resending the Flour webhook for a given order ID.
func ResendFlourWebhook(w http.ResponseWriter, r *http.Request) {
	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil || orderID <= 0 {
		http.Error(w, "invalid orderID", http.StatusBadRequest)
		return
	}

	order, err := database.Db.GetOrderByID(orderID)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	vendor, err := database.Db.GetVendor(order.Vendor)
	if err != nil {
		http.Error(w, "vendor not found", http.StatusNotFound)
		return
	}

	// Compute total sum consistent with integrations.SendPaymentToFlour logic
	totalSum := 0
	for _, item := range order.Entries {
		itemPrice := item.Price
		if item.Receiver != vendor.ID {
			itemPrice = -itemPrice
		}
		totalSum += itemPrice * item.Quantity
	}

	// Send webhook
	if err := integrations.SendPaymentToFlour(order.ID, order.Timestamp, order.Entries, vendor, totalSum); err != nil {
		http.Error(w, "failed to send webhook", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resendResponse{
		Status:  "ok",
		OrderID: order.ID,
		Message: "webhook resent",
		SentAt:  time.Now().UTC().Format(time.RFC3339),
	})
}
