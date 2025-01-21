package database

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestMain is the main function for the database tests and initializes the database
func TestMain(m *testing.M) {
	// run tests in mainfolder
	err := os.Chdir("..")
	if err != nil {
		panic(err)
	}
	config.InitConfig()
	Db.InitEmptyTestDb()
	os.Exit(m.Run())
}

// Test_pingDB tests if the database can be pinged
func Test_pingDB(t *testing.T) {
	if Db.Dbpool == nil {
		t.Error("dbpool is nil")
	}
	err := Db.Dbpool.Ping(context.Background())
	if err != nil {
		t.Error("can't ping database")
	}
}

func Test_DatabaseTablesCreated(t *testing.T) {
	var tableCount int
	err := Db.Dbpool.QueryRow(context.Background(), "select count(*) from information_schema.tables where table_schema = 'public'").Scan(&tableCount)
	if err != nil {
		t.Error("can't get table count")
	}

	// Check if there is at least one table
	if tableCount < 1 {
		t.Error("No tables exist")
	}
}

func Test_GetHelloWorld(t *testing.T) {
	var greeting string
	greeting, err := Db.GetHelloWorld()
	// err := dbpool.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		t.Error("can't get hello world")
	}
	if reflect.TypeOf(greeting).Kind() != reflect.String {
		t.Error("Hello World not of type string")
	}

	require.Equal(t, string(greeting), "Hello, world!")
}

func Test_UpdateSettings(t *testing.T) {
	// Define settings
	err := Db.UpdateSettings(Settings{
		Color: "red",
		Logo:  "/img/Augustin-Logo-Rechteck.jpg",
	},
	)

	if err != nil {
		t.Errorf("UpdateSettings failed: %v\n", err)
	}
}

func Test_GetSettings(t *testing.T) {
	// Define settings
	settings, err := Db.GetSettings()

	if err != nil {
		t.Errorf("GetSettings failed: %v\n", err)
	}

	require.Equal(t, "red", settings.Color)
	require.Equal(t, "/img/Augustin-Logo-Rechteck.jpg", settings.Logo)
}

func TestAccounts(t *testing.T) {
	// Get account by type (default accounts should exist)
	account, err := Db.GetAccountByType("Cash")
	utils.CheckError(t, err)
	require.Equal(t, "Cash", account.Name)

	// Create new account with known
	test_vendor := Vendor{
		LicenseID: null.StringFrom("UserAuth"),
		Email:     "UserAuth@augustina.cc",
	}

	id, err := Db.CreateSpecialVendorAccount(test_vendor)
	utils.CheckError(t, err)

	// Get account by ID
	test_vendor, err = Db.GetVendor(id)
	utils.CheckError(t, err)

	require.Equal(t, "UserAuth", test_vendor.LicenseID.String)
}

func TestVendors(t *testing.T) {
	Db.InitEmptyTestDb()
	licenseId := "tt-123"
	vendorName := "test"
	vendorEmail := vendorName + "@example.com"
	// Create new vendor
	vendor := Vendor{
		FirstName:      vendorName,
		LastName:       vendorName,
		Email:          vendorEmail,
		LicenseID:      null.StringFrom(licenseId),
		HasBankAccount: true,
		Locations: []*ent.Location{
			&ent.Location{
				Name:        "test",
				Address:     "test",
				Longitude:   10.0,
				Latitude:    20.0,
				Zip:         "1234",
				WorkingTime: "G",
			},
		},
		Comments: []*ent.Comment{
			&ent.Comment{
				Comment:    "test",
				Warning:    false,
				CreatedAt:  time.Now(),
				ResolvedAt: time.Now(),
			},
		},
	}
	id, err := Db.CreateVendor(vendor)
	utils.CheckError(t, err)

	// Get vendor by ID
	vendor, err = Db.GetVendor(id)
	utils.CheckError(t, err)

	require.Equal(t, vendorName, vendor.FirstName)
	require.Equal(t, true, vendor.HasBankAccount)

	// Update vendor
	vendorName = "test2"
	vendor.FirstName = vendorName
	vendor.HasBankAccount = false
	err = Db.UpdateVendor(id, vendor)
	utils.CheckError(t, err)

	// Get vendor by ID
	vendor, err = Db.GetVendor(id)
	utils.CheckError(t, err)

	require.Equal(t, vendorName, vendor.FirstName)

	// Get all vendors
	vendors, err := Db.ListVendors()
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors))

	// Get vendor by LicenseID
	vendor, err = Db.GetVendorByLicenseID(licenseId)
	utils.CheckError(t, err)
	require.Equal(t, vendorName, vendor.FirstName)

	// Get vendor by Email
	vendor, err = Db.GetVendorByEmail(vendorEmail)
	utils.CheckError(t, err)
	require.Equal(t, vendorName, vendor.FirstName)

	// Get vendor locations
	vendorMap, err := Db.GetVendorLocations()
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendorMap))
	require.Equal(t, 10.0, vendorMap[0].Longitude)
	require.Equal(t, 20.0, vendorMap[0].Latitude)
	require.Equal(t, vendorName, vendorMap[0].FirstName)
	require.Equal(t, id, vendorMap[0].ID)

	// Delete vendor
	err = Db.DeleteVendor(id)
	utils.CheckError(t, err)
}

