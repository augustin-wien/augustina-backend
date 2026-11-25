package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/augustin-wien/augustina-backend/config"
	dbpkg "github.com/augustin-wien/augustina-backend/database"
)

func TestCreateCustomerHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Prepare request body
	body := dbpkg.Customer{
		KeycloakID:    "test-keycloak-create",
		Email:         "create@example.com",
		FirstName:     "Create",
		LastName:      "Test",
		LicenseGroups: []string{"group1"},
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	CreateCustomer(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var response dbpkg.Customer
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.NotZero(t, response.ID)
	require.Equal(t, "test-keycloak-create", response.KeycloakID)
}

func TestGetCustomerHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create a test customer first
	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-get",
		Email:         "get@example.com",
		FirstName:     "Get",
		LastName:      "Test",
		LicenseGroups: []string{"group1"},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Make request to get the customer
	req := httptest.NewRequest(http.MethodGet, "/api/customers/"+strconv.Itoa(createdCustomer.ID), nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", strconv.Itoa(createdCustomer.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	rr := httptest.NewRecorder()

	GetCustomer(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response dbpkg.Customer
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, createdCustomer.ID, response.ID)
}

func TestListCustomersHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create test customers
	for i := 1; i <= 2; i++ {
		customer := &dbpkg.Customer{
			KeycloakID:    "test-keycloak-list-" + strconv.Itoa(i),
			Email:         "list" + strconv.Itoa(i) + "@example.com",
			FirstName:     "List",
			LastName:      "Test",
			LicenseGroups: []string{},
		}
		_, err := dbpkg.Db.CreateCustomer(customer)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	rr := httptest.NewRecorder()

	ListCustomers(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response []dbpkg.Customer
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(response), 2)
}

func TestUpdateCustomerHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create a test customer first
	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-update",
		Email:         "update@example.com",
		FirstName:     "Update",
		LastName:      "Test",
		LicenseGroups: []string{"group1"},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Update the customer
	updatedCustomer := *createdCustomer
	updatedCustomer.LastName = "Updated"

	b, err := json.Marshal(updatedCustomer)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/customers/"+strconv.Itoa(createdCustomer.ID), bytes.NewReader(b))
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", strconv.Itoa(createdCustomer.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	UpdateCustomer(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response dbpkg.Customer
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "Updated", response.LastName)
}

func TestDeleteCustomerHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create a test customer first
	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-delete",
		Email:         "delete@example.com",
		FirstName:     "Delete",
		LastName:      "Test",
		LicenseGroups: []string{"group1"},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/api/customers/"+strconv.Itoa(createdCustomer.ID), nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", strconv.Itoa(createdCustomer.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	rr := httptest.NewRecorder()

	DeleteCustomer(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
}
