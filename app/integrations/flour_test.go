package integrations

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestSendPaymentToFlourSuccess tests successful payment sending to Flour webhook
func TestSendPaymentToFlourSuccess(t *testing.T) {
	// Mock getItemByID
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		return database.Item{
			ID:   id,
			Name: "Test Item",
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	// Create a mock server to capture the webhook request
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse the incoming payload
		var receivedPayload FlourPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		// Verify payload structure
		require.Equal(t, 123, receivedPayload.OrderID)
		// Total price: (2500 * 2) + (2500 * 1) = 7500
		require.Equal(t, 7500, receivedPayload.Price)
		require.Equal(t, "test-license-123", receivedPayload.LicenseID.String)
		require.Len(t, receivedPayload.Items, 2)
		require.Equal(t, 1, receivedPayload.Items[0].ID)
		require.Equal(t, 2, receivedPayload.Items[0].Quantity)
		require.Equal(t, 2500, receivedPayload.Items[0].Price)
		require.Equal(t, "Test Item", receivedPayload.Items[0].Name)
		require.Equal(t, 2, receivedPayload.Items[1].ID)
		require.Equal(t, 1, receivedPayload.Items[1].Quantity)
		require.Equal(t, 2500, receivedPayload.Items[1].Price)
		require.Equal(t, "Test Item", receivedPayload.Items[1].Name)

		// Return successful response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	// Set the webhook URL to our mock server
	originalURL := config.Config.FlourWebhookURL
	config.Config.FlourWebhookURL = mockServer.URL
	defer func() { config.Config.FlourWebhookURL = originalURL }()

	// Create test data
	timestamp := time.Date(2025, 12, 10, 14, 30, 0, 0, time.UTC)
	items := []database.OrderEntry{
		{
			ID:       1,
			Item:     1,
			Quantity: 2,
			Price:    2500,
			Receiver: 6234, // Set to vendor ID for positive price
		},
		{
			ID:       2,
			Item:     2,
			Quantity: 1,
			Price:    2500,
			Receiver: 6234, // Set to vendor ID for positive price
		},
	}
	vendor := database.Vendor{
		ID:        6234,
		FirstName: "Test",
		LastName:  "Vendor",
		Email:     "test@example.com",
		LicenseID: null.StringFrom("test-license-123"),
	}

	// Call the function
	// Total price: (2500 * 2) + (2500 * 1) = 5000 + 2500 = 7500
	err := SendPaymentToFlour(123, timestamp, items, vendor, 7500)
	require.NoError(t, err)
}

// TestSendPaymentToFlourEmptyItems tests sending payment with empty items list
func TestSendPaymentToFlourEmptyItems(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedPayload FlourPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		// Verify empty items
		require.Len(t, receivedPayload.Items, 0)
		require.Equal(t, 456, receivedPayload.OrderID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.FlourWebhookURL
	config.Config.FlourWebhookURL = mockServer.URL
	defer func() { config.Config.FlourWebhookURL = originalURL }()

	timestamp := time.Now()
	vendor := database.Vendor{
		ID:        6235,
		LicenseID: null.StringFrom("test-license-456"),
	}

	err := SendPaymentToFlour(456, timestamp, []database.OrderEntry{}, vendor, 1000)
	require.NoError(t, err)
}

// TestSendPaymentToFlourWithNullLicenseID tests sending payment with null license ID
func TestSendPaymentToFlourWithNullLicenseID(t *testing.T) {
	// Mock getItemByID
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		return database.Item{
			ID:   id,
			Name: "Test Item",
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedPayload FlourPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		// Verify license ID is null
		require.False(t, receivedPayload.LicenseID.Valid)
		require.Equal(t, 789, receivedPayload.OrderID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.FlourWebhookURL
	config.Config.FlourWebhookURL = mockServer.URL
	defer func() { config.Config.FlourWebhookURL = originalURL }()

	timestamp := time.Now()
	items := []database.OrderEntry{
		{
			ID:       1,
			Item:     1,
			Quantity: 1,
			Price:    3000,
			Receiver: 6236, // Vendor ID
		},
	}
	vendor := database.Vendor{
		ID:        6236,
		LicenseID: null.String{}, // null license ID
	}

	err := SendPaymentToFlour(789, timestamp, items, vendor, 3000)
	require.NoError(t, err)
}

// TestSendPaymentToFlourConnectionError tests handling of connection errors
// func TestSendPaymentToFlourConnectionError(t *testing.T) {
// 	// Use an invalid URL that will cause connection errors
// 	originalURL := config.Config.FlourWebhookURL
// 	config.Config.FlourWebhookURL = "http://invalid-url"
// 	defer func() { config.Config.FlourWebhookURL = originalURL }()

// 	timestamp := time.Now()
// 	items := []database.OrderEntry{
// 		{
// 			ID:       1,
// 			Item:     1,
// 			Quantity: 1,
// 			Price:    5000,
// 			Receiver: 6237, // Vendor ID
// 		},
// 	}
// 	vendor := database.Vendor{
// 		ID:        6237,
// 		LicenseID: null.StringFrom("test-license-789"),
// 	}

// 	// Should return an error after retries
// 	err := SendPaymentToFlour(999, timestamp, items, vendor, 5000)
// 	require.Error(t, err)
// 	require.Contains(t, err.Error(), "exceeded maximum retries")
// }

// TestSendPaymentToFlourMultipleItems tests sending payment with multiple items
func TestSendPaymentToFlourMultipleItems(t *testing.T) {
	// Mock getItemByID
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		return database.Item{
			ID:   id,
			Name: "Test Item",
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedPayload FlourPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		// Verify multiple items
		require.Len(t, receivedPayload.Items, 5)
		for i := 0; i < 5; i++ {
			require.Equal(t, i+1, receivedPayload.Items[i].ID)
		}
		require.Equal(t, 555, receivedPayload.OrderID)
		// Price should be: (3000*3) + (3000*2) + (3000*2) + (3000*1) + (3000*1) = 9000 + 6000 + 6000 + 3000 + 3000 = 27000
		require.Equal(t, 27000, receivedPayload.Price)

		w.WriteHeader(http.StatusCreated) // Test with 201 Created status as well
		w.Write([]byte(`{"status":"created"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.FlourWebhookURL
	config.Config.FlourWebhookURL = mockServer.URL
	defer func() { config.Config.FlourWebhookURL = originalURL }()

	timestamp := time.Date(2025, 12, 10, 15, 45, 30, 0, time.UTC)
	items := []database.OrderEntry{
		{ID: 1, Item: 1, Quantity: 3, Price: 3000, Receiver: 6238},
		{ID: 2, Item: 2, Quantity: 2, Price: 3000, Receiver: 6238},
		{ID: 3, Item: 3, Quantity: 2, Price: 3000, Receiver: 6238},
		{ID: 4, Item: 4, Quantity: 1, Price: 3000, Receiver: 6238},
		{ID: 5, Item: 5, Quantity: 1, Price: 3000, Receiver: 6238},
	}
	vendor := database.Vendor{
		ID:        6238,
		LicenseID: null.StringFrom("test-license-555"),
	}

	// Price should be: (3000*3) + (3000*2) + (3000*2) + (3000*1) + (3000*1) = 27000
	err := SendPaymentToFlour(555, timestamp, items, vendor, 27000)
	require.NoError(t, err)
}

// TestSendPaymentToFlourPayloadStructure tests that payload structure is correct
func TestSendPaymentToFlourPayloadStructure(t *testing.T) {
	// Mock getItemByID
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		return database.Item{
			ID:   id,
			Name: "Test Item",
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	var capturedPayload []byte
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		buf.ReadFrom(r.Body)
		capturedPayload = buf.Bytes()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.FlourWebhookURL
	config.Config.FlourWebhookURL = mockServer.URL
	defer func() { config.Config.FlourWebhookURL = originalURL }()

	timestamp := time.Date(2025, 12, 10, 16, 0, 0, 0, time.UTC)
	items := []database.OrderEntry{
		{ID: 1, Item: 100, Quantity: 1, Price: 2000, Receiver: 9999},
	}
	vendor := database.Vendor{
		ID:        9999,
		LicenseID: null.StringFrom("structure-test-license"),
	}

	err := SendPaymentToFlour(666, timestamp, items, vendor, 2000)
	require.NoError(t, err)

	// Verify the captured payload is valid JSON
	var payload FlourPayload
	err = json.Unmarshal(capturedPayload, &payload)
	require.NoError(t, err)

	// Verify all fields are present and correct
	require.Equal(t, 666, payload.OrderID)
	// Price should be sum of items: 1 item * 2000 = 2000
	require.Equal(t, 2000, payload.Price)
	require.Equal(t, "structure-test-license", payload.LicenseID.String)
	require.True(t, payload.Timestamp.Equal(timestamp))
	require.Len(t, payload.Items, 1)
	require.Equal(t, 100, payload.Items[0].ID)
	require.Equal(t, 1, payload.Items[0].Quantity)
	require.Equal(t, 2000, payload.Items[0].Price)
}

// TestSendPaymentToFlourDonationWithMultipleQuantities tests 100 cents donation with multiple items and quantities
func TestSendPaymentToFlourDonationWithMultipleQuantities(t *testing.T) {
	// Mock getItemByID
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		return database.Item{
			ID:   id,
			Name: "Test Item",
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedPayload FlourPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		// Verify payload for donation scenario
		require.Equal(t, 777, receivedPayload.OrderID)
		// Price should be: (50*5) + (-100*1) + (-50*1) = 250 - 100 - 50 = 100 cents
		require.Equal(t, 100, receivedPayload.Price)
		require.Equal(t, "donation-license-123", receivedPayload.LicenseID.String)

		// Verify items with different quantities
		require.Len(t, receivedPayload.Items, 3)

		// Item 1: Magazine (quantity 5, price 50 cents each)
		require.Equal(t, 1, receivedPayload.Items[0].ID)
		require.Equal(t, 5, receivedPayload.Items[0].Quantity)
		require.Equal(t, 50, receivedPayload.Items[0].Price)

		// Item 2: Donation to charity (quantity 1, price -100 cents, receiver is charity not vendor)
		require.Equal(t, 2, receivedPayload.Items[1].ID)
		require.Equal(t, 100, receivedPayload.Items[1].Quantity)
		require.Equal(t, -1, receivedPayload.Items[1].Price)

		// Item 3: Service fee (quantity 1, price -50 cents, receiver is not vendor)
		require.Equal(t, 3, receivedPayload.Items[2].ID)
		require.Equal(t, 1, receivedPayload.Items[2].Quantity)
		require.Equal(t, -50, receivedPayload.Items[2].Price)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.FlourWebhookURL
	config.Config.FlourWebhookURL = mockServer.URL
	defer func() { config.Config.FlourWebhookURL = originalURL }()

	timestamp := time.Date(2025, 12, 12, 10, 30, 0, 0, time.UTC)
	items := []database.OrderEntry{
		{
			ID:       1,
			Item:     1, // Magazine
			Quantity: 5,
			Price:    50,   // 50 cents each
			Receiver: 6240, // Vendor receives this
		},
		{
			ID:       2,
			Item:     2, // Donation to charity
			Quantity: 100,
			Price:    1, // 100 cents donation
			Receiver: 1, // Charity receives this (not vendor, so negated)
		},
		{
			ID:       3,
			Item:     3, // Service fee
			Quantity: 1,
			Price:    50, // 50 cents service fee
			Receiver: 2,  // Platform receives this (not vendor, so negated)
		},
	}
	vendor := database.Vendor{
		ID:        6240,
		FirstName: "Donation",
		LastName:  "Vendor",
		Email:     "donation@example.com",
		LicenseID: null.StringFrom("donation-license-123"),
	}

	// Total price: (50*5) + (-100*1) + (-50*1) = 250 - 100 - 50 = 100 cents
	err := SendPaymentToFlour(777, timestamp, items, vendor, 100)
	require.NoError(t, err)
}
