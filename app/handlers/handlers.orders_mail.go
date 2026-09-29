package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

type resendOrderMailResponse struct {
	OrderID int
	Sent    int
}

// ResendOrderMail godoc
//
//	@Summary		Resend the mails of a digital sale
//	@Description	Sends the online paper and PDF download mails of a verified order to its customer again. PDF download links are valid for another six weeks.
//	@Tags			Orders
//	@Produce		json
//	@Param			orderID	path	int	true	"Order ID"
//	@Success		200	{object}	resendOrderMailResponse
//	@Security		KeycloakAuth
//	@Router			/orders/{orderID}/resend-mail/ [post]
//
// ResendOrderMail API Handler
func ResendOrderMail(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.Atoi(chi.URLParam(r, "orderID"))
	if err != nil || orderID <= 0 {
		utils.ErrorJSON(w, errors.New("invalid orderID"), http.StatusBadRequest)
		return
	}

	sent, err := database.Db.ResendOrderMails(orderID)
	switch {
	case ent.IsNotFound(err):
		utils.ErrorJSON(w, errors.New("order not found"), http.StatusNotFound)
		return
	case errors.Is(err, database.ErrOrderNotVerified),
		errors.Is(err, database.ErrOrderHasNoCustomer),
		errors.Is(err, database.ErrOrderHasNoDigitalItems):
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	case err != nil:
		utils.ErrorJSON(w, err, http.StatusBadGateway)
		return
	}

	respond(w, nil, resendOrderMailResponse{OrderID: orderID, Sent: sent})
}
