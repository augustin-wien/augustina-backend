package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestVendorCommentsCRUD(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	vendorID := createTestVendor(t, "testlicense-comments")
	base := "/api/vendors/" + vendorID + "/comments/"

	listComments := func() []ent.Comment {
		res := utils.TestRequestWithAuth(t, r, "GET", base, nil, 200, adminUserToken)
		var comments []ent.Comment
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &comments))
		return comments
	}

	require.Empty(t, listComments())

	utils.TestRequestWithAuth(t, r, "POST", base, map[string]any{
		"comment":    "Ausweis abgelaufen",
		"warning":    true,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}, 200, adminUserToken)

	comments := listComments()
	require.Len(t, comments, 1)
	require.Equal(t, "Ausweis abgelaufen", comments[0].Comment)
	require.True(t, comments[0].Warning)

	commentID := strconv.Itoa(comments[0].ID)
	utils.TestRequestWithAuth(t, r, "PATCH", base+commentID+"/", map[string]any{
		"id":          comments[0].ID,
		"comment":     "Ausweis erneuert",
		"warning":     false,
		"resolved_at": time.Now().UTC().Format(time.RFC3339),
	}, 200, adminUserToken)

	comments = listComments()
	require.Len(t, comments, 1)
	require.Equal(t, "Ausweis erneuert", comments[0].Comment)
	require.False(t, comments[0].Warning)
	require.False(t, comments[0].ResolvedAt.IsZero())

	utils.TestRequestWithAuth(t, r, "DELETE", base+commentID+"/", nil, 200, adminUserToken)
	require.Empty(t, listComments())

	// Invalid IDs and bodies
	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/abc/comments/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/abc/comments/", map[string]any{}, 400, adminUserToken)
	utils.TestRequestStrWithAuth(t, r, "POST", base, "{not json", 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "PATCH", "/api/vendors/abc/comments/1/", map[string]any{}, 400, adminUserToken)
	utils.TestRequestStrWithAuth(t, r, "PATCH", base+"1/", "{not json", 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "PATCH", base+"999999/", map[string]any{"id": 999999, "comment": "x"}, 500, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/abc/comments/1/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", base+"abc/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", base+"999999/", nil, 500, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/999999/comments/", map[string]any{"comment": "x"}, 500, adminUserToken)
}

func TestMailTemplatesCRUD(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	utils.TestRequestWithAuth(t, r, "POST", "/api/mail-templates/", map[string]any{
		"name":    "zz-test-template",
		"subject": "Hallo",
		"body":    "<p>Body</p>",
	}, 200, adminUserToken)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/mail-templates/zz-test-template/", nil, 200, adminUserToken)
	var mt database.MailTemplate
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &mt))
	require.Equal(t, "Hallo", mt.Subject)
	require.Equal(t, "<p>Body</p>", mt.Body)

	// Posting the same name again updates the template
	utils.TestRequestWithAuth(t, r, "POST", "/api/mail-templates/", map[string]any{
		"name":    "zz-test-template",
		"subject": "Servus",
		"body":    "<p>Neu</p>",
	}, 200, adminUserToken)

	res = utils.TestRequestWithAuth(t, r, "GET", "/api/mail-templates/", nil, 200, adminUserToken)
	var list []database.MailTemplate
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &list))
	var found *database.MailTemplate
	for i := range list {
		if list[i].Name == "zz-test-template" {
			found = &list[i]
		}
	}
	require.NotNil(t, found)
	require.Equal(t, "Servus", found.Subject)
	require.Equal(t, "<p>Neu</p>", found.Body)

	utils.TestRequestWithAuth(t, r, "DELETE", "/api/mail-templates/zz-test-template/", nil, 200, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/mail-templates/zz-test-template/", nil, 400, adminUserToken)

	// Validation
	utils.TestRequestWithAuth(t, r, "POST", "/api/mail-templates/", map[string]any{"subject": "no name"}, 400, adminUserToken)
	utils.TestRequestStrWithAuth(t, r, "POST", "/api/mail-templates/", "{not json", 400, adminUserToken)
}

