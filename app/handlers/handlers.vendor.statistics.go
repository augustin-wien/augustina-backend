package handlers

import (
	"net/http"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
)

// VendorUsageStatistics is the response to ListVendorUsageStatistics
type VendorUsageStatistics struct {
	From             time.Time
	To               time.Time
	TotalVendors     int
	UsedVendors      int
	UnusedVendors    int
	UsedPercentage   float64
	UnusedPercentage float64
}

// ListVendorUsageStatistics godoc
//
//	@Summary		Calculate vendor usage percentages
//	@Description	Filter by date and return how many vendors used the tool in the period.
//	@Tags			Vendors
//	@Accept			json
//	@Produce		json
//	@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//	@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//	@Success		200	{object}	VendorUsageStatistics
//	@Security		KeycloakAuth
//	@Router			/vendors/statistics/ [get]
func ListVendorUsageStatistics(w http.ResponseWriter, r *http.Request) {
	var err error

	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")

	var minDate, maxDate time.Time
	if minDateRaw != "" {
		minDate, err = time.Parse(time.RFC3339, minDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}
	if maxDateRaw != "" {
		maxDate, err = time.Parse(time.RFC3339, maxDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}

	vendors, err := database.Db.ListVendors()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	vendorAccountByAccountID := make(map[int]int)
	for _, vendor := range vendors {
		account, accErr := database.Db.GetAccountByVendorID(vendor.ID)
		if accErr != nil {
			utils.ErrorJSON(w, accErr, http.StatusBadRequest)
			return
		}
		vendorAccountByAccountID[account.ID] = vendor.ID
	}

	payments, err := database.Db.ListPayments(minDate, maxDate, "", false, false, false)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	usedVendorIDs := make(map[int]struct{})
	for _, payment := range payments {
		if vendorID, ok := vendorAccountByAccountID[payment.Sender]; ok {
			usedVendorIDs[vendorID] = struct{}{}
		}
		if vendorID, ok := vendorAccountByAccountID[payment.Receiver]; ok {
			usedVendorIDs[vendorID] = struct{}{}
		}
	}

	totalVendors := len(vendors)
	usedVendors := len(usedVendorIDs)
	unusedVendors := totalVendors - usedVendors

	statistics := VendorUsageStatistics{
		From:          minDate,
		To:            maxDate,
		TotalVendors:  totalVendors,
		UsedVendors:   usedVendors,
		UnusedVendors: unusedVendors,
	}

	if totalVendors > 0 {
		statistics.UsedPercentage = (float64(usedVendors) / float64(totalVendors)) * 100
		statistics.UnusedPercentage = (float64(unusedVendors) / float64(totalVendors)) * 100
	}

	respond(w, nil, statistics)
}
