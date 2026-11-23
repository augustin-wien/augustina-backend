package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestGetOrCreateCustomerByEmail(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	// A new customer is created
	created, err := Db.GetOrCreateCustomerByEmail("lookup@example.com", "")
	require.NoError(t, err)
	require.Empty(t, created.KeycloakID)

	// A later Keycloak account is backfilled on the existing record
	backfilled, err := Db.GetOrCreateCustomerByEmail("lookup@example.com", "kc-lookup")
	require.NoError(t, err)
	require.Equal(t, created.ID, backfilled.ID)
	require.Equal(t, "kc-lookup", backfilled.KeycloakID)

	// Found by Keycloak ID, even with a changed email
	byKeycloak, err := Db.GetOrCreateCustomerByEmail("changed@example.com", "kc-lookup")
	require.NoError(t, err)
	require.Equal(t, created.ID, byKeycloak.ID)

	// A failing backfill still returns the record
	_, err = Db.GetOrCreateCustomerByEmail("other@example.com", "")
	require.NoError(t, err)
	kept, err := failingMutations("Customer").GetOrCreateCustomerByEmail("other@example.com", "kc-other")
	require.NoError(t, err)
	require.Empty(t, kept.KeycloakID)

	_, err = failingMutations("Customer").GetOrCreateCustomerByEmail("new@example.com", "")
	require.ErrorIs(t, err, errInjected)
}

func TestInviteLicenseGroup(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	licenseID, err := Db.CreateItem(Item{Name: "Invite License", Description: testItemDescription, Price: 100, Type: "license_item", IsLicenseItem: true, LicenseGroup: null.StringFrom("from_license")})
	require.NoError(t, err)
	issue := Item{Type: "online_issue", LicenseItem: null.IntFrom(int64(licenseID))}
	require.Equal(t, "from_license", Db.inviteLicenseGroup(issue))

	require.Equal(t, "own", Db.inviteLicenseGroup(Item{Type: "online_issue", LicenseGroup: null.StringFrom("own")}))
	require.Equal(t, "", Db.inviteLicenseGroup(Item{Type: "online_issue", LicenseItem: null.IntFrom(999999)}))

	// An abonement grants the current online issue
	_, err = Db.CreateItem(Item{Name: "Current Issue", Description: testItemDescription, Price: 300, Type: "online_issue", LicenseGroup: null.StringFrom("current_issue")})
	require.NoError(t, err)
	require.Equal(t, "current_issue", Db.inviteLicenseGroup(Item{Type: "abonement", LicenseGroup: null.StringFrom("abo")}))
}

func TestBackfillSettingsFromEnvironment(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	ctx := context.Background()
	_, err := Db.EntClient.Settings.Update().SetOnlinePaperUrl("").SetOdooWebhookURL("").SetOdooWebhookToken("").Save(ctx)
	require.NoError(t, err)

	require.NoError(t, Db.BackfillOnlinePaperUrl(""))
	require.NoError(t, Db.BackfillOdooWebhook("", "x"))
	require.NoError(t, Db.BackfillOnlinePaperUrl("https://zeitung.example.com"))
	require.NoError(t, Db.BackfillOdooWebhook("https://odoo.example.com", "token"))
	// An admin's value is never overridden
	require.NoError(t, Db.BackfillOnlinePaperUrl("https://other.example.com"))
	require.NoError(t, Db.BackfillOdooWebhook("https://other.example.com", "other"))

	settings, err := Db.GetSettings()
	require.NoError(t, err)
	require.Equal(t, "https://zeitung.example.com", settings.OnlinePaperUrl)
	require.Equal(t, "https://odoo.example.com", settings.OdooWebhookURL)
	require.Equal(t, "token", settings.OdooWebhookToken)

	require.ErrorIs(t, failingMutations("Settings").BackfillOnlinePaperUrl("x"), errInjected)
	require.ErrorIs(t, failingMutations("Settings").BackfillOdooWebhook("x", "y"), errInjected)
}

func TestInitiateSingletons(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	// Both exist already, so nothing is written
	require.NoError(t, failingMutations("DBSettings").InitiateDBSettings())
	require.NoError(t, failingMutations("Settings").InitiateSettings())
}

func TestPDFDownloadLinksOfOrder(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	vendorID, err := Db.CreateVendor(Vendor{FirstName: "PDF", LastName: "Links", Email: "pdflinks@vendor.test", LicenseID: null.StringFrom("tv-pdflinks")})
	require.NoError(t, err)
	orderID, err := Db.CreateOrder(Order{Vendor: vendorID})
	require.NoError(t, err)
	order, err := Db.GetOrderByID(orderID)
	require.NoError(t, err)
	require.Nil(t, order.GetPDFDownloadLinks())

	pdfID, err := Db.CreatePDF(PDF{Path: "/tmp/links.pdf"})
	require.NoError(t, err)
	itemID, err := Db.CreateItem(Item{Name: "PDF Links Item", Description: testItemDescription, Price: 300, Type: "issue", IsPDFItem: true, PDF: null.IntFrom(pdfID)})
	require.NoError(t, err)
	tx, err := Db.EntClient.Tx(context.Background())
	require.NoError(t, err)
	_, err = Db.CreatePDFDownload(tx, PDF{ID: int(pdfID)}, orderID, itemID)
	require.NoError(t, err)
	byOrder, err := Db.GetPDFDownloadByOrderIdTx(tx, orderID)
	require.NoError(t, err)
	require.Len(t, byOrder, 1)
	require.NoError(t, tx.Commit())

	links := order.GetPDFDownloadLinks()
	require.NotNil(t, links)
	require.Len(t, *links, 1)
	require.EqualValues(t, itemID, (*links)[0].ItemID.Int64)
}