func TestItemArchiveRestoreAndLicenseGroups(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	licenseID, err := database.Db.CreateItem(database.Item{
		Name:          "Digital License",
		Description:   "Test item",
		Price:         100,
		Type:          "license_item",
		IsLicenseItem: true,
		LicenseGroup:  null.StringFrom("test_digital_group"),
	})
	require.NoError(t, err)
	issueID, err := database.Db.CreateItem(database.Item{
		Name:         "Online Issue 1",
		Description:  "Test item",
		Price:        300,
		Type:         "online_issue",
		LicenseGroup: null.StringFrom("test_digital_group"),
		LicenseItem:  null.IntFrom(int64(licenseID)),
	})
	require.NoError(t, err)
	_, err = database.Db.CreateItem(database.Item{
		Name:         "Print Issue 1",
		Description:  "Test item",
		Price:        300,
		Type:         "issue",
		LicenseGroup: null.StringFrom("test_analog_group"),
	})
	require.NoError(t, err)
	_, err = database.Db.CreateItem(database.Item{
		Name:         "Print Issue 2",
		Description:  "Test item",
		Price:        300,
		Type:         "issue",
		LicenseGroup: null.StringFrom("test_analog_group"),
	})
	require.NoError(t, err)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/items/licensegroups/", nil, 200, adminUserToken)
	var groups []string
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &groups))
	require.Contains(t, groups, "test_digital_group")
	require.Contains(t, groups, "test_analog_group")
	require.Len(t, groups, len(uniqueStrings(groups)))

	// A license item still linked to an active issue cannot be deleted
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/items/"+strconv.Itoa(licenseID)+"/", nil, 409, adminUserToken)

	// Deleting the issue together with its license archives both
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/items/"+strconv.Itoa(issueID)+"/?deleteLicense=true", nil, 204, adminUserToken)

	res = utils.TestRequestWithAuth(t, r, "GET", "/api/items/archived/", nil, 200, adminUserToken)
	var archived []database.Item
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &archived))
	archivedIDs := []int{}
	for _, it := range archived {
		archivedIDs = append(archivedIDs, it.ID)
	}
	require.ElementsMatch(t, []int{issueID, licenseID}, archivedIDs)

	res = utils.TestRequestWithAuth(t, r, "GET", "/api/items/licensegroups/", nil, 200, adminUserToken)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &groups))
	require.Contains(t, groups, "test_analog_group")
	require.NotContains(t, groups, "test_digital_group")

	// Archived items are hidden from the backoffice list unless requested
	backofficeIDs := func(query string) []int {
		res := utils.TestRequestWithAuth(t, r, "GET", "/api/items/backoffice/"+query, nil, 200, adminUserToken)
		var items []database.Item
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &items))
		ids := []int{}
		for _, it := range items {
			ids = append(ids, it.ID)
		}
		return ids
	}
	require.NotContains(t, backofficeIDs(""), issueID)
	require.Contains(t, backofficeIDs("?includeArchived=true"), issueID)
	require.NotContains(t, backofficeIDs("?includeArchived=true&skipLicenses=true"), licenseID)
	require.NotContains(t, backofficeIDs("?skipHiddenItems=true"), issueID)
	utils.TestRequestWithAuth(t, r, "GET", "/api/items/backoffice/?skipHiddenItems=nope", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/items/backoffice/?skipLicenses=nope", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/items/backoffice/?includeArchived=nope", nil, 400, adminUserToken)

	utils.TestRequestWithAuth(t, r, "POST", "/api/items/"+strconv.Itoa(issueID)+"/restore/", nil, 204, adminUserToken)
	require.Contains(t, backofficeIDs(""), issueID)

	utils.TestRequestWithAuth(t, r, "DELETE", "/api/items/abc/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/items/999999/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/items/abc/restore/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/items/999999/restore/", nil, 400, adminUserToken)
}

func uniqueStrings(in []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, s := range in {
		out[s] = struct{}{}
	}
	return out
}

