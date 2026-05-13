//go:build integration
// +build integration

package database

import (
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

// skipIfNoCustomerAbonementTables skips the test if customer/abonement tables don't exist
func SkipIfNoCustomerAbonementTables(t *testing.T) {
	_, err := Db.ListCustomers()
	if err != nil && err.Error() == "pq: relation \"customer\" does not exist" {
		t.Skip("Customer/Abonement tables do not exist - migrations need to be applied")
	}
}

func TestCustomerCRUD(t *testing.T) {
	config.InitConfig()
	err := Db.InitEmptyTestDb()
	require.NoError(t, err)
	SkipIfNoCustomerAbonementTables(t)

	// Test Create
	customer := &Customer{
		KeycloakID:    "test-keycloak-123",
		Email:         "test@example.com",
		FirstName:     "John",
		LastName:      "Doe",
		LicenseGroups: []string{"group1", "group2"},
	}

	createdCustomer, err := Db.CreateCustomer(customer)
	require.NoError(t, err)
	require.NotNil(t, createdCustomer)
	require.NotZero(t, createdCustomer.ID)
	require.Equal(t, "test-keycloak-123", createdCustomer.KeycloakID)

	// Test GetCustomerByID
	fetchedCustomer, err := Db.GetCustomerByID(createdCustomer.ID)
	require.NoError(t, err)
	require.Equal(t, createdCustomer.ID, fetchedCustomer.ID)
	require.Equal(t, "John", fetchedCustomer.FirstName)

	// Test GetCustomerByKeycloakID
	fetchedByKeycloak, err := Db.GetCustomerByKeycloakID("test-keycloak-123")
	require.NoError(t, err)
	require.Equal(t, createdCustomer.ID, fetchedByKeycloak.ID)

	// Test Update
	createdCustomer.LastName = "Smith"
	createdCustomer.LicenseGroups = []string{"group1", "group2", "group3"}
	updatedCustomer, err := Db.UpdateCustomer(createdCustomer)
	require.NoError(t, err)
	require.Equal(t, "Smith", updatedCustomer.LastName)
	require.Equal(t, []string{"group1", "group2", "group3"}, updatedCustomer.LicenseGroups)

	// Test AddLicenseGroupToCustomer
	result, err := Db.AddLicenseGroupToCustomer(createdCustomer.ID, "newgroup")
	require.NoError(t, err)
	require.Contains(t, result.LicenseGroups, "newgroup")

	// Test ListCustomers
	customer2 := &Customer{
		KeycloakID: "test-keycloak-456",
		Email:      "test2@example.com",
		FirstName:  "Jane",
		LastName:   "Smith",
	}
	_, err = Db.CreateCustomer(customer2)
	require.NoError(t, err)

	customers, err := Db.ListCustomers()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(customers), 2)

	// Test Delete
	err = Db.DeleteCustomer(createdCustomer.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = Db.GetCustomerByID(createdCustomer.ID)
	require.Error(t, err)
}

func TestAbonementCRUD(t *testing.T) {
	config.InitConfig()
	err := Db.InitEmptyTestDb()
	require.NoError(t, err)
	SkipIfNoCustomerAbonementTables(t)

	// Create test customer
	customer := &Customer{
		KeycloakID: "test-keycloak-abo",
		Email:      "abo@example.com",
		FirstName:  "Abo",
		LastName:   "Customer",
	}
	createdCustomer, err := Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Get or create test item
	items, err := Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	testItem := items[0]

	// Test Create Abonement
	fromDate := time.Now()
	toDate := fromDate.AddDate(0, 1, 0) // One month later

	abonement := &Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     testItem.ID,
		FromDate:   fromDate,
		ToDate:     toDate,
		Status:     "active",
	}

	createdAbonement, err := Db.CreateAbonement(abonement)
	require.NoError(t, err)
	require.NotNil(t, createdAbonement)
	require.NotZero(t, createdAbonement.ID)
	require.Equal(t, "active", createdAbonement.Status)

	// Test GetAbonementByID
	fetchedAbonement, err := Db.GetAbonementByID(createdAbonement.ID)
	require.NoError(t, err)
	require.Equal(t, createdAbonement.ID, fetchedAbonement.ID)
	require.Equal(t, createdCustomer.ID, fetchedAbonement.CustomerID)

	// Test ListAbonementsByCustomer
	customerAbonements, err := Db.ListAbonementsByCustomer(createdCustomer.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(customerAbonements), 1)

	// Test GetActiveAbonementsByDate
	activeAbonements, err := Db.GetActiveAbonementsByDate(time.Now())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(activeAbonements), 1)

	// Test Update
	createdAbonement.Status = "inactive"
	updatedAbonement, err := Db.UpdateAbonement(createdAbonement)
	require.NoError(t, err)
	require.Equal(t, "inactive", updatedAbonement.Status)

	// Test ListAbonements
	allAbonements, err := Db.ListAbonements()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(allAbonements), 1)

	// Test Delete
	err = Db.DeleteAbonement(createdAbonement.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = Db.GetAbonementByID(createdAbonement.ID)
	require.Error(t, err)
}

