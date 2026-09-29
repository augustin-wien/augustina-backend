package handlers

import (
	"net/http"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
)

// ListVerifiedOrders godoc
//
//	@Summary		List verified orders
//	@Description	List verified orders (sales) with all their entries, newest first. The date range filters by verification time.
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//	@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//	@Success		200	{array}	database.Order
//	@Security		KeycloakAuth
//	@Router			/orders/verified/ [get]
//
// ListVerifiedOrders API Handler fetching data from database
func ListVerifiedOrders(w http.ResponseWriter, r *http.Request) {
	var minDate, maxDate time.Time
	var err error
	if v := r.URL.Query().Get("from"); v != "" {
		minDate, err = time.Parse(time.RFC3339, v)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		maxDate, err = time.Parse(time.RFC3339, v)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}
	orders, err := database.Db.GetVerifiedOrders(minDate, maxDate)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, nil, orders)
}
