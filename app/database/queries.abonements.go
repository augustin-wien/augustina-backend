package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entabonement "github.com/augustin-wien/augustina-backend/ent/abonement"
)

// Abonement represents a subscription/abonement in the system
type Abonement struct {
	ID         int        `json:"id"`
	CustomerID int        `json:"customer_id"`
	ItemID     int        `json:"item_id"`
	FromDate   time.Time  `json:"from_date"`
	ToDate     time.Time  `json:"to_date"`
	Status     string     `json:"status"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

// UnmarshalJSON also accepts plain dates ("2006-01-02") for from_date and to_date, which is
// what the backoffice date inputs send. A plain from_date starts at the beginning of that day
// and a plain to_date runs until its end, so the abonement covers both days completely. Plain
// dates are UTC days, matching the backoffice, which shows the date part of the UTC timestamp.
func (a *Abonement) UnmarshalJSON(data []byte) error {
	type plain Abonement
	aux := struct {
		*plain
		FromDate string `json:"from_date"`
		ToDate   string `json:"to_date"`
	}{plain: (*plain)(a)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if a.FromDate, err = parseAbonementDate(aux.FromDate, false); err != nil {
		return fmt.Errorf("from_date: %w", err)
	}
	if a.ToDate, err = parseAbonementDate(aux.ToDate, true); err != nil {
		return fmt.Errorf("to_date: %w", err)
	}
	return nil
}

func parseAbonementDate(value string, endOfDay bool) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	day, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD or RFC3339", value)
	}
	if endOfDay {
		// Postgres stores microseconds and would round a nanosecond offset up to the next day
		return day.AddDate(0, 0, 1).Add(-time.Microsecond), nil
	}
	return day, nil
}

// AbonementEntIntoAbonement converts an ent.Abonement to Abonement struct
func (db *Database) AbonementEntIntoAbonement(a *ent.Abonement) Abonement {
	itemID := 0
	if a.Edges.Item != nil {
		itemID = a.Edges.Item.ID
	} else if a.ItemID != nil {
		itemID = *a.ItemID
	}

	return Abonement{
		ID:         a.ID,
		CustomerID: a.CustomerID,
		ItemID:     itemID,
		FromDate:   a.FromDate,
		ToDate:     a.ToDate,
		Status:     a.Status,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
	}
}

// CreateAbonement creates a new abonement in the database
func (db *Database) CreateAbonement(abonement *Abonement) (*Abonement, error) {
	ctx := context.Background()

	var itemID *int
	if abonement.ItemID > 0 {
		itemID = &abonement.ItemID
	}
	entAbonement, err := db.EntClient.Abonement.Create().
		SetCustomerID(abonement.CustomerID).
		SetNillableItemID(itemID).
		SetFromDate(abonement.FromDate).
		SetToDate(abonement.ToDate).
		SetStatus(abonement.Status).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	result := db.AbonementEntIntoAbonement(entAbonement)
	return &result, nil
}

// GetAbonementByID retrieves an abonement by ID
func (db *Database) GetAbonementByID(id int) (*Abonement, error) {
	ctx := context.Background()

	entAbonement, err := db.EntClient.Abonement.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	result := db.AbonementEntIntoAbonement(entAbonement)
	return &result, nil
}

// ListAbonements retrieves all abonements
func (db *Database) ListAbonements() ([]*Abonement, error) {
	ctx := context.Background()

	abonements, err := db.EntClient.Abonement.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Abonement, len(abonements))
	for i, a := range abonements {
		abonement := db.AbonementEntIntoAbonement(a)
		result[i] = &abonement
	}

	return result, nil
}

// ListAbonementsByCustomer retrieves all abonements for a customer
func (db *Database) ListAbonementsByCustomer(customerID int) ([]*Abonement, error) {
	ctx := context.Background()

	abonements, err := db.EntClient.Abonement.Query().
		Where(entabonement.CustomerID(customerID)).
		All(ctx)

	if err != nil {
		return nil, err
	}

	result := make([]*Abonement, len(abonements))
	for i, a := range abonements {
		abonement := db.AbonementEntIntoAbonement(a)
		result[i] = &abonement
	}

	return result, nil
}

// UpdateAbonement updates an abonement in the database
func (db *Database) UpdateAbonement(abonement *Abonement) (*Abonement, error) {
	ctx := context.Background()

	ub := db.EntClient.Abonement.UpdateOneID(abonement.ID).
		SetFromDate(abonement.FromDate).
		SetToDate(abonement.ToDate).
		SetStatus(abonement.Status).
		SetUpdatedAt(time.Now())
	if abonement.ItemID > 0 {
		ub = ub.SetItemID(abonement.ItemID)
	} else {
		ub = ub.ClearItem()
	}
	entAbonement, err := ub.Save(ctx)

	if err != nil {
		return nil, err
	}

	result := db.AbonementEntIntoAbonement(entAbonement)
	return &result, nil
}

// DeleteAbonement deletes an abonement from the database
func (db *Database) DeleteAbonement(id int) error {
	ctx := context.Background()
	return db.EntClient.Abonement.DeleteOneID(id).Exec(ctx)
}

// GetActiveAbonementsByDate retrieves all active abonements for a given date
func (db *Database) GetActiveAbonementsByDate(date time.Time) ([]*Abonement, error) {
	ctx := context.Background()

	abonements, err := db.EntClient.Abonement.Query().
		Where(
			entabonement.FromDateLTE(date),
			entabonement.ToDateGTE(date),
			entabonement.Status("active"),
		).
		All(ctx)

	if err != nil {
		return nil, err
	}

	result := make([]*Abonement, len(abonements))
	for i, a := range abonements {
		abonement := db.AbonementEntIntoAbonement(a)
		result[i] = &abonement
	}

	return result, nil
}

// GetAbonementsByDateRange retrieves all abonements that overlap with a date range
func (db *Database) GetAbonementsByDateRange(fromDate, toDate time.Time) ([]*Abonement, error) {
	ctx := context.Background()

	abonements, err := db.EntClient.Abonement.Query().
		Where(
			entabonement.FromDateLTE(toDate),
			entabonement.ToDateGTE(fromDate),
			entabonement.Status("active"),
		).
		All(ctx)

	if err != nil {
		return nil, err
	}

	result := make([]*Abonement, len(abonements))
	for i, a := range abonements {
		abonement := db.AbonementEntIntoAbonement(a)
		result[i] = &abonement
	}

	return result, nil
}
