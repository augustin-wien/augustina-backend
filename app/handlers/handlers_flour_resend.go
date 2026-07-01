package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
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

// ResendOdooWebhook triggers resending the Odoo webhook for a given order ID.
func ResendOdooWebhook(w http.ResponseWriter, r *http.Request) {
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

	totalSum := 0
	for _, item := range order.Entries {
		itemPrice := item.Price
		if item.Receiver <= 5 {
			itemPrice = -itemPrice
		}
		totalSum += itemPrice * item.Quantity
	}

	var errs []error

	if config.Config.FlourWebhookURL != "" {
		if err := integrations.SendPaymentToFlour(order.ID, order.Timestamp, order.Entries, vendor, totalSum); err != nil {
			errs = append(errs, err)
		}
	}

	if config.Config.OdooWebhookURL != "" {
		if err := integrations.SendPaymentToOdoo(order.ID, order.Timestamp, order.Entries, vendor, totalSum); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		http.Error(w, errors.Join(errs...).Error(), http.StatusBadGateway)
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
