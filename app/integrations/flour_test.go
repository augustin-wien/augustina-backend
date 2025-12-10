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
		require.Equal(t, 5000, receivedPayload.Price)
		require.Equal(t, "test-license-123", receivedPayload.LicenseID.String)
		require.Len(t, receivedPayload.Items, 2)
		require.Equal(t, 1, receivedPayload.Items[0].ID)
		require.Equal(t, 2, receivedPayload.Items[0].Quantity)
		require.Equal(t, 2500, receivedPayload.Items[0].Price)
		require.Equal(t, 2, receivedPayload.Items[1].ID)
		require.Equal(t, 1, receivedPayload.Items[1].Quantity)
		require.Equal(t, 2500, receivedPayload.Items[1].Price)

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
	err := SendPaymentToFlour(123, timestamp, items, vendor, 5000)
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
		// Price should be adjusted: 15000 + 15000 (5 items * 3000 negated) = 30000
		require.Equal(t, 30000, receivedPayload.Price)

		w.WriteHeader(http.StatusCreated) // Test with 201 Created status as well
		w.Write([]byte(`{"status":"created"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.FlourWebhookURL
	config.Config.FlourWebhookURL = mockServer.URL
	defer func() { config.Config.FlourWebhookURL = originalURL }()

	timestamp := time.Date(2025, 12, 10, 15, 45, 30, 0, time.UTC)
	items := []database.OrderEntry{
		{ID: 1, Item: 1, Quantity: 3, Price: 3000, Receiver: 6235},
		{ID: 2, Item: 2, Quantity: 2, Price: 3000, Receiver: 6235},
		{ID: 3, Item: 3, Quantity: 2, Price: 3000, Receiver: 6235},
		{ID: 4, Item: 4, Quantity: 1, Price: 3000, Receiver: 6235},
		{ID: 5, Item: 5, Quantity: 1, Price: 3000, Receiver: 6235},
	}
	vendor := database.Vendor{
		ID:        6238,
		LicenseID: null.StringFrom("test-license-555"),
	}

	// Price should be 15000 + (5 items * 3000 negated) = 15000 + 15000 = 30000
	err := SendPaymentToFlour(555, timestamp, items, vendor, 15000)
	require.NoError(t, err)
}

// TestSendPaymentToFlourPayloadStructure tests that payload structure is correct
func TestSendPaymentToFlourPayloadStructure(t *testing.T) {
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
		{ID: 1, Item: 100, Quantity: 1, Price: 2000, Receiver: 6238},
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
	// Price should be adjusted: 2000 + 2000 (negated item price) = 4000
	require.Equal(t, 4000, payload.Price)
	require.Equal(t, "structure-test-license", payload.LicenseID.String)
	require.True(t, payload.Timestamp.Equal(timestamp))
	require.Len(t, payload.Items, 1)
	require.Equal(t, 100, payload.Items[0].ID)
	require.Equal(t, 1, payload.Items[0].Quantity)
	require.Equal(t, -2000, payload.Items[0].Price)
}
