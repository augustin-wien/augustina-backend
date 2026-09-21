package database

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/augustin-wien/augustina-backend/config"
	ent "github.com/augustin-wien/augustina-backend/ent"
	entitem "github.com/augustin-wien/augustina-backend/ent/item"
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
		it := convertEntItem(e)
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
		log.Error("ListItemsWithDisabled (ent): ", err)
		return out, err
	}
	for _, e := range ents {
		it := convertEntItem(e)
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

// ListLicenseGroups returns the distinct non-empty LicenseGroup values from issue and online_issue items.
func (db *Database) ListLicenseGroups() ([]string, error) {
	ctx := context.Background()
	ents, err := db.EntClient.Item.Query().
		Where(
			entitem.ArchivedEQ(false),
			entitem.TypeIn("issue", "online_issue"),
		).
		All(ctx)
	if err != nil {
		log.Error("ListLicenseGroups (ent): ", err)
		return nil, err
	}

	seen := make(map[string]struct{})
	var out []string
	for _, e := range ents {
		if e.LicenseGroup != "" && e.LicenseGroup != "default" {
			if _, ok := seen[e.LicenseGroup]; !ok {
				seen[e.LicenseGroup] = struct{}{}
				out = append(out, e.LicenseGroup)
			}
		}
	}
	return out, nil
}

func (db *Database) ListItemsShop() ([]Item, error) {
	var out []Item
	ctx := context.Background()
	ents, err := db.EntClient.Item.Query().Where(entitem.ArchivedEQ(false), entitem.DisabledEQ(false)).Order(ent.Desc(entitem.FieldItemOrder)).WithLicenseItem().WithPDF().All(ctx)
	if err != nil {
		log.Error("ListItemsShop (ent): ", err)
		return out, err
	}
	for _, e := range ents {
		it := convertEntItem(e)
		if it.Name == config.Config.TransactionCostsName || it.Name == config.Config.DonationName {
			continue
		}
		if it.IsLicenseItem {
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
	item = convertEntItem(e)
	if item.Disabled {
		return item, errors.New("item is disabled")
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
	item = convertEntItem(e)
	if item.Disabled {
		return item, errors.New("item is disabled")
	}
	return
}

// GetItemIncludingDisabled returns the item with the given ID without
// enforcing disabled checks.
func (db *Database) GetItemIncludingDisabled(id int) (item Item, err error) {
	e, err := db.EntClient.Item.Query().
		Where(entitem.IDEQ(id), entitem.ArchivedEQ(false)).
		WithLicenseItem().
		WithPDF().
		Only(context.Background())
	if err != nil {
		log.Error("GetItemIncludingDisabled (ent): ", err)
		return
	}
	item = convertEntItem(e)
	return
}

// GetItemTx returns the item with the given ID inside a transaction
func (db *Database) GetItemTx(tx *ent.Tx, id int) (item Item, err error) {
	e, err := tx.Item.Query().
		Where(entitem.ID(id)).
		WithPDF().
		WithLicenseItem().
		Only(context.Background())

	if err != nil {
		log.Error("GetItem: failed in GetItemTx() ", err)
		return
	}
	item = convertEntItem(e)
	if item.Disabled {
		return item, errors.New("item is disabled")
	}
	return
}

func convertEntItem(e *ent.Item) Item {
	var it Item
	it.ID = e.ID
	it.Name = e.Name
	it.Description = e.Description
	it.Price = int(math.Round(e.Price))
	it.Image = e.Image
	it.Archived = e.Archived
	it.Disabled = e.Disabled
	it.IsLicenseItem = e.IsLicenseItem
	it.Type = e.Type
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
	return it
}

// GetItemByLicenseID returns the item that currently references the given
// license item (via the licenseitem foreign key). If no item references the
// license, found will be false and err will be nil.
func (db *Database) GetItemByLicenseID(licenseID int) (item Item, found bool, err error) {
	e, err := db.EntClient.Item.Query().Where(entitem.HasLicenseItemWith(entitem.ID(licenseID))).First(context.Background())
	if err != nil {
		if ent.IsNotFound(err) {
			return item, false, nil
		}
		log.Error("GetItemByLicenseID: query failed", err)
		return item, false, err
	}
	item = convertEntItem(e)
	// if item.Disabled {
	// 	return item, true, errors.New("item is disabled")
	// }
	return item, true, nil
}

// GetLatestPublishedOnlineIssueByLicenseGroup returns the most recently
// published (enabled, non-archived) online_issue that carries the given
// licenseGroup. "Most recent" is determined by the highest item ID.
// Returns found=false when no matching issue exists.
func (db *Database) GetLatestPublishedOnlineIssueByLicenseGroup(licenseGroup string) (item Item, found bool, err error) {
	e, err := db.EntClient.Item.Query().
		Where(
			entitem.TypeEQ("online_issue"),
			entitem.ArchivedEQ(false),
			entitem.DisabledEQ(false),
			entitem.LicenseGroupEQ(licenseGroup),
		).
		Order(ent.Desc(entitem.FieldID)).
		First(context.Background())
	if err != nil {
		if ent.IsNotFound(err) {
			return item, false, nil
		}
		return item, false, err
	}
	item = convertEntItem(e)
	return item, true, nil
}

// CreateItem creates an item in the database
func (db *Database) CreateItem(item Item) (id int, err error) {
	ctx := context.Background()

	// An item with this name may already exist. The "name" column has carried a
	// hard UNIQUE constraint since the very first schema, so even an archived
	// (soft-deleted) row permanently occupies its name — it can never be reused
	// by a brand-new row. If the existing row is archived, resurrect it in place
	// instead of failing; only reject when an active item already owns the name.
	existing, err := db.EntClient.Item.Query().Where(entitem.NameEQ(item.Name)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error("CreateItem (ent) existence check: ", err)
		return 0, err
	}

	if err == nil {
		if !existing.Archived {
			err = errors.New("Item with the same name already exists. Update it or delete it first")
			return 0, err
		}
		ub := db.EntClient.Item.UpdateOneID(existing.ID).
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
			SetItemTextColor(item.ItemTextColor.String).
			SetType(item.Type)
		if item.LicenseItem.Valid {
			v := int(item.LicenseItem.ValueOrZero())
			ub = ub.SetNillableLicenseItemID(&v)
		} else {
			ub = ub.ClearLicenseItem()
		}
		if item.PDF.Valid {
			v := int(item.PDF.ValueOrZero())
			ub = ub.SetNillablePDFID(&v)
		} else {
			ub = ub.ClearPDF()
		}
		e, saveErr := ub.Save(ctx)
		if saveErr != nil {
			err = saveErr
			log.Error("CreateItem (ent) resurrect failed: ", err)
			return 0, err
		}
		return e.ID, nil
	}

	builder := db.EntClient.Item.Create().SetName(item.Name).SetDescription(item.Description).SetPrice(float64(item.Price)).SetImage(item.Image).SetArchived(item.Archived).SetDisabled(item.Disabled).SetIsLicenseItem(item.IsLicenseItem).SetLicenseGroup(item.LicenseGroup.String).SetIsPDFItem(item.IsPDFItem).SetItemOrder(item.ItemOrder).SetItemColor(item.ItemColor.String).SetItemTextColor(item.ItemTextColor.String).SetType(item.Type)
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

// createItemWithLicense is the shared implementation for creating an item with an
// auto-generated license_item in a single transaction. itemType must be a valid type
// such as "online_issue" or "abonement". online_issue items are created disabled=true.
func (db *Database) createItemWithLicense(item Item, licenseCost int, itemType string) (mainID int, licenseID int, err error) {
	if licenseCost <= 0 {
		return 0, 0, errors.New("license_cost must be greater than 0")
	}

	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	baseName := strings.TrimSpace(item.Name)
	if baseName == "" {
		return 0, 0, errors.New("item name is required")
	}

	licenseName := baseName + " license"
	licenseDesc := item.Description
	if strings.TrimSpace(licenseDesc) == "" {
		licenseDesc = "License for " + baseName
	}

	licenseItem := Item{
		Name:          licenseName,
		Description:   licenseDesc,
		Price:         licenseCost,
		Image:         item.Image,
		Archived:      false,
		Disabled:      false,
		IsLicenseItem: true,
		IsPDFItem:     false,
		ItemOrder:     item.ItemOrder,
		LicenseGroup:  item.LicenseGroup,
		Type:          "license_item",
	}

	// An item with this name may already exist as a leftover, unassigned
	// license_item from a previously deleted issue/abonement of the same name
	// (deleting an item unassigns its license but does not remove it, so that
	// a license can be reassigned later). Reuse it instead of failing outright;
	// only reject when the name belongs to something else or is still in use.
	existingLicense, err := tx.Item.Query().Where(entitem.NameEQ(licenseItem.Name)).Only(context.Background())
	if err != nil && !ent.IsNotFound(err) {
		return 0, 0, err
	}

	var licenseEnt *ent.Item
	if err == nil {
		if !existingLicense.IsLicenseItem {
			err = errors.New("Item with the same name already exists. Update it or delete it first")
			return 0, 0, err
		}
		var hasOwner bool
		hasOwner, err = tx.Item.Query().Where(entitem.HasLicenseItemWith(entitem.ID(existingLicense.ID))).Exist(context.Background())
		if err != nil {
			return 0, 0, err
		}
		if hasOwner {
			err = errors.New("Item with the same name already exists. Update it or delete it first")
			return 0, 0, err
		}
		licenseEnt, err = tx.Item.UpdateOneID(existingLicense.ID).
			SetDescription(licenseItem.Description).
			SetPrice(float64(licenseItem.Price)).
			SetImage(licenseItem.Image).
			SetArchived(licenseItem.Archived).
			SetDisabled(licenseItem.Disabled).
			SetLicenseGroup(licenseItem.LicenseGroup.String).
			SetIsPDFItem(licenseItem.IsPDFItem).
			SetItemOrder(licenseItem.ItemOrder).
			SetItemColor(licenseItem.ItemColor.String).
			SetItemTextColor(licenseItem.ItemTextColor.String).
			Save(context.Background())
		if err != nil {
			return 0, 0, err
		}
	} else {
		err = nil
		licenseEnt, err = tx.Item.Create().
			SetName(licenseItem.Name).
			SetDescription(licenseItem.Description).
			SetPrice(float64(licenseItem.Price)).
			SetImage(licenseItem.Image).
			SetArchived(licenseItem.Archived).
			SetDisabled(licenseItem.Disabled).
			SetIsLicenseItem(licenseItem.IsLicenseItem).
			SetLicenseGroup(licenseItem.LicenseGroup.String).
			SetIsPDFItem(licenseItem.IsPDFItem).
			SetItemOrder(licenseItem.ItemOrder).
			SetItemColor(licenseItem.ItemColor.String).
			SetItemTextColor(licenseItem.ItemTextColor.String).
			SetType(licenseItem.Type).
			Save(context.Background())
		if err != nil {
			return 0, 0, err
		}
	}

	item.Type = itemType
	// both online_issue and abonement items start disabled and are enabled when ready
	item.Disabled = true
	item.IsLicenseItem = false
	item.LicenseItem = null.IntFrom(int64(licenseEnt.ID))

	// Same reasoning as the license lookup above: the "name" column is globally
	// UNIQUE, so an archived leftover main item (from a previously deleted issue
	// of the same name) must be resurrected in place rather than blocking creation.
	existingMain, err := tx.Item.Query().Where(entitem.NameEQ(item.Name)).Only(context.Background())
	if err != nil && !ent.IsNotFound(err) {
		return 0, 0, err
	}

	var mainEnt *ent.Item
	if err == nil {
		if !existingMain.Archived {
			err = errors.New("Item with the same name already exists. Update it or delete it first")
			return 0, 0, err
		}
		mub := tx.Item.UpdateOneID(existingMain.ID).
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
			SetItemTextColor(item.ItemTextColor.String).
			SetType(item.Type).
			SetNillableLicenseItemID(&licenseEnt.ID)
		if item.PDF.Valid {
			v := int(item.PDF.ValueOrZero())
			mub = mub.SetNillablePDFID(&v)
		} else {
			mub = mub.ClearPDF()
		}
		mainEnt, err = mub.Save(context.Background())
		if err != nil {
			return 0, 0, err
		}
	} else {
		err = nil
		mainBuilder := tx.Item.Create().
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
			SetItemTextColor(item.ItemTextColor.String).
			SetType(item.Type)

		if item.LicenseItem.Valid {
			v := int(item.LicenseItem.ValueOrZero())
			mainBuilder = mainBuilder.SetNillableLicenseItemID(&v)
		}
		if item.PDF.Valid {
			v := int(item.PDF.ValueOrZero())
			mainBuilder = mainBuilder.SetNillablePDFID(&v)
		}

		mainEnt, err = mainBuilder.Save(context.Background())
		if err != nil {
			return 0, 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}

	return mainEnt.ID, licenseEnt.ID, nil
}

// CreateOnlineIssueWithLicense creates an online_issue item and its required
// license_item in one transaction. The returned IDs are (onlineIssueID, licenseID).
func (db *Database) CreateOnlineIssueWithLicense(item Item, licenseCost int) (onlineIssueID int, licenseID int, err error) {
	return db.createItemWithLicense(item, licenseCost, "online_issue")
}

// CreateAbonementItemWithLicense creates an abonement item and its required
// license_item in one transaction. The returned IDs are (abonementID, licenseID).
func (db *Database) CreateAbonementItemWithLicense(item Item, licenseCost int) (abonementID int, licenseID int, err error) {
	return db.createItemWithLicense(item, licenseCost, "abonement")
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
		SetItemTextColor(item.ItemTextColor.String).
		SetType(item.Type)
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
