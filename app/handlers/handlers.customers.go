package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/augustin-wien/augustina-backend/config"
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

	// Keep the Keycloak login in step with a changed email, before the DB so
	// both don't drift apart when Keycloak refuses (e.g. address already taken)
	if utils.ToLower(customer.Email) != utils.ToLower(oldCustomer.Email) {
		keycloakID := oldCustomer.KeycloakID
		// Customers whose record predates their login aren't linked yet
		if keycloakID == "" && oldCustomer.Email != "" {
			if user, err := keycloak.KeycloakClient.GetUserByEmail(oldCustomer.Email); err == nil && user.ID != nil {
				keycloakID = *user.ID
				if customer.KeycloakID == "" {
					customer.KeycloakID = keycloakID
				}
			}
		}
		if keycloakID != "" {
			if err := keycloak.KeycloakClient.UpdateUserById(keycloakID, customer.Email, customer.FirstName, customer.LastName, customer.Email); err != nil {
				log.Error("UpdateCustomer: failed to update email in Keycloak: ", err)
				utils.ErrorJSON(w, err, http.StatusBadRequest)
				return
			}
		}
	}

	updatedCustomer, err := database.Db.UpdateCustomer(&customer)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	if updatedCustomer.KeycloakID != "" {
		if err := keycloak.KeycloakClient.SyncLicenseGroupsDiffToKeycloak(updatedCustomer.KeycloakID, oldCustomer.LicenseGroups, updatedCustomer.LicenseGroups); err != nil {
			log.Error("UpdateCustomer: failed to sync license groups to Keycloak: ", err)
		}
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

// SendCustomerPasswordResetEmail godoc
//
//	@Summary		Send the customer a password reset email
//	@Tags			Customers
//	@Param			id path int true "Customer ID"
//	@Success		204
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers/{id}/password-reset-email/ [post]
func SendCustomerPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	sendCustomerActionEmail(w, r, "password reset", keycloak.KeycloakClient.SendPasswordResetEmailToUser)
}

// SendCustomerVerifyEmail godoc
//
//	@Summary		Send the customer an email to verify their address
//	@Tags			Customers
//	@Param			id path int true "Customer ID"
//	@Success		204
//	@Failure		400	{object}	utils.ErrorResponse
//	@Security		KeycloakAuth
//	@Router			/customers/{id}/verify-email/ [post]
func SendCustomerVerifyEmail(w http.ResponseWriter, r *http.Request) {
	sendCustomerActionEmail(w, r, "verification", keycloak.KeycloakClient.SendVerifyEmailToUser)
}

// sendCustomerActionEmail sends the customer a Keycloak action mail, creating
// their login first if they don't have one yet
func sendCustomerActionEmail(w http.ResponseWriter, r *http.Request, kind string, send func(userID string) error) {
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
	// The customer mails link back to the online paper; without it Keycloak's
	// helper skips sending silently
	if config.Config.OnlinePaperUrl == "" {
		utils.ErrorJSON(w, errors.New("the online paper URL is not configured in the settings"), http.StatusServiceUnavailable)
		return
	}
	sendActionEmail(w, r, kind, "customer", id, customer.Email, func(string) error {
		userID, err := ensureCustomerKeycloakUser(customer)
		if err != nil {
			return err
		}
		return send(userID)
	})
}

// ensureCustomerKeycloakUser returns the customer's Keycloak user ID, creating
// the user if needed. A newly linked user gets the customer's groups and its
// ID is stored on the customer.
func ensureCustomerKeycloakUser(customer *database.Customer) (string, error) {
	userID, created, err := keycloak.KeycloakClient.GetOrCreateCustomerUser(customer.KeycloakID, customer.Email, customer.FirstName, customer.LastName)
	if err != nil {
		return "", err
	}
	if userID == customer.KeycloakID {
		return userID, nil
	}
	if created {
		if err := keycloak.KeycloakClient.AssignGroup(userID, "customer"); err != nil {
			log.Error("ensureCustomerKeycloakUser: failed to assign customer group: ", customer.ID, err)
		}
	}
	if err := keycloak.KeycloakClient.SyncLicenseGroupsDiffToKeycloak(userID, nil, customer.LicenseGroups); err != nil {
		log.Error("ensureCustomerKeycloakUser: failed to sync license groups: ", customer.ID, err)
	}
	if err := database.Db.SetCustomerKeycloakID(customer.ID, userID); err != nil {
		log.Error("ensureCustomerKeycloakUser: failed to store keycloak id: ", customer.ID, err)
	}
	return userID, nil
}
