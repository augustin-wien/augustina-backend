package database

import (
	"context"
	"errors"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entcampaign "github.com/augustin-wien/augustina-backend/ent/campaign"
	"github.com/augustin-wien/augustina-backend/ent/predicate"
)

// Campaigns ------------------------------------------------------------------

// activeCampaign matches campaigns that are switched on and whose period includes now.
func activeCampaign(now time.Time) predicate.Campaign {
	return entcampaign.And(
		entcampaign.Enabled(true),
		entcampaign.Or(entcampaign.StartsAtIsNil(), entcampaign.StartsAtLTE(now)),
		entcampaign.Or(entcampaign.EndsAtIsNil(), entcampaign.EndsAtGT(now)),
	)
}

func validateCampaign(c *ent.Campaign) error {
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.ItemID <= 0 {
		return errors.New("item_id is required")
	}
	if c.StartsAt != nil && c.EndsAt != nil && !c.EndsAt.After(*c.StartsAt) {
		return errors.New("ends_at must be after starts_at")
	}
	return nil
}

// ListCampaigns returns all campaigns, newest first
func (db *Database) ListCampaigns() ([]*ent.Campaign, error) {
	campaigns, err := db.EntClient.Campaign.Query().
		Order(ent.Desc(entcampaign.FieldID)).
		All(context.Background())
	if err != nil {
		log.Error("ListCampaigns: ", err)
	}
	return campaigns, err
}

// ListActiveCampaigns returns the campaigns the shop should advertise right now, newest first
func (db *Database) ListActiveCampaigns(now time.Time) ([]*ent.Campaign, error) {
	campaigns, err := db.EntClient.Campaign.Query().
		Where(activeCampaign(now)).
		Order(ent.Desc(entcampaign.FieldID)).
		All(context.Background())
	if err != nil {
		log.Error("ListActiveCampaigns: ", err)
	}
	return campaigns, err
}

// CreateCampaign creates a campaign. The counters always start at zero.
func (db *Database) CreateCampaign(c *ent.Campaign) (*ent.Campaign, error) {
	if err := validateCampaign(c); err != nil {
		return nil, err
	}
	created, err := db.EntClient.Campaign.Create().
		SetName(c.Name).
		SetItemID(c.ItemID).
		SetTitle(c.Title).
		SetText(c.Text).
		SetNillableStartsAt(c.StartsAt).
		SetNillableEndsAt(c.EndsAt).
		SetEnabled(c.Enabled).
		Save(context.Background())
	if err != nil {
		log.Error("CreateCampaign: ", err)
	}
	return created, err
}

// UpdateCampaign updates the editable fields of a campaign. The counters are left untouched.
func (db *Database) UpdateCampaign(id int, c *ent.Campaign) (*ent.Campaign, error) {
	if err := validateCampaign(c); err != nil {
		return nil, err
	}
	update := db.EntClient.Campaign.UpdateOneID(id).
		SetName(c.Name).
		SetItemID(c.ItemID).
		SetTitle(c.Title).
		SetText(c.Text).
		SetEnabled(c.Enabled)
	if c.StartsAt != nil {
		update.SetStartsAt(*c.StartsAt)
	} else {
		update.ClearStartsAt()
	}
	if c.EndsAt != nil {
		update.SetEndsAt(*c.EndsAt)
	} else {
		update.ClearEndsAt()
	}
	updated, err := update.Save(context.Background())
	if err != nil {
		log.Error("UpdateCampaign: ", err)
	}
	return updated, err
}

// DeleteCampaign deletes a campaign
func (db *Database) DeleteCampaign(id int) error {
	err := db.EntClient.Campaign.DeleteOneID(id).Exec(context.Background())
	if err != nil {
		log.Error("DeleteCampaign: ", err)
	}
	return err
}

// CampaignCounter selects which counter TrackCampaign increments
type CampaignCounter string

const (
	CampaignView  CampaignCounter = "view"
	CampaignClick CampaignCounter = "click"
)

// TrackCampaign increments a counter of a campaign. Only active campaigns count, so a stale
// shop tab can't keep adding to a campaign that has been switched off. Reports whether a
// campaign was counted.
func (db *Database) TrackCampaign(id int, counter CampaignCounter, now time.Time) (bool, error) {
	update := db.EntClient.Campaign.Update().
		Where(entcampaign.ID(id), activeCampaign(now))
	switch counter {
	case CampaignView:
		update.AddViews(1)
	case CampaignClick:
		update.AddClicks(1)
	default:
		return false, errors.New("unknown counter")
	}
	n, err := update.Save(context.Background())
	if err != nil {
		log.Error("TrackCampaign: ", err)
		return false, err
	}
	return n > 0, nil
}
