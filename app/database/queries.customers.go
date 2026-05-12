package database

import (
	"context"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entcustomer "github.com/augustin-wien/augustina-backend/ent/customer"
)

// Customer represents a customer in the system
type Customer struct {
	ID            int        `json:"id"`
	KeycloakID    string     `json:"keycloakid"`
	Email         string     `json:"email"`
	FirstName     string     `json:"firstname"`
	LastName      string     `json:"lastname"`
	LicenseGroups []string   `json:"licensegroups"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

func licenseGroupsFromString(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func licenseGroupsToString(groups []string) string {
	return strings.Join(groups, ",")
}

// CustomerEntIntoCustomer converts an ent.Customer to Customer struct
func (db *Database) CustomerEntIntoCustomer(c *ent.Customer) Customer {
	return Customer{
		ID:            c.ID,
		KeycloakID:    c.Keycloakid,
		Email:         c.Email,
		FirstName:     c.Firstname,
		LastName:      c.Lastname,
		LicenseGroups: licenseGroupsFromString(c.Licensegroups),
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

// CreateCustomer creates a new customer in the database
func (db *Database) CreateCustomer(customer *Customer) (*Customer, error) {
	ctx := context.Background()

	entCustomer, err := db.EntClient.Customer.Create().
		SetKeycloakid(customer.KeycloakID).
		SetEmail(customer.Email).
		SetFirstname(customer.FirstName).
		SetLastname(customer.LastName).
		SetLicensegroups(licenseGroupsToString(customer.LicenseGroups)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	result := db.CustomerEntIntoCustomer(entCustomer)
	return &result, nil
}

// GetCustomerByID retrieves a customer by ID
func (db *Database) GetCustomerByID(id int) (*Customer, error) {
	ctx := context.Background()

	entCustomer, err := db.EntClient.Customer.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	result := db.CustomerEntIntoCustomer(entCustomer)
	return &result, nil
}

// GetCustomerByKeycloakID retrieves a customer by Keycloak ID
func (db *Database) GetCustomerByKeycloakID(keycloakID string) (*Customer, error) {
	ctx := context.Background()

	entCustomer, err := db.EntClient.Customer.Query().
		Where(entcustomer.Keycloakid(keycloakID)).
		Only(ctx)

	if err != nil {
		return nil, err
	}

	result := db.CustomerEntIntoCustomer(entCustomer)
	return &result, nil
}

// ListCustomers retrieves all customers
func (db *Database) ListCustomers() ([]*Customer, error) {
	ctx := context.Background()

	customers, err := db.EntClient.Customer.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Customer, len(customers))
	for i, c := range customers {
		customer := db.CustomerEntIntoCustomer(c)
		result[i] = &customer
	}

	return result, nil
}

// UpdateCustomer updates a customer in the database
func (db *Database) UpdateCustomer(customer *Customer) (*Customer, error) {
	ctx := context.Background()

	entCustomer, err := db.EntClient.Customer.UpdateOneID(customer.ID).
		SetKeycloakid(customer.KeycloakID).
		SetEmail(customer.Email).
		SetFirstname(customer.FirstName).
		SetLastname(customer.LastName).
		SetLicensegroups(licenseGroupsToString(customer.LicenseGroups)).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	result := db.CustomerEntIntoCustomer(entCustomer)
	return &result, nil
}

// DeleteCustomer deletes a customer from the database
func (db *Database) DeleteCustomer(id int) error {
	ctx := context.Background()
	return db.EntClient.Customer.DeleteOneID(id).Exec(ctx)
}

// GetCustomerByEmail retrieves a customer by email address
func (db *Database) GetCustomerByEmail(email string) (*Customer, error) {
	ctx := context.Background()

	entCustomer, err := db.EntClient.Customer.Query().
		Where(entcustomer.Email(email)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	result := db.CustomerEntIntoCustomer(entCustomer)
	return &result, nil
}

// GetOrCreateCustomerByEmail finds a customer by Keycloak ID or email, creating one if needed.
// keycloakID and email come from the Keycloak GetOrCreateUser call at purchase time.
// Safe to call concurrently: unique constraint violations on create fall back to a re-fetch.
func (db *Database) GetOrCreateCustomerByEmail(email, keycloakID string) (*Customer, error) {
	if keycloakID != "" {
		c, err := db.GetCustomerByKeycloakID(keycloakID)
		if err == nil {
			return c, nil
		}
	}

	c, err := db.GetCustomerByEmail(email)
	if err == nil {
		// Backfill keycloak ID if the record pre-dates the Keycloak account
		if c.KeycloakID == "" && keycloakID != "" {
			updated, updateErr := db.EntClient.Customer.UpdateOneID(c.ID).
				SetKeycloakid(keycloakID).
				SetUpdatedAt(time.Now()).
				Save(context.Background())
			if updateErr != nil {
				log.Warnf("GetOrCreateCustomerByEmail: backfill keycloakid failed for customer %d: %v", c.ID, updateErr)
				return c, nil // non-fatal: return the record we have
			}
			result := db.CustomerEntIntoCustomer(updated)
			return &result, nil
		}
		return c, nil
	}

	created, err := db.CreateCustomer(&Customer{
		KeycloakID: keycloakID,
		Email:      email,
	})
	if err != nil {
		// Race condition: another goroutine created the record between our lookup and insert.
		// Re-fetch instead of surfacing a constraint violation to the caller.
		if ent.IsConstraintError(err) {
			if keycloakID != "" {
				if c, rerr := db.GetCustomerByKeycloakID(keycloakID); rerr == nil {
					return c, nil
				}
			}
			return db.GetCustomerByEmail(email)
		}
		return nil, err
	}
	return created, nil
}

// AddLicenseGroupToCustomer adds a license group to a customer's licensegroups
func (db *Database) AddLicenseGroupToCustomer(customerID int, licenseGroup string) (*Customer, error) {
	ctx := context.Background()

	// Get current customer
	customer, err := db.GetCustomerByID(customerID)
	if err != nil {
		return nil, err
	}

	// Add license group (avoid duplicates)
	groups := customer.LicenseGroups
	found := false
	for _, g := range groups {
		if g == licenseGroup {
			found = true
			break
		}
	}
	if !found {
		groups = append(groups, licenseGroup)
	}

	// Update customer
	entCustomer, err := db.EntClient.Customer.UpdateOneID(customerID).
		SetLicensegroups(licenseGroupsToString(groups)).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	result := db.CustomerEntIntoCustomer(entCustomer)
	return &result, nil
}
