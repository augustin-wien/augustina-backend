package database

import (
	"context"
	"errors"
	"math"

	"github.com/augustin-wien/augustina-backend/config"
	ent "github.com/augustin-wien/augustina-backend/ent"
	entitem "github.com/augustin-wien/augustina-backend/ent/item"
	"github.com/jackc/pgx/v5"
	"gopkg.in/guregu/null.v4"
)

func (db *Database) ListItems(skipHiddenItems bool, skipLicenses bool, skipDisabled bool) ([]Item, error) {
	var out []Item
	ctx := context.Background()
	ents, err := db.EntClient.Item.Query().Where(entitem.ArchivedEQ(false), entitem.DisabledEQ(skipDisabled)).Order(ent.Desc(entitem.FieldItemOrder)).WithLicenseItem().WithPDF().All(ctx)
	if err != nil {
		log.Error("ListItems (ent): ", err)
		return out, err
	}
	for _, e := range ents {
		var it Item
		it.ID = e.ID
		it.Name = e.Name
		it.Description = e.Description
		it.Price = int(math.Round(e.Price))
		it.Image = e.Image
		it.Archived = e.Archived
		it.Disabled = e.Disabled
		it.IsLicenseItem = e.IsLicenseItem
		if e.LicenseGroup != "" {
			it.LicenseGroup = null.NewString(e.LicenseGroup, true)
		}
		it.IsPDFItem = e.IsPDFItem
		if e.Edges.PDF != nil {
			it.PDF = null.IntFrom(int64(e.Edges.PDF.ID))
		}
		it.ItemOrder = e.ItemOrder
		if e.ItemColor != "" {
			it.ItemColor = null.NewString(e.ItemColor, true)
		}
		if e.ItemTextColor != "" {
			it.ItemTextColor = null.NewString(e.ItemTextColor, true)
		}
		if e.Edges.LicenseItem != nil {
			it.LicenseItem = null.IntFrom(int64(e.Edges.LicenseItem.ID))
		}

		if skipHiddenItems && (it.Name == config.Config.TransactionCostsName || it.Name == config.Config.DonationName) {
			continue
		}
		if skipLicenses && it.IsLicenseItem {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

// ListItems returns all items from the database.
func (db *Database) ListItemsWithDisabled(skipHiddenItems bool, skipLicenses bool) ([]Item, error) {
	var out []Item
	ctx := context.Background()
	ents, err := db.EntClient.Item.Query().Where(entitem.ArchivedEQ(false)).Order(ent.Desc(entitem.FieldItemOrder)).WithLicenseItem().WithPDF().All(ctx)
	if err != nil {
		log.Error("ListItems (ent): ", err)
		return out, err
	}
	for _, e := range ents {
		var it Item
		it.ID = e.ID
		it.Name = e.Name
		it.Description = e.Description
		it.Price = int(math.Round(e.Price))
		it.Image = e.Image
		it.Archived = e.Archived
		it.Disabled = e.Disabled
		it.IsLicenseItem = e.IsLicenseItem
		if e.LicenseGroup != "" {
			it.LicenseGroup = null.NewString(e.LicenseGroup, true)
		}
		it.IsPDFItem = e.IsPDFItem
		if e.Edges.PDF != nil {
			it.PDF = null.IntFrom(int64(e.Edges.PDF.ID))
		}
		it.ItemOrder = e.ItemOrder
		if e.ItemColor != "" {
			it.ItemColor = null.NewString(e.ItemColor, true)
		}
		if e.ItemTextColor != "" {
			it.ItemTextColor = null.NewString(e.ItemTextColor, true)
		}
		if e.Edges.LicenseItem != nil {
			it.LicenseItem = null.IntFrom(int64(e.Edges.LicenseItem.ID))
		}

		if skipHiddenItems && (it.Name == config.Config.TransactionCostsName || it.Name == config.Config.DonationName) {
			continue
		}
		if skipLicenses && it.IsLicenseItem {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

func (db *Database) ListItemsShop() ([]Item, error) {
	var out []Item
	ctx := context.Background()
	ents, err := db.EntClient.Item.Query().Where(entitem.ArchivedEQ(false), entitem.DisabledEQ(false)).Order(ent.Desc(entitem.FieldItemOrder)).WithLicenseItem().WithPDF().All(ctx)
	if err != nil {
		log.Error("ListItems (ent): ", err)
		return out, err
	}
	skipHiddenItems := true
	skipLicenses := true
	for _, e := range ents {
		var it Item
		it.ID = e.ID
		it.Name = e.Name
		it.Description = e.Description
		it.Price = int(math.Round(e.Price))
		it.Image = e.Image
		it.Archived = e.Archived
		it.Disabled = e.Disabled
		it.IsLicenseItem = e.IsLicenseItem
		if e.LicenseGroup != "" {
			it.LicenseGroup = null.NewString(e.LicenseGroup, true)
		}
		it.IsPDFItem = e.IsPDFItem
		if e.Edges.PDF != nil {
			it.PDF = null.IntFrom(int64(e.Edges.PDF.ID))
		}
		it.ItemOrder = e.ItemOrder
		if e.ItemColor != "" {
			it.ItemColor = null.NewString(e.ItemColor, true)
		}
		if e.ItemTextColor != "" {
			it.ItemTextColor = null.NewString(e.ItemTextColor, true)
		}
		if e.Edges.LicenseItem != nil {
			it.LicenseItem = null.IntFrom(int64(e.Edges.LicenseItem.ID))
		}

		if skipHiddenItems && (it.Name == config.Config.TransactionCostsName || it.Name == config.Config.DonationName) {
			continue
		}
		if skipLicenses && it.IsLicenseItem {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

// GetItemByName returns the item with the given name
func (db *Database) GetItemByName(name string) (item Item, err error) {
	ctx := context.Background()
	e, err := db.EntClient.Item.Query().Where(entitem.NameEQ(name), entitem.ArchivedEQ(false)).WithLicenseItem().WithPDF().Only(ctx)
	if err != nil {
		log.Error("GetItemByName (ent): ", err)
		return
	}
	if e.Disabled {
		return item, errors.New("item is disabled")
	}
	item.ID = e.ID
	item.Name = e.Name
	item.Description = e.Description
	item.Price = int(math.Round(e.Price))
	item.Image = e.Image
	item.Archived = e.Archived
	item.Disabled = e.Disabled
	item.IsLicenseItem = e.IsLicenseItem
	if e.LicenseGroup != "" {
		item.LicenseGroup = null.NewString(e.LicenseGroup, true)
	}
	item.IsPDFItem = e.IsPDFItem
	if e.Edges.PDF != nil {
		item.PDF = null.IntFrom(int64(e.Edges.PDF.ID))
	}
	item.ItemOrder = e.ItemOrder
	if e.ItemColor != "" {
		item.ItemColor = null.NewString(e.ItemColor, true)
	}
	if e.ItemTextColor != "" {
		item.ItemTextColor = null.NewString(e.ItemTextColor, true)
	}
	if e.Edges.LicenseItem != nil {
		item.LicenseItem = null.IntFrom(int64(e.Edges.LicenseItem.ID))
	}
	return
}

// GetItem returns the item with the given ID
func (db *Database) GetItem(id int) (item Item, err error) {
	ctx := context.Background()
	e, err := db.EntClient.Item.Query().Where(entitem.IDEQ(id), entitem.ArchivedEQ(false)).WithLicenseItem().WithPDF().Only(ctx)
	if err != nil {
		log.Error("GetItem (ent): ", err)
		return
	}
	if e.Disabled {
		return item, errors.New("item is disabled")
	}
	item.ID = e.ID
	item.Name = e.Name
	item.Description = e.Description
	item.Price = int(math.Round(e.Price))
	item.Image = e.Image
	item.Archived = e.Archived
	item.Disabled = e.Disabled
	item.IsLicenseItem = e.IsLicenseItem
	if e.LicenseGroup != "" {
		item.LicenseGroup = null.NewString(e.LicenseGroup, true)
	}
	item.IsPDFItem = e.IsPDFItem
	if e.Edges.PDF != nil {
		item.PDF = null.IntFrom(int64(e.Edges.PDF.ID))
	}
	item.ItemOrder = e.ItemOrder
	if e.ItemColor != "" {
		item.ItemColor = null.NewString(e.ItemColor, true)
	}
	if e.ItemTextColor != "" {
		item.ItemTextColor = null.NewString(e.ItemTextColor, true)
	}
	if e.Edges.LicenseItem != nil {
		item.LicenseItem = null.IntFrom(int64(e.Edges.LicenseItem.ID))
	}
	return
}

// GetItemTx returns the item with the given ID inside a transaction
func (db *Database) GetItemTx(tx pgx.Tx, id int) (item Item, err error) {
	// keep tx-based retrieval for transactional callers (uses pgx.Tx)
	err = tx.QueryRow(context.Background(), "SELECT ID, Name, Description, Price, Image, LicenseItem, Archived, Disabled, IsLicenseItem, LicenseGroup, IsPDFItem, PDF, ItemOrder, ItemColor, ItemTextColor FROM Item WHERE ID = $1", id).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.Disabled, &item.IsLicenseItem, &item.LicenseGroup, &item.IsPDFItem, &item.PDF, &item.ItemOrder, &item.ItemColor, &item.ItemTextColor)
	if err != nil {
		log.Error("GetItem: failed in GetItemTx() ", err)
		return
	}
	if item.Disabled {
		return item, errors.New("item is disabled")
	}
	return
}

// GetItemByLicenseID returns the item that currently references the given
// license item (via the licenseitem foreign key). If no item references the
// license, found will be false and err will be nil.
func (db *Database) GetItemByLicenseID(licenseID int) (item Item, found bool, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT ID, Name, Description, Price, Image, LicenseItem, Archived, Disabled, IsLicenseItem, LicenseGroup, IsPDFItem, PDF, ItemOrder, ItemColor, ItemTextColor FROM Item WHERE LicenseItem = $1 LIMIT 1", licenseID).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.Disabled, &item.IsLicenseItem, &item.LicenseGroup, &item.IsPDFItem, &item.PDF, &item.ItemOrder, &item.ItemColor, &item.ItemTextColor)
	if err != nil {
		if err == pgx.ErrNoRows {
			return item, false, nil
		}
		log.Error("GetItemByLicenseID: query failed", err)
		return item, false, err
	}
	// if item.Disabled {
	// 	return item, true, errors.New("item is disabled")
	// }
	return item, true, nil
}

// CreateItem creates an item in the database
func (db *Database) CreateItem(item Item) (id int, err error) {
	ctx := context.Background()
	// ensure name uniqueness
	exists, err := db.EntClient.Item.Query().Where(entitem.NameEQ(item.Name)).Exist(ctx)
	if err != nil {
		log.Error("CreateItem (ent) existence check: ", err)
		return 0, err
	}
	if exists {
		return 0, errors.New("Item with the same name already exists. Update it or delete it first")
	}

	builder := db.EntClient.Item.Create().SetName(item.Name).SetDescription(item.Description).SetPrice(float64(item.Price)).SetImage(item.Image).SetArchived(item.Archived).SetDisabled(item.Disabled).SetIsLicenseItem(item.IsLicenseItem).SetLicenseGroup(item.LicenseGroup.String).SetIsPDFItem(item.IsPDFItem).SetItemOrder(item.ItemOrder).SetItemColor(item.ItemColor.String).SetItemTextColor(item.ItemTextColor.String)
	if item.LicenseItem.Valid {
		v := int(item.LicenseItem.ValueOrZero())
		builder = builder.SetNillableLicenseItemID(&v)
	}
	if item.PDF.Valid {
		v := int(item.PDF.ValueOrZero())
		builder = builder.SetNillablePDFID(&v)
	}
	e, err := builder.Save(ctx)
	if err != nil {
		log.Error("CreateItem (ent) failed: ", err)
		return 0, err
	}
	return e.ID, nil
}

// UpdateItem updates an item in the database
func (db *Database) UpdateItem(id int, item Item) (err error) {
	log.Infof("UpdateItem: %s", item.Name)
	ctx := context.Background()
	ub := db.EntClient.Item.UpdateOneID(id).
		SetName(item.Name).
		SetDescription(item.Description).
		SetPrice(float64(item.Price)).
		SetImage(item.Image).
		SetArchived(item.Archived).
		SetDisabled(item.Disabled).
		SetIsLicenseItem(item.IsLicenseItem).
		SetLicenseGroup(item.LicenseGroup.String).
		SetIsPDFItem(item.IsPDFItem).
		SetItemOrder(item.ItemOrder).
		SetItemColor(item.ItemColor.String).
		SetItemTextColor(item.ItemTextColor.String)
	if item.LicenseItem.Valid {
		v := int(item.LicenseItem.ValueOrZero())
		// Avoid redundant license-edge update when the item already has this license assigned.
		// Fetch current item and compare its license edge; only set the edge when it actually changes.
		cur, errCur := db.EntClient.Item.Query().Where(entitem.IDEQ(id)).WithLicenseItem().Only(ctx)
		if errCur != nil {
			// Could not fetch current item, fall back to setting the license (ent will validate)
			ub = ub.SetNillableLicenseItemID(&v)
		} else {
			if cur.Edges.LicenseItem != nil && cur.Edges.LicenseItem.ID == v {
				// license already assigned to this item — skip changing the edge to avoid triggering constraint
				// No-op: do not call SetNillableLicenseItemID
			} else {
				ub = ub.SetNillableLicenseItemID(&v)
			}
		}
	} else {
		// clear the license edge when license is not provided
		ub = ub.ClearLicenseItem()
	}
	if item.PDF.Valid {
		v := int(item.PDF.ValueOrZero())
		ub = ub.SetNillablePDFID(&v)
	} else {
		ub = ub.ClearPDF()
	}
	_, err = ub.Save(ctx)
	if err != nil {
		log.Errorf("UpdateItem 1(ent): %s %+v", err, item)
	}
	return err
}

// DeleteItem archives an item in the database
func (db *Database) DeleteItem(id int) (err error) {
	ctx := context.Background()
	// Clear any assigned license on this item and mark it archived so that
	// digital licenses become unassigned when the owning item is deleted.
	_, err = db.EntClient.Item.UpdateOneID(id).ClearLicenseItem().SetArchived(true).Save(ctx)
	if err != nil {
		log.Error("DeleteItem (ent): ", err)
	}
	return
}