// TestItems tests the item database functions

// TODO: Test payments

// TODO: This test breaks the CI pipeline as it somehow runs in parallel
// to handlers_tests
func TestQueryOrders(t *testing.T) {
	vendorLicenseId := "tt-124"
	// Preperation
	vendorID, err := Db.CreateVendor(Vendor{
		LicenseID: null.StringFrom(vendorLicenseId),
	})
	utils.CheckError(t, err)
	senderVendorID, err := Db.CreateVendor(Vendor{LicenseID: null.StringFrom("sender")})
	utils.CheckError(t, err)
	receiverVendorID, err := Db.CreateVendor(Vendor{LicenseID: null.StringFrom("receiver")})
	utils.CheckError(t, err)
	itemID, err := Db.CreateItem(Item{Price: 1})
	utils.CheckError(t, err)

	senderAccount, err := Db.GetAccountByVendorID(senderVendorID)
	utils.CheckError(t, err)
	receiverAccount, err := Db.GetAccountByVendorID(receiverVendorID)
	utils.CheckError(t, err)

	// Create order
	order := Order{
		Vendor:    int(vendorID),
		OrderCode: null.NewString("0", true),
		Entries: []OrderEntry{
			{
				Item:     int(itemID),
				Quantity: 315,
				Sender:   senderAccount.ID,
				Receiver: receiverAccount.ID,
			},
		},
	}
	orderID, err := Db.CreateOrder(order)
	utils.CheckError(t, err)

	// Create extra order entries with payments
	err = Db.CreatePayedOrderEntries(orderID, []OrderEntry{
		{
			Item:     int(itemID),
			Quantity: 316,
			Sender:   senderAccount.ID,
			Receiver: receiverAccount.ID,
		},
	})
	utils.CheckError(t, err)

	// Verify order and create payments
	err = Db.VerifyOrderAndCreatePayments(orderID, 64)
	utils.CheckError(t, err)

	// Check order results
	order1, err := Db.GetOrderByOrderCode("0")
	utils.CheckError(t, err)
	require.Equal(t, 2, len(order1.Entries))

	// Check payment results
	payments, err := Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false)
	utils.CheckError(t, err)
	require.Equal(t, 2, len(payments))

	// Repeat with reverse order
	order2 := Order{
		Vendor:    int(vendorID),
		OrderCode: null.NewString("1", true),
		Entries: []OrderEntry{
			{
				Item:     int(itemID),
				Quantity: 315,
				Sender:   senderAccount.ID,
				Receiver: receiverAccount.ID,
			},
		},
	}
	orderID2, err := Db.CreateOrder(order2)
	utils.CheckError(t, err)

	// Create extra order entries with payments
	err = Db.CreatePayedOrderEntries(orderID2, []OrderEntry{
		{
			Item:     int(itemID),
			Quantity: 316,
			Sender:   senderAccount.ID,
			Receiver: receiverAccount.ID,
		},
	})
	utils.CheckError(t, err)

	// Verify order and create payments
	err = Db.VerifyOrderAndCreatePayments(orderID2, 64)
	utils.CheckError(t, err)

	// Check order results
	order22, err := Db.GetOrderByOrderCode("1")
	utils.CheckError(t, err)
	require.Equal(t, 2, len(order22.Entries))

	// Check payment results
	payments2, err := Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false)
	utils.CheckError(t, err)
	require.Equal(t, 4, len(payments2))

	// Cleanup
	for _, payment := range payments2 {
		err = Db.DeletePayment(payment.ID)
		utils.CheckError(t, err)
	}
	order1, err = Db.GetOrderByID(orderID)
	utils.CheckError(t, err)
	for _, entry := range order1.Entries {
		err = Db.DeleteOrderEntry(entry.ID)
		utils.CheckError(t, err)
	}
	order2, err = Db.GetOrderByID(orderID2)
	utils.CheckError(t, err)
	for _, entry := range order2.Entries {
		err = Db.DeleteOrderEntry(entry.ID)
		utils.CheckError(t, err)
	}
	err = Db.DeleteOrder(orderID)
	utils.CheckError(t, err)
	err = Db.DeleteOrder(orderID2)
	utils.CheckError(t, err)
	err = Db.DeleteVendor(vendorID)
	utils.CheckError(t, err)

}
