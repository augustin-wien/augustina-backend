package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
	"gopkg.in/guregu/null.v4"
)

type posOrderEntry struct {
	Item     int `json:"item"`
	Quantity int `json:"quantity"`
}

type posOrderRequest struct {
	Entries    []posOrderEntry `json:"entries"`
	UseBalance bool            `json:"useBalance"`
}

// CreatePOSOrder godoc
//
//	@Summary		Create a cash POS order for a vendor
//	@Description	Admin-only endpoint. Creates a cash sale for a vendor at the backoffice.
//	@Tags			pos
//	@Accept			json
//	@Produce		json
//	@Param			licenseID	path		string			true	"Vendor license ID"
//	@Param			body		body		posOrderRequest	true	"POS order"
//	@Success		200			{object}	map[string]bool
//	@Router			/vendors/{licenseID}/pos-order/ [post]
func CreatePOSOrder(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")

	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if !settings.POSEnabled {
		utils.ErrorJSON(w, errors.New("POS is disabled"), http.StatusForbidden)
		return
	}

	var req posOrderRequest
	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if len(req.Entries) == 0 {
		utils.ErrorJSON(w, errors.New("entries must not be empty"), http.StatusBadRequest)
		return
	}

	// Validate items and compute total
	total := 0
	for _, e := range req.Entries {
		if e.Quantity <= 0 {
			utils.ErrorJSON(w, errors.New("quantity must be greater than 0"), http.StatusBadRequest)
			return
		}
		item, err := database.Db.GetItem(e.Item)
		if err != nil {
			utils.ErrorJSON(w, errors.New("item not found"), http.StatusBadRequest)
			return
		}
		if item.Type != "normal_item" && item.Type != "issue" {
			utils.ErrorJSON(w, errors.New("only normal_item and issue types are allowed in POS"), http.StatusBadRequest)
			return
		}
		if item.IsPDFItem {
			utils.ErrorJSON(w, errors.New("PDF/digital items are not allowed in POS"), http.StatusBadRequest)
			return
		}
		if item.LicenseItem.Valid {
			utils.ErrorJSON(w, errors.New("items requiring a digital license are not allowed in POS"), http.StatusBadRequest)
			return
		}
		total += item.Price * e.Quantity
	}

	// Resolve accounts
	vendor, err := database.Db.GetVendorByLicenseID(licenseID)
	if err != nil {
		utils.ErrorJSON(w, errors.New("vendor not found"), http.StatusBadRequest)
		return
	}
	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	backofficeAccount, err := database.Db.GetAccountByType("Backoffice")
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	orgaAccount, err := database.Db.GetAccountByType("Orga")
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	cashAccount, err := database.Db.GetAccountByType("Cash")
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// Determine split: use all vendor balance (capped at total) or none
	balancePortion := 0
	if req.UseBalance && vendorAccount.Balance > 0 {
		balancePortion = vendorAccount.Balance
		if balancePortion > total {
			balancePortion = total
		}
	}
	cashPortion := total - balancePortion

	authorizedBy := r.Header.Get("X-Auth-User-Name")

	// Build payment list
	var payments []database.Payment

	// Balance chain: Vendor → Orga → Backoffice
	if balancePortion > 0 {
		payments = append(payments,
			database.Payment{
				Sender:       vendorAccount.ID,
				Receiver:     orgaAccount.ID,
				Amount:       balancePortion,
				AuthorizedBy: authorizedBy,
				IsSale:       false,
				IsPOS:        true,
				Quantity:     1,
				Price:        balancePortion,
			},
			database.Payment{
				Sender:       orgaAccount.ID,
				Receiver:     backofficeAccount.ID,
				Amount:       balancePortion,
				AuthorizedBy: authorizedBy,
				IsSale:       false,
				IsPOS:        true,
				Quantity:     1,
				Price:        balancePortion,
			},
		)
	}

	// Cash: Cash → Backoffice
	if cashPortion > 0 {
		payments = append(payments, database.Payment{
			Sender:       cashAccount.ID,
			Receiver:     backofficeAccount.ID,
			Amount:       cashPortion,
			AuthorizedBy: authorizedBy,
			IsSale:       false,
			IsPOS:        true,
			Quantity:     1,
			Price:        cashPortion,
		})
	}

	// Per-item bookkeeping records (is_sale=true, is_pos=true).
	// createPaymentTx skips account-balance updates for these, so they don't
	// affect the vendor's credit balance — the actual money flow is captured
	// by the balance-chain and cash payments above.
	for _, e := range req.Entries {
		item, _ := database.Db.GetItem(e.Item)
		payments = append(payments, database.Payment{
			Sender:       vendorAccount.ID,
			Receiver:     backofficeAccount.ID,
			Amount:       item.Price * e.Quantity,
			AuthorizedBy: authorizedBy,
			IsSale:       true,
			IsPOS:        true,
			Item:         null.IntFrom(int64(e.Item)),
			Quantity:     e.Quantity,
			Price:        item.Price,
		})
	}

	if err := database.Db.CreatePayments(payments); err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ListAllPOSOrders godoc
//
//	@Summary		List all POS orders across all vendors
//	@Tags			pos
//	@Produce		json
//	@Param			from	query		string	false	"Start date (RFC3339)"
//	@Param			to		query		string	false	"End date (RFC3339)"
//	@Success		200		{array}		database.POSOrder
//	@Router			/pos-orders/ [get]
func ListAllPOSOrders(w http.ResponseWriter, r *http.Request) {
	var minDate, maxDate time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			minDate = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			maxDate = t
		}
	}
	orders, err := database.Db.ListAllPOSOrders(minDate, maxDate)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, nil, orders)
}

// ListPOSOrdersForVendor godoc
//
//	@Summary		List recent POS orders for a vendor
//	@Tags			pos
//	@Produce		json
//	@Param			licenseID	path		string	true	"Vendor license ID"
//	@Success		200			{array}		database.POSOrder
//	@Router			/vendors/{licenseID}/pos-orders/ [get]
func ListPOSOrdersForVendor(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")
	orders, err := database.Db.ListPOSOrdersForVendor(licenseID, 20)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, nil, orders)
}
