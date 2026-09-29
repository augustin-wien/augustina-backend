package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

// CampaignInput is what the backoffice sends when creating or updating a campaign
type CampaignInput struct {
	Name     string     `json:"name"`
	ItemID   int        `json:"item_id"`
	Title    string     `json:"title"`
	Text     string     `json:"text"`
	StartsAt *time.Time `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
	Enabled  bool       `json:"enabled"`
}

// Campaign is the backoffice view of a campaign including its counters.
// ent's own struct tags use omitempty, which would drop enabled=false and zero counters.
type Campaign struct {
	CampaignInput
	ID        int       `json:"id"`
	Views     int       `json:"views"`
	Clicks    int       `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}

// PublicCampaign is what the shop gets: no internal name, no counters
type PublicCampaign struct {
	ID       int        `json:"id"`
	ItemID   int        `json:"item_id"`
	Title    string     `json:"title"`
	Text     string     `json:"text"`
	StartsAt *time.Time `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
}

func toCampaign(c *ent.Campaign) Campaign {
	return Campaign{
		CampaignInput: CampaignInput{
			Name:     c.Name,
			ItemID:   c.ItemID,
			Title:    c.Title,
			Text:     c.Text,
			StartsAt: c.StartsAt,
			EndsAt:   c.EndsAt,
			Enabled:  c.Enabled,
		},
		ID:        c.ID,
		Views:     c.Views,
		Clicks:    c.Clicks,
		CreatedAt: c.CreatedAt,
	}
}

func (in CampaignInput) toEnt() *ent.Campaign {
	return &ent.Campaign{
		Name:     in.Name,
		ItemID:   in.ItemID,
		Title:    in.Title,
		Text:     in.Text,
		StartsAt: in.StartsAt,
		EndsAt:   in.EndsAt,
		Enabled:  in.Enabled,
	}
}

func campaignIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, errors.New("invalid campaign id"), http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// ListCampaigns godoc
//
//	@Summary		List campaigns
//	@Description	List all campaigns including their view and click counters
//	@Tags			Campaigns
//	@Produce		json
//	@Success		200	{array}	Campaign
//	@Security		KeycloakAuth
//	@Router			/campaigns/ [get]
func ListCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := database.Db.ListCampaigns()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	res := make([]Campaign, 0, len(campaigns))
	for _, c := range campaigns {
		res = append(res, toCampaign(c))
	}
	respond(w, nil, res)
}

// ListActiveCampaigns godoc
//
//	@Summary		List active campaigns
//	@Description	List the campaigns the shop should advertise right now, newest first
//	@Tags			Campaigns
//	@Produce		json
//	@Success		200	{array}	PublicCampaign
//	@Router			/campaigns/active/ [get]
func ListActiveCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := database.Db.ListActiveCampaigns(time.Now())
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	res := make([]PublicCampaign, 0, len(campaigns))
	for _, c := range campaigns {
		res = append(res, PublicCampaign{
			ID:       c.ID,
			ItemID:   c.ItemID,
			Title:    c.Title,
			Text:     c.Text,
			StartsAt: c.StartsAt,
			EndsAt:   c.EndsAt,
		})
	}
	respond(w, nil, res)
}

// CreateCampaign godoc
//
//	@Summary		Create campaign
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			data	body		CampaignInput	true	"Campaign"
//	@Success		200		{object}	Campaign
//	@Security		KeycloakAuth
//	@Router			/campaigns/ [post]
func CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var in CampaignInput
	if err := utils.ReadJSON(w, r, &in); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	c, err := database.Db.CreateCampaign(in.toEnt())
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, nil, toCampaign(c))
}

// UpdateCampaign godoc
//
//	@Summary		Update campaign
//	@Description	Update a campaign. The view and click counters are not changed.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Campaign ID"
//	@Param			data	body		CampaignInput	true	"Campaign"
//	@Success		200		{object}	Campaign
//	@Security		KeycloakAuth
//	@Router			/campaigns/{id}/ [put]
func UpdateCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := campaignIDParam(w, r)
	if !ok {
		return
	}
	var in CampaignInput
	if err := utils.ReadJSON(w, r, &in); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	c, err := database.Db.UpdateCampaign(id, in.toEnt())
	if ent.IsNotFound(err) {
		utils.ErrorJSON(w, errors.New("campaign not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, nil, toCampaign(c))
}

// DeleteCampaign godoc
//
//	@Summary		Delete campaign
//	@Tags			Campaigns
//	@Param			id	path	int	true	"Campaign ID"
//	@Success		200
//	@Security		KeycloakAuth
//	@Router			/campaigns/{id}/ [delete]
func DeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := campaignIDParam(w, r)
	if !ok {
		return
	}
	err := database.Db.DeleteCampaign(id)
	if ent.IsNotFound(err) {
		utils.ErrorJSON(w, errors.New("campaign not found"), http.StatusNotFound)
		return
	}
	respond(w, err, nil)
}

func trackCampaign(counter database.CampaignCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := campaignIDParam(w, r)
		if !ok {
			return
		}
		counted, err := database.Db.TrackCampaign(id, counter, time.Now())
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusInternalServerError)
			return
		}
		if !counted {
			utils.ErrorJSON(w, errors.New("campaign not active"), http.StatusNotFound)
			return
		}
		respond(w, nil, nil)
	}
}

// TrackCampaignView godoc
//
//	@Summary		Count a campaign view
//	@Description	Called by the shop when it shows the campaign popup. Only active campaigns are counted.
//	@Tags			Campaigns
//	@Param			id	path	int	true	"Campaign ID"
//	@Success		200
//	@Router			/campaigns/{id}/view/ [post]
var TrackCampaignView = trackCampaign(database.CampaignView)

// TrackCampaignClick godoc
//
//	@Summary		Count a campaign click
//	@Description	Called by the shop when a customer chooses the advertised item. Only active campaigns are counted.
//	@Tags			Campaigns
//	@Param			id	path	int	true	"Campaign ID"
//	@Success		200
//	@Router			/campaigns/{id}/click/ [post]
var TrackCampaignClick = trackCampaign(database.CampaignClick)