func TestPOSOrderListings(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	licenseID := "testlicense-pos-list"
	createTestVendor(t, licenseID)

	itemID, err := database.Db.CreateItem(database.Item{
		Name:        "POS Newspaper",
		Description: "Test item",
		Price:       300,
		Type:        "normal_item",
	})
	require.NoError(t, err)

	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+licenseID+"/pos-order/", map[string]any{
		"entries": []map[string]any{{"item": itemID, "quantity": 2}},
	}, 200, adminUserToken)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+licenseID+"/pos-orders/", nil, 200, adminUserToken)
	var orders []database.POSOrder
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &orders))
	require.Len(t, orders, 1)
	require.Equal(t, 600, orders[0].TotalAmount)
	require.Equal(t, 600, orders[0].CashAmount)
	require.Equal(t, 0, orders[0].BalanceUsed)
	require.Len(t, orders[0].Items, 1)
	require.Equal(t, "POS Newspaper", orders[0].Items[0].ItemName)
	require.Equal(t, 2, orders[0].Items[0].Quantity)

	res = utils.TestRequestWithAuth(t, r, "GET", "/api/pos-orders/", nil, 200, adminUserToken)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &orders))
	require.Len(t, orders, 1)
	require.Equal(t, licenseID, orders[0].VendorLicenseID)
	require.Equal(t, 600, orders[0].TotalAmount)

	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	past := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/pos-orders/?from="+future, nil, 200, adminUserToken)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &orders))
	require.Empty(t, orders)
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/pos-orders/?from="+past+"&to="+future, nil, 200, adminUserToken)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &orders))
	require.Len(t, orders, 1)

	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/unknown-license/pos-orders/", nil, 500, adminUserToken)

	// Validation of the POS order itself
	posURL := "/api/vendors/" + licenseID + "/pos-order/"
	utils.TestRequestWithAuth(t, r, "POST", posURL, map[string]any{"entries": []map[string]any{}}, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", posURL, map[string]any{"entries": []map[string]any{{"item": itemID, "quantity": 0}}}, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", posURL, map[string]any{"entries": []map[string]any{{"item": 999999, "quantity": 1}}}, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/unknown-license/pos-order/", map[string]any{"entries": []map[string]any{{"item": itemID, "quantity": 1}}}, 400, adminUserToken)
	utils.TestRequestStrWithAuth(t, r, "POST", posURL, "{not json", 400, adminUserToken)

	abonementItem, err := database.Db.CreateItem(database.Item{Name: "POS Abo", Description: "Test item", Price: 1000, Type: "abonement"})
	require.NoError(t, err)
	utils.TestRequestWithAuth(t, r, "POST", posURL, map[string]any{"entries": []map[string]any{{"item": abonementItem, "quantity": 1}}}, 400, adminUserToken)
}

func TestAbonementListings(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	customer, err := database.Db.CreateCustomer(&database.Customer{
		KeycloakID: "kc-abo-list",
		Email:      "abo-list@example.com",
		FirstName:  "Abo",
		LastName:   "List",
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	active, err := database.Db.CreateAbonement(&database.Abonement{
		CustomerID: customer.ID,
		FromDate:   now.AddDate(0, -1, 0),
		ToDate:     now.AddDate(0, 1, 0),
		Status:     "active",
	})
	require.NoError(t, err)
	_, err = database.Db.CreateAbonement(&database.Abonement{
		CustomerID: customer.ID,
		FromDate:   now.AddDate(-2, 0, 0),
		ToDate:     now.AddDate(-1, 0, 0),
		Status:     "active",
	})
	require.NoError(t, err)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/abonements/", nil, 200, adminUserToken)
	var all []database.Abonement
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &all))
	require.Len(t, all, 2)

	res = utils.TestRequestWithAuth(t, r, "GET", "/api/abonements/by-date/?date="+now.Format("2006-01-02"), nil, 200, adminUserToken)
	var byDate []database.Abonement
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &byDate))
	require.Len(t, byDate, 1)
	require.Equal(t, active.ID, byDate[0].ID)

	utils.TestRequestWithAuth(t, r, "GET", "/api/abonements/by-date/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/abonements/by-date/?date=23.11.2026", nil, 400, adminUserToken)
}

