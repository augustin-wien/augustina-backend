package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

// CreateCustomer godoc
//
//	@Summary		Create a new customer
//	@Tags			Customers
//	@Accept			json
//	@Produce		json
//	@Param			customer body database.Customer true "Customer data"
//	@Success		201	{object}	database.Customer
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers [post]
func CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var customer database.Customer

	err := json.NewDecoder(r.Body).Decode(&customer)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	createdCustomer, err := database.Db.CreateCustomer(&customer)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusCreated, createdCustomer)
	if err != nil {
		log.Error("CreateCustomer", err)
	}
}

// GetCustomer godoc
//
//	@Summary		Get a customer by ID
//	@Tags			Customers
//	@Produce		json
//	@Param			id path int true "Customer ID"
//	@Success		200	{object}	database.Customer
//	@Failure		404	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers/{id} [get]
func GetCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	customer, err := database.Db.GetCustomerByID(id)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, customer)
	if err != nil {
		log.Error("GetCustomer", err)
	}
}

// ListCustomers godoc
//
//	@Summary		List all customers
//	@Tags			Customers
//	@Produce		json
//	@Success		200	{array}	database.Customer
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers [get]
func ListCustomers(w http.ResponseWriter, r *http.Request) {
	customers, err := database.Db.ListCustomers()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, customers)
	if err != nil {
		log.Error("ListCustomers", err)
	}
}

// UpdateCustomer godoc
//
//	@Summary		Update a customer
//	@Tags			Customers
//	@Accept			json
//	@Produce		json
//	@Param			id path int true "Customer ID"
//	@Param			customer body database.Customer true "Customer data"
//	@Success		200	{object}	database.Customer
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers/{id} [put]
func UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	var customer database.Customer
	err = json.NewDecoder(r.Body).Decode(&customer)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	customer.ID = id

	oldCustomer, err := database.Db.GetCustomerByID(id)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}

	updatedCustomer, err := database.Db.UpdateCustomer(&customer)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	if updatedCustomer.KeycloakID != "" {
		oldGroups := oldCustomer.LicenseGroups
		newGroups := updatedCustomer.LicenseGroups
		go func() {
			if err := keycloak.KeycloakClient.SyncLicenseGroupsDiffToKeycloak(updatedCustomer.KeycloakID, oldGroups, newGroups); err != nil {
				log.Error("UpdateCustomer: failed to sync license groups to Keycloak: ", err)
			}
		}()
	}

	err = utils.WriteJSON(w, http.StatusOK, updatedCustomer)
	if err != nil {
		log.Error("UpdateCustomer", err)
	}
}

// DeleteCustomer godoc
//
//	@Summary		Delete a customer
//	@Tags			Customers
//	@Param			id path int true "Customer ID"
//	@Success		204
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers/{id} [delete]
func DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.DeleteCustomer(id)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
