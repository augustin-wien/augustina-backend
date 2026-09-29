package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func TestCampaigns(t *testing.T) {
	utils.CheckError(t, database.Db.InitEmptyTestDb())
	itemID, err := strconv.Atoi(CreateTestItem(t, "Campaign item", 800, "", ""))
	utils.CheckError(t, err)

	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)

	create := func(in CampaignInput, status int) Campaign {
		res := utils.TestRequestWithAuth(t, r, "POST", "/api/campaigns/", in, status, adminUserToken)
		var c Campaign
		if status == 200 {
			utils.CheckError(t, json.Unmarshal(res.Body.Bytes(), &c))
		}
		return c
	}
	active := func() []PublicCampaign {
		res := utils.TestRequest(t, r, "GET", "/api/campaigns/active/", nil, 200)
		var list []PublicCampaign
		utils.CheckError(t, json.Unmarshal(res.Body.Bytes(), &list))
		return list
	}

	// Validation
	create(CampaignInput{ItemID: itemID}, 400)
	create(CampaignInput{Name: "no item"}, 400)
	create(CampaignInput{Name: "bad period", ItemID: itemID, StartsAt: &future, EndsAt: &past}, 400)

	running := create(CampaignInput{Name: "running", ItemID: itemID, Title: "Neu!", Text: "Text", StartsAt: &past, EndsAt: &future, Enabled: true}, 200)
	create(CampaignInput{Name: "switched off", ItemID: itemID, Enabled: true}, 200)
	upcoming := create(CampaignInput{Name: "upcoming", ItemID: itemID, StartsAt: &future, Enabled: true}, 200)

	// Switch the second one off again
	list := []Campaign{}
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/campaigns/", nil, 200, adminUserToken)
	utils.CheckError(t, json.Unmarshal(res.Body.Bytes(), &list))
	require.Len(t, list, 3)
	off := list[1]
	require.Equal(t, "switched off", off.Name)
	off.Enabled = false
	utils.TestRequestWithAuth(t, r, "PUT", fmt.Sprintf("/api/campaigns/%d/", off.ID), off.CampaignInput, 200, adminUserToken)

	got := active()
	require.Len(t, got, 1)
	require.Equal(t, running.ID, got[0].ID)
	require.Equal(t, "Neu!", got[0].Title)

	// Counters: only active campaigns count
	utils.TestRequest(t, r, "POST", fmt.Sprintf("/api/campaigns/%d/view/", running.ID), nil, 200)
	utils.TestRequest(t, r, "POST", fmt.Sprintf("/api/campaigns/%d/view/", running.ID), nil, 200)
	utils.TestRequest(t, r, "POST", fmt.Sprintf("/api/campaigns/%d/click/", running.ID), nil, 200)
	utils.TestRequest(t, r, "POST", fmt.Sprintf("/api/campaigns/%d/view/", off.ID), nil, 404)
	utils.TestRequest(t, r, "POST", fmt.Sprintf("/api/campaigns/%d/click/", upcoming.ID), nil, 404)

	res = utils.TestRequestWithAuth(t, r, "GET", "/api/campaigns/", nil, 200, adminUserToken)
	utils.CheckError(t, json.Unmarshal(res.Body.Bytes(), &list))
	byID := map[int]Campaign{}
	for _, c := range list {
		byID[c.ID] = c
	}
	require.Equal(t, 2, byID[running.ID].Views)
	require.Equal(t, 1, byID[running.ID].Clicks)
	require.Equal(t, 0, byID[off.ID].Views)

	// Updating keeps the counters
	in := byID[running.ID].CampaignInput
	in.Title = "Anders"
	res = utils.TestRequestWithAuth(t, r, "PUT", fmt.Sprintf("/api/campaigns/%d/", running.ID), in, 200, adminUserToken)
	var updated Campaign
	utils.CheckError(t, json.Unmarshal(res.Body.Bytes(), &updated))
	require.Equal(t, "Anders", updated.Title)
	require.Equal(t, 2, updated.Views)
	require.Equal(t, 1, updated.Clicks)

	// Admin routes need auth
	utils.TestRequest(t, r, "GET", "/api/campaigns/", nil, 401)

	// Delete
	utils.TestRequestWithAuth(t, r, "DELETE", fmt.Sprintf("/api/campaigns/%d/", running.ID), nil, 200, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", fmt.Sprintf("/api/campaigns/%d/", running.ID), nil, 404, adminUserToken)
	require.Len(t, active(), 0)
}
