//go:build integration
// +build integration

package database

import (
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

func TestProcessAbonementLicenseGroupsForDate(t *testing.T) {
	config.InitConfig()
	err := Db.InitEmptyTestDb()
	require.NoError(t, err)
	SkipIfNoCustomerAbonementTables(t)
	SkipIfNoCustomerAbonementTables(t)
	service := NewAbonementService(&Db)

	// Create test customer
	customer := &Customer{
		KeycloakID:    "test-service-customer",
		Email:         "service@example.com",
		FirstName:     "Service",
		LastName:      "Test",
		LicenseGroups: []string{},
	}
	createdCustomer, err := Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Get test item with license group
	items, err := Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	var itemWithLicense Item
	for _, item := range items {
		if item.LicenseGroup.Valid && item.LicenseGroup.String != "" {
			itemWithLicense = item
			break
		}
	}

	if itemWithLicense.ID == 0 && len(items) > 0 {
		// If no item with license found, use first item
		itemWithLicense = items[0]
	}

	// Create active abonement for today
	fromDate := time.Now().AddDate(0, -1, 0)
	toDate := time.Now().AddDate(0, 1, 0)

	abonement := &Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     itemWithLicense.ID,
		FromDate:   fromDate,
		ToDate:     toDate,
		Status:     "active",
	}

	createdAbonement, err := Db.CreateAbonement(abonement)
	require.NoError(t, err)

	// Process abonements for today
	err = service.ProcessAbonementLicenseGroupsForDate(time.Now())
	require.NoError(t, err)

	// Verify customer's license groups were updated
	updatedCustomer, err := Db.GetCustomerByID(createdCustomer.ID)
	require.NoError(t, err)

	// License group should be added (it might already be there or this is new)
	require.NotEmpty(t, updatedCustomer.LicenseGroups)

	// Cleanup
	err = Db.DeleteAbonement(createdAbonement.ID)
	require.NoError(t, err)
}

func TestProcessAbonementForCustomer(t *testing.T) {
	config.InitConfig()
	err := Db.InitEmptyTestDb()
	require.NoError(t, err)
	SkipIfNoCustomerAbonementTables(t)

	service := NewAbonementService(&Db)

	// Create test customer
	customer := &Customer{
		KeycloakID:    "test-process-customer",
		Email:         "process@example.com",
		FirstName:     "Process",
		LastName:      "Test",
		LicenseGroups: []string{},
	}
	createdCustomer, err := Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Get test item
	items, err := Db.ListItems(false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	testItem := items[0]

	// Create multiple abonements
	fromDate1 := time.Now().AddDate(0, -1, 0)
	toDate1 := time.Now().AddDate(0, 1, 0)

	abonement1 := &Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     testItem.ID,
		FromDate:   fromDate1,
		ToDate:     toDate1,
		Status:     "active",
	}
	createdAbonement1, err := Db.CreateAbonement(abonement1)
	require.NoError(t, err)

	// Create inactive abonement (should not be included)
	inactiveFromDate := time.Now().AddDate(0, -1, 0)
	inactiveToDate := time.Now().AddDate(0, 1, 0)

	inactiveAbonement := &Abonement{
		CustomerID: createdCustomer.ID,
		ItemID:     testItem.ID,
		FromDate:   inactiveFromDate,
		ToDate:     inactiveToDate,
		Status:     "inactive",
	}
	createdInactiveAbonement, err := Db.CreateAbonement(inactiveAbonement)
	require.NoError(t, err)

	// Process customer's active abonements
	err = service.ProcessAbonementForCustomer(createdCustomer.ID, time.Now())
	require.NoError(t, err)

	// Cleanup
	err = Db.DeleteAbonement(createdAbonement1.ID)
	require.NoError(t, err)
	err = Db.DeleteAbonement(createdInactiveAbonement.ID)
	require.NoError(t, err)
}

func TestAbonementServiceWithoutActiveAbonements(t *testing.T) {
	config.InitConfig()
	err := Db.InitEmptyTestDb()
	require.NoError(t, err)
	SkipIfNoCustomerAbonementTables(t)
	SkipIfNoCustomerAbonementTables(t)

	service := NewAbonementService(&Db)

	// Create test customer with no abonements
	customer := &Customer{
		KeycloakID:    "test-no-abo-customer",
		Email:         "noabo@example.com",
		FirstName:     "NoAbo",
		LastName:      "Test",
		LicenseGroups: []string{},
	}
	createdCustomer, err := Db.CreateCustomer(customer)
	require.NoError(t, err)

	// Process should complete without error even with no abonements
	err = service.ProcessAbonementLicenseGroupsForDate(time.Now())
	require.NoError(t, err)

	// Process for customer should also complete without error
	err = service.ProcessAbonementForCustomer(createdCustomer.ID, time.Now())
	require.NoError(t, err)

	// Customer's license groups should remain unchanged
	updatedCustomer, err := Db.GetCustomerByID(createdCustomer.ID)
	require.NoError(t, err)
	require.Empty(t, updatedCustomer.LicenseGroups)
}
