package database

import (
	"time"

	"github.com/augustin-wien/augustina-backend/keycloak"
)

// DigitalLicenseAssigner assigns a digital license group to a Keycloak user.
// Extracted as an interface so it can be replaced with a mock in tests.
type DigitalLicenseAssigner interface {
	AssignDigitalLicenseGroup(userID string, licenseGroup string) error
}

// AbonementService handles business logic related to abonements
type AbonementService struct {
	db               *Database
	keycloakAssigner DigitalLicenseAssigner
}

// NewAbonementService creates a new AbonementService backed by the global Keycloak client.
func NewAbonementService(db *Database) *AbonementService {
	return &AbonementService{db: db, keycloakAssigner: &keycloak.KeycloakClient}
}

// ProcessAbonementLicenseGroupsForDate processes and updates license groups for customers
// with active abonements on a given date. This is called when a new issue is created.
func (as *AbonementService) ProcessAbonementLicenseGroupsForDate(issueDate time.Time) error {
	// Get all active abonements for this date
	abonements, err := as.db.GetActiveAbonementsByDate(issueDate)
	if err != nil {
		return err
	}

	// For each abonement, update the customer's license groups
	for _, abonement := range abonements {
		// Get the item to get its license group
		item, err := as.db.GetItem(abonement.ItemID)
		if err != nil {
			continue // Skip if item not found
		}

		// Add the license group to the customer (handle null.String)
		licenseGroup := ""
		if item.LicenseGroup.Valid {
			licenseGroup = item.LicenseGroup.String
		}
		updatedCustomer, err := as.db.AddLicenseGroupToCustomer(abonement.CustomerID, licenseGroup)
		if err != nil {
			// Log error but continue processing other abonements
			continue
		}

		if updatedCustomer.KeycloakID != "" {
			_ = as.SyncAbonementLicensesToKeycloak(updatedCustomer.KeycloakID, updatedCustomer.LicenseGroups)
		}
	}

	return nil
}

// ProcessAbonementForCustomer updates a customer's license groups based on their active abonements
func (as *AbonementService) ProcessAbonementForCustomer(customerID int, checkDate time.Time) error {
	// Get all abonements for the customer that are active on checkDate
	abonements, err := as.db.ListAbonementsByCustomer(customerID)
	if err != nil {
		return err
	}

	var licenseGroups []string
	now := checkDate

	for _, abonement := range abonements {
		// Check if abonement is active on the check date
		if abonement.FromDate.Before(now) && abonement.ToDate.After(now) && abonement.Status == "active" {
			// Get the item to get its license group
			item, err := as.db.GetItem(abonement.ItemID)
			if err != nil {
				continue
			}

			// Get license group value, handling null.String
			licenseGroupValue := ""
			if item.LicenseGroup.Valid {
				licenseGroupValue = item.LicenseGroup.String
			}

			// Add license group if not already in the list
			found := false
			for _, lg := range licenseGroups {
				if lg == licenseGroupValue {
					found = true
					break
				}
			}
			if !found && licenseGroupValue != "" {
				licenseGroups = append(licenseGroups, licenseGroupValue)
			}
		}
	}

	// Update customer with combined license groups
	customer, err := as.db.GetCustomerByID(customerID)
	if err != nil {
		return err
	}

	customer.LicenseGroups = licenseGroups
	_, err = as.db.UpdateCustomer(customer)
	if err != nil {
		return err
	}

	if customer.KeycloakID != "" {
		return as.SyncAbonementLicensesToKeycloak(customer.KeycloakID, licenseGroups)
	}

	return nil
}

// SyncAbonementLicensesToKeycloak assigns each license group to the user's Keycloak account.
func (as *AbonementService) SyncAbonementLicensesToKeycloak(keycloakID string, licenseGroups []string) error {
	for _, group := range licenseGroups {
		if group == "" {
			continue
		}
		if err := as.keycloakAssigner.AssignDigitalLicenseGroup(keycloakID, group); err != nil {
			return err
		}
	}
	return nil
}