func TestAbonementDateQueries(t *testing.T) {
	config.InitConfig()
	err := Db.InitEmptyTestDb()
	require.NoError(t, err)
	SkipIfNoCustomerAbonementTables(t)

	// Create test customer
	customer := &Customer{
		KeycloakID: "test-keycloak-dates",
		Email:      "dates@example.com",
		FirstName:  "Date",
		LastName:   "Test",
	}
	createdCustomer, err := Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Get test item
	items, err := Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	testItem := items[0]

	// Create abonement for past date
	pastFromDate := time.Now().AddDate(0, -2, 0)
	pastToDate := time.Now().AddDate(0, -1, 0)
	pastAbonement := &Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     testItem.ID,
		FromDate:   pastFromDate,
		ToDate:     pastToDate,
		Status:     "active",
	}
	_, err = Db.CreateAbonement(pastAbonement)
	require.NoError(t, err)

	// Create abonement for future date
	futureFromDate := time.Now().AddDate(0, 1, 0)
	futureToDate := time.Now().AddDate(0, 2, 0)
	futureAbonement := &Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     testItem.ID,
		FromDate:   futureFromDate,
		ToDate:     futureToDate,
		Status:     "active",
	}
	_, err = Db.CreateAbonement(futureAbonement)
	require.NoError(t, err)

	// Create abonement active today
	todayFromDate := time.Now().AddDate(0, -1, 0)
	todayToDate := time.Now().AddDate(0, 1, 0)
	todayAbonement := &Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     testItem.ID,
		FromDate:   todayFromDate,
		ToDate:     todayToDate,
		Status:     "active",
	}
	_, err = Db.CreateAbonement(todayAbonement)
	require.NoError(t, err)

	// Test GetActiveAbonementsByDate for today
	activeToday, err := Db.GetActiveAbonementsByDate(time.Now())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(activeToday), 1)

	// Test GetAbonementsByDateRange
	rangeAbonements, err := Db.GetAbonementsByDateRange(
		time.Now().AddDate(0, -3, 0),
		time.Now().AddDate(0, 3, 0),
	)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rangeAbonements), 3)
}

func TestItemTypeField(t *testing.T) {
	config.InitConfig()
	err := Db.InitEmptyTestDb()
	require.NoError(t, err)
	SkipIfNoCustomerAbonementTables(t)

	// Get items and verify they have Type field
	items, err := Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	// Verify Type field is populated (can be any valid type)
	for _, item := range items {
		require.NotEmpty(t, item.Type)
		// Verify it's one of the valid types
		validTypes := map[string]bool{
			"normal_item":       true,
			"license_item":      true,
			"issue":             true,
			"donation":          true,
			"transaction_costs": true,
			"abonement":         true,
		}
		require.True(t, validTypes[item.Type], "Invalid item type: %s", item.Type)
	}
}
