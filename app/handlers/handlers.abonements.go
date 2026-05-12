package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

type ActiveAbonementWithCustomer struct {
	Abonement database.Abonement `json:"abonement"`
	Customer  database.Customer  `json:"customer"`
}

// CreateAbonement godoc
//
//	@Summary		Create a new abonement
//	@Tags			Abonements
//	@Accept			json
//	@Produce		json
//	@Param			abonement body database.Abonement true "Abonement data"
//	@Success		201	{object}	database.Abonement
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/abonements [post]
func CreateAbonement(w http.ResponseWriter, r *http.Request) {
	var abonement database.Abonement

	err := json.NewDecoder(r.Body).Decode(&abonement)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	createdAbonement, err := database.Db.CreateAbonement(&abonement)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Fetch customer and item details for email
	customer, err := database.Db.GetCustomerByID(abonement.CustomerID)
	if err == nil && customer.Email != "" {
		item, err := database.Db.GetItem(abonement.ItemID)
		if err == nil {
			// Send abonement confirmation email
			templateData := map[string]interface{}{
				"CustomerName": customer.FirstName + " " + customer.LastName,
				"ItemName":     item.Name,
				"FromDate":     createdAbonement.FromDate.Format("2006-01-02"),
				"ToDate":       createdAbonement.ToDate.Format("2006-01-02"),
				"Status":       createdAbonement.Status,
			}
			mailReq, err := database.BuildEmailRequestFromTemplate("abonementConfirmation", []string{customer.Email}, templateData)
			if err == nil {
				_, _ = mailer.Send(mailReq) // Send async, don't block on email errors
			}
		}
	}

	err = utils.WriteJSON(w, http.StatusCreated, createdAbonement)
	if err != nil {
		log.Error("CreateAbonement", err)
	}
}

// GetAbonement godoc
//
//	@Summary		Get an abonement by ID
//	@Tags			Abonements
//	@Produce		json
//	@Param			id path int true "Abonement ID"
//	@Success		200	{object}	database.Abonement
//	@Failure		404	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/abonements/{id} [get]
func GetAbonement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	abonement, err := database.Db.GetAbonementByID(id)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, abonement)
	if err != nil {
		log.Error("GetAbonement", err)
	}
}

// ListAbonements godoc
//
//	@Summary		List all abonements
//	@Tags			Abonements
//	@Produce		json
//	@Success		200	{array}	database.Abonement
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/abonements [get]
func ListAbonements(w http.ResponseWriter, r *http.Request) {
	abonements, err := database.Db.ListAbonements()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, abonements)
	if err != nil {
		log.Error("ListAbonements", err)
	}
}

// ListMyAbonements godoc
//
//	@Summary		List abonements for the authenticated customer
//	@Tags			Abonements
//	@Produce		json
//	@Success		200	{array}	database.Abonement
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers/me/abonements [get]
func ListMyAbonements(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-Auth-User")
	if userID == "" {
		utils.ErrorJSON(w, errors.New("Unauthorized"), http.StatusUnauthorized)
		return
	}

	customer, err := database.Db.GetCustomerByKeycloakID(userID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}

	abonements, err := database.Db.ListAbonementsByCustomer(customer.ID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, abonements)
	if err != nil {
		log.Error("ListMyAbonements", err)
	}
}

// ListAbonementsByCustomer godoc
//
//	@Summary		List all abonements for a customer
//	@Tags			Abonements
//	@Produce		json
//	@Param			customer_id path int true "Customer ID"
//	@Success		200	{array}	database.Abonement
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers/{customer_id}/abonements [get]
func ListAbonementsByCustomer(w http.ResponseWriter, r *http.Request) {
	customerID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	abonements, err := database.Db.ListAbonementsByCustomer(customerID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, abonements)
	if err != nil {
		log.Error("ListAbonementsByCustomer", err)
	}
}

// UpdateAbonement godoc
//
//	@Summary		Update an abonement
//	@Tags			Abonements
//	@Accept			json
//	@Produce		json
//	@Param			id path int true "Abonement ID"
//	@Param			abonement body database.Abonement true "Abonement data"
//	@Success		200	{object}	database.Abonement
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/abonements/{id} [put]
func UpdateAbonement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	var abonement database.Abonement
	err = json.NewDecoder(r.Body).Decode(&abonement)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	abonement.ID = id
	updatedAbonement, err := database.Db.UpdateAbonement(&abonement)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, updatedAbonement)
	if err != nil {
		log.Error("UpdateAbonement", err)
	}
}

// DeleteAbonement godoc
//
//	@Summary		Delete an abonement
//	@Tags			Abonements
//	@Param			id path int true "Abonement ID"
//	@Success		204
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/abonements/{id} [delete]
func DeleteAbonement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.DeleteAbonement(id)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetActiveAbonementsByDate godoc
//
//	@Summary		Get active abonements for a given date
//	@Tags			Abonements
//	@Produce		json
//	@Param			date query string true "Date in ISO 8601 format (YYYY-MM-DD)"
//	@Success		200	{array}	database.Abonement
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/abonements/by-date [get]
func GetActiveAbonementsByDate(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		utils.ErrorJSON(w, errors.New("date parameter is required"), http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	abonements, err := database.Db.GetActiveAbonementsByDate(date)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, abonements)
	if err != nil {
		log.Error("GetActiveAbonementsByDate", err)
	}
}

// ListActiveAbonementsWithCustomers godoc
//
//	@Summary		List active abonements with customer details (admin)
//	@Tags			Abonements
//	@Produce		json
//	@Success		200	{array}	handlers.ActiveAbonementWithCustomer
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/abonements/active [get]
func ListActiveAbonementsWithCustomers(w http.ResponseWriter, r *http.Request) {
	abonements, err := database.Db.GetActiveAbonementsByDate(time.Now())
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	result := make([]ActiveAbonementWithCustomer, 0, len(abonements))
	for _, abonement := range abonements {
		customer, customerErr := database.Db.GetCustomerByID(abonement.CustomerID)
		if customerErr != nil {
			log.Error("ListActiveAbonementsWithCustomers: failed to load customer", customerErr)
			continue
		}
		result = append(result, ActiveAbonementWithCustomer{
			Abonement: *abonement,
			Customer:  *customer,
		})
	}

	err = utils.WriteJSON(w, http.StatusOK, result)
	if err != nil {
		log.Error("ListActiveAbonementsWithCustomers", err)
	}
}
