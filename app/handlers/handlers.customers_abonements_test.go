package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"

	"github.com/augustin-wien/augustina-backend/config"
	dbpkg "github.com/augustin-wien/augustina-backend/database"
)

func skipIfNoCustomerAbonementTables(t *testing.T) {
	_, err := dbpkg.Db.ListCustomers()
	if err != nil && err.Error() == "pq: relation \"customer\" does not exist" {
		t.Skip("Customer/Abonement tables do not exist - migrations need to be applied")
	}
}

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

// Abonement Handler Tests

func TestCreateAbonementHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create test customer
	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-abo-create",
		Email:         "aboCreate@example.com",
		FirstName:     "Abo",
		LastName:      "Create",
		LicenseGroups: []string{},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Get test item
	items, err := dbpkg.Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	// Prepare abonement request
	fromDate := time.Now()
	toDate := fromDate.AddDate(0, 1, 0)

	body := dbpkg.Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     items[0].ID,
		FromDate:   fromDate,
		ToDate:     toDate,
		Status:     "active",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/abonements", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	CreateAbonement(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var response dbpkg.Abonement
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.NotZero(t, response.ID)
	require.Equal(t, createdCustomer.ID, response.CustomerID)
}

func TestGetAbonementHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create test customer
	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-abo-get",
		Email:         "aboGet@example.com",
		FirstName:     "Abo",
		LastName:      "Get",
		LicenseGroups: []string{},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Get test item and create abonement
	items, err := dbpkg.Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	fromDate := time.Now()
	toDate := fromDate.AddDate(0, 1, 0)

	abonement := &dbpkg.Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     items[0].ID,
		FromDate:   fromDate,
		ToDate:     toDate,
		Status:     "active",
	}
	createdAbonement, err := dbpkg.Db.CreateAbonement(abonement)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/abonements/"+strconv.Itoa(createdAbonement.ID), nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", strconv.Itoa(createdAbonement.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	rr := httptest.NewRecorder()

	GetAbonement(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response dbpkg.Abonement
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, createdAbonement.ID, response.ID)
}

func TestListAbonementsByCustomerHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create test customer
	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-abo-list",
		Email:         "aboList@example.com",
		FirstName:     "Abo",
		LastName:      "List",
		LicenseGroups: []string{},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/customers/"+strconv.Itoa(createdCustomer.ID)+"/abonements/", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", strconv.Itoa(createdCustomer.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	rr := httptest.NewRecorder()

	ListAbonementsByCustomer(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response []dbpkg.Abonement
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	// Response should be empty or have abonements
	require.NotNil(t, response)
}

func TestUpdateAbonementHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create test customer and abonement
	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-abo-update",
		Email:         "aboUpdate@example.com",
		FirstName:     "Abo",
		LastName:      "Update",
		LicenseGroups: []string{},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	items, err := dbpkg.Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	fromDate := time.Now()
	toDate := fromDate.AddDate(0, 1, 0)

	abonement := &dbpkg.Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     items[0].ID,
		FromDate:   fromDate,
		ToDate:     toDate,
		Status:     "active",
	}
	createdAbonement, err := dbpkg.Db.CreateAbonement(abonement)
	require.NoError(t, err)

	// Update the abonement
	updatedAbonement := *createdAbonement
	updatedAbonement.Status = "inactive"

	b, err := json.Marshal(updatedAbonement)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/abonements/"+strconv.Itoa(createdAbonement.ID), bytes.NewReader(b))
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", strconv.Itoa(createdAbonement.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	UpdateAbonement(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response dbpkg.Abonement
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "inactive", response.Status)
}

func TestDeleteAbonementHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	// Create test customer and abonement
	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-abo-delete",
		Email:         "aboDelete@example.com",
		FirstName:     "Abo",
		LastName:      "Delete",
		LicenseGroups: []string{},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	items, err := dbpkg.Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	fromDate := time.Now()
	toDate := fromDate.AddDate(0, 1, 0)

	abonement := &dbpkg.Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     items[0].ID,
		FromDate:   fromDate,
		ToDate:     toDate,
		Status:     "active",
	}
	createdAbonement, err := dbpkg.Db.CreateAbonement(abonement)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/api/abonements/"+strconv.Itoa(createdAbonement.ID), nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", strconv.Itoa(createdAbonement.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	rr := httptest.NewRecorder()

	DeleteAbonement(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
}

func TestListMyAbonementsHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	customer := &dbpkg.Customer{
		KeycloakID:    "test-keycloak-my-abo",
		Email:         "my-abo@example.com",
		FirstName:     "My",
		LastName:      "Abo",
		LicenseGroups: []string{},
	}
	createdCustomer, err := dbpkg.Db.CreateCustomer(customer)
	require.NoError(t, err)

	items, err := dbpkg.Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	abonement := &dbpkg.Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     items[0].ID,
		FromDate:   time.Now(),
		ToDate:     time.Now().AddDate(0, 1, 0),
		Status:     "active",
	}
	createdAbonement, err := dbpkg.Db.CreateAbonement(abonement)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/customers/me/abonements/", nil)
	req.Header.Set("X-Auth-User", createdCustomer.KeycloakID)
	rr := httptest.NewRecorder()

	ListMyAbonements(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response []dbpkg.Abonement
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Len(t, response, 1)
	require.Equal(t, createdAbonement.ID, response[0].ID)
	require.Equal(t, createdCustomer.ID, response[0].CustomerID)
}

func TestListMyPaymentsHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	vendorID, err := dbpkg.Db.CreateVendor(dbpkg.Vendor{LicenseID: null.StringFrom("test-vendor-my-payments")})
	require.NoError(t, err)
	vendorAccount, err := dbpkg.Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	anonAccount, err := dbpkg.Db.GetAccountByType("UserAnon")
	require.NoError(t, err)
	itemID, err := dbpkg.Db.CreateItem(dbpkg.Item{Name: "My Payments Item", Description: "Test item", Price: 100, Type: "normal_item"})
	require.NoError(t, err)

	customerEmail := "my-payments@example.com"
	otherEmail := "other-payments@example.com"

	order1ID, err := dbpkg.Db.CreateOrder(dbpkg.Order{
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom(customerEmail),
		Entries: []dbpkg.OrderEntry{{
			Item:     itemID,
			Quantity: 1,
			Sender:   anonAccount.ID,
			Receiver: vendorAccount.ID,
			IsSale:   true,
		}},
	})
	require.NoError(t, err)

	order2ID, err := dbpkg.Db.CreateOrder(dbpkg.Order{
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom(otherEmail),
		Entries: []dbpkg.OrderEntry{{
			Item:     itemID,
			Quantity: 1,
			Sender:   anonAccount.ID,
			Receiver: vendorAccount.ID,
			IsSale:   true,
		}},
	})
	require.NoError(t, err)

	_, err = dbpkg.Db.CreatePayment(dbpkg.Payment{
		Sender:       anonAccount.ID,
		Receiver:     vendorAccount.ID,
		Amount:       100,
		AuthorizedBy: "test",
		Order:        null.NewInt(int64(order1ID), true),
		Item:         null.NewInt(int64(itemID), true),
		Quantity:     1,
		Price:        100,
		IsSale:       true,
	})
	require.NoError(t, err)

	_, err = dbpkg.Db.CreatePayment(dbpkg.Payment{
		Sender:       anonAccount.ID,
		Receiver:     vendorAccount.ID,
		Amount:       100,
		AuthorizedBy: "test",
		Order:        null.NewInt(int64(order2ID), true),
		Item:         null.NewInt(int64(itemID), true),
		Quantity:     1,
		Price:        100,
		IsSale:       true,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/customers/me/payments/", nil)
	req.Header.Set("X-Auth-User-Email", customerEmail)
	rr := httptest.NewRecorder()

	ListMyPayments(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response []dbpkg.Payment
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Len(t, response, 1)
	require.True(t, response[0].Order.Valid)
	require.Equal(t, order1ID, int(response[0].Order.Int64))
}

func TestListActiveAbonementsWithCustomersHandler(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	config.InitConfig()
	err := dbpkg.Db.InitEmptyTestDb()
	require.NoError(t, err)
	skipIfNoCustomerAbonementTables(t)

	items, err := dbpkg.Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	activeCustomer, err := dbpkg.Db.CreateCustomer(&dbpkg.Customer{
		KeycloakID: "active-customer-keycloak",
		Email:      "active-customer@example.com",
		FirstName:  "Active",
		LastName:   "Customer",
	})
	require.NoError(t, err)

	inactiveCustomer, err := dbpkg.Db.CreateCustomer(&dbpkg.Customer{
		KeycloakID: "inactive-customer-keycloak",
		Email:      "inactive-customer@example.com",
		FirstName:  "Inactive",
		LastName:   "Customer",
	})
	require.NoError(t, err)

	_, err = dbpkg.Db.CreateAbonement(&dbpkg.Abonement{
		CustomerID: activeCustomer.ID,
		ItemID:     items[0].ID,
		FromDate:   time.Now().AddDate(0, 0, -1),
		ToDate:     time.Now().AddDate(0, 0, 7),
		Status:     "active",
	})
	require.NoError(t, err)

	_, err = dbpkg.Db.CreateAbonement(&dbpkg.Abonement{
		CustomerID: inactiveCustomer.ID,
		ItemID:     items[0].ID,
		FromDate:   time.Now().AddDate(0, 0, -10),
		ToDate:     time.Now().AddDate(0, 0, -5),
		Status:     "active",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/abonements/active/", nil)
	rr := httptest.NewRecorder()

	ListActiveAbonementsWithCustomers(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response []ActiveAbonementWithCustomer
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Len(t, response, 1)
	require.Equal(t, activeCustomer.ID, response[0].Customer.ID)
	require.Equal(t, activeCustomer.Email, response[0].Customer.Email)
	require.Equal(t, "active", response[0].Abonement.Status)
}