func TestPDFDownloadLink(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	pdfPath := filepath.Join(t.TempDir(), "issue.pdf")
	pdfContent := []byte("%PDF-1.4 test content")
	require.NoError(t, os.WriteFile(pdfPath, pdfContent, 0o600))

	pdfID, err := database.Db.CreatePDF(database.PDF{Path: pdfPath, Timestamp: time.Now()})
	require.NoError(t, err)
	itemID, err := database.Db.CreateItem(database.Item{
		Name:        "PDF Issue",
		Description: "Test item",
		Price:       300,
		Type:        "issue",
		IsPDFItem:   true,
		PDF:         null.IntFrom(pdfID),
	})
	require.NoError(t, err)

	download, err := database.Db.CreatePDFDownloadForItem(int(pdfID), itemID)
	require.NoError(t, err)

	res := utils.TestRequest(t, r, "GET", "/api/pdf/"+download.LinkID+"/validate/", nil, 200)
	require.Equal(t, "\"valid\"", res.Body.String())

	res = utils.TestRequest(t, r, "GET", "/api/pdf/"+download.LinkID+"/", nil, 200)
	require.Equal(t, pdfContent, res.Body.Bytes())

	updated, err := database.Db.GetPDFDownload(download.LinkID)
	require.NoError(t, err)
	require.Equal(t, 1, updated.DownloadCount)
	require.False(t, updated.LastDownload.IsZero())

	utils.TestRequest(t, r, "GET", "/api/pdf/unknown-link/validate/", nil, 400)
	utils.TestRequest(t, r, "GET", "/api/pdf/unknown-link/", nil, 400)

	// Links older than six weeks are expired
	_, err = database.Db.EntClient.PDFDownload.UpdateOneID(download.ID).
		SetTimestamp(time.Now().AddDate(0, 0, -50)).
		Save(t.Context())
	require.NoError(t, err)
	utils.TestRequest(t, r, "GET", "/api/pdf/"+download.LinkID+"/validate/", nil, 400)
	utils.TestRequest(t, r, "GET", "/api/pdf/"+download.LinkID+"/", nil, 400)
}

func TestCreatePaymentEndpoints(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	licenseID := "testlicense-payments"
	vendorID, err := strconv.Atoi(createTestVendor(t, licenseID))
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	cashAccount, err := database.Db.GetAccountByType("Cash")
	require.NoError(t, err)

	res := utils.TestRequestWithAuth(t, r, "POST", "/api/payments/", database.Payment{
		Sender:   cashAccount.ID,
		Receiver: vendorAccount.ID,
		Amount:   250,
		Quantity: 1,
		Price:    250,
	}, 200, adminUserToken)
	paymentID, err := strconv.Atoi(res.Body.String())
	require.NoError(t, err)
	require.Positive(t, paymentID)

	utils.TestRequestWithAuth(t, r, "POST", "/api/payments/batch/", map[string]any{
		"Payments": []database.Payment{
			{Sender: cashAccount.ID, Receiver: vendorAccount.ID, Amount: 100, Quantity: 1, Price: 100},
			{Sender: cashAccount.ID, Receiver: vendorAccount.ID, Amount: 50, Quantity: 1, Price: 50},
		},
	}, 200, adminUserToken)

	vendorAccount, err = database.Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	require.Equal(t, 400, vendorAccount.Balance)

	// Recalculating the balances from the payments keeps the same result
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/recalculate-balances/", nil, 200, adminUserToken)
	vendorAccount, err = database.Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	require.Equal(t, 400, vendorAccount.Balance)

	utils.TestRequestStrWithAuth(t, r, "POST", "/api/payments/", "{not json", 400, adminUserToken)
	utils.TestRequestStrWithAuth(t, r, "POST", "/api/payments/batch/", "{not json", 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/payments/", database.Payment{Sender: 999999, Receiver: vendorAccount.ID, Amount: 1}, 400, adminUserToken)
}

func TestListVerifiedAndUnsyncedOrders(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/orders/verified/", nil, 200, adminUserToken)
	var orders []database.Order
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &orders))
	require.Empty(t, orders)

	from := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	to := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	utils.TestRequestWithAuth(t, r, "GET", "/api/orders/verified/?from="+from+"&to="+to, nil, 200, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/orders/verified/?from=yesterday", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/orders/verified/?to=tomorrow", nil, 400, adminUserToken)

	utils.TestRequestWithAuth(t, r, "GET", "/api/orders/unsynced/", nil, 200, adminUserToken)
}
