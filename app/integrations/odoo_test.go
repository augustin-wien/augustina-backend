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

// TestSendPaymentToOdooSuccess tests successful payment sending to Odoo webhook
func TestSendPaymentToOdooSuccess(t *testing.T) {
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		return database.Item{
			ID:   id,
			Name: "Test Item",
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var receivedPayload OdooPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		require.Equal(t, 123, receivedPayload.OrderID)
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

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.OdooWebhookURL
	config.Config.OdooWebhookURL = mockServer.URL
	defer func() { config.Config.OdooWebhookURL = originalURL }()

	timestamp := time.Date(2025, 12, 10, 14, 30, 0, 0, time.UTC)
	items := []database.OrderEntry{
		{
			ID:       1,
			Item:     1,
			Quantity: 2,
			Price:    2500,
			Receiver: 6234,
		},
		{
			ID:       2,
			Item:     2,
			Quantity: 1,
			Price:    2500,
			Receiver: 6234,
		},
	}
	vendor := database.Vendor{
		ID:        6234,
		FirstName: "Test",
		LastName:  "Vendor",
		Email:     "test@example.com",
		LicenseID: null.StringFrom("test-license-123"),
	}

	err := SendPaymentToOdoo(123, timestamp, items, vendor, 7500)
	require.NoError(t, err)
}

// TestSendPaymentToOdooEmptyItems tests sending payment with empty items list
func TestSendPaymentToOdooEmptyItems(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedPayload OdooPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		require.Len(t, receivedPayload.Items, 0)
		require.Equal(t, 456, receivedPayload.OrderID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.OdooWebhookURL
	config.Config.OdooWebhookURL = mockServer.URL
	defer func() { config.Config.OdooWebhookURL = originalURL }()

	timestamp := time.Now()
	vendor := database.Vendor{
		ID:        6235,
		LicenseID: null.StringFrom("test-license-456"),
	}

	err := SendPaymentToOdoo(456, timestamp, []database.OrderEntry{}, vendor, 1000)
	require.NoError(t, err)
}

// TestSendPaymentToOdooWithNullLicenseID tests sending payment with null license ID
func TestSendPaymentToOdooWithNullLicenseID(t *testing.T) {
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		return database.Item{
			ID:   id,
			Name: "Test Item",
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedPayload OdooPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		require.False(t, receivedPayload.LicenseID.Valid)
		require.Equal(t, 789, receivedPayload.OrderID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.OdooWebhookURL
	config.Config.OdooWebhookURL = mockServer.URL
	defer func() { config.Config.OdooWebhookURL = originalURL }()

	timestamp := time.Now()
	items := []database.OrderEntry{
		{
			ID:       1,
			Item:     1,
			Quantity: 1,
			Price:    3000,
			Receiver: 6236,
		},
	}
	vendor := database.Vendor{
		ID:        6236,
		LicenseID: null.String{},
	}

	err := SendPaymentToOdoo(789, timestamp, items, vendor, 3000)
	require.NoError(t, err)
}

// TestSendPaymentToOdooMultipleItems tests sending payment with multiple items
func TestSendPaymentToOdooMultipleItems(t *testing.T) {
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		return database.Item{
			ID:   id,
			Name: "Test Item",
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedPayload OdooPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		require.Len(t, receivedPayload.Items, 5)
		for i := 0; i < 5; i++ {
			require.Equal(t, i+1, receivedPayload.Items[i].ID)
		}
		require.Equal(t, 555, receivedPayload.OrderID)
		require.Equal(t, 27000, receivedPayload.Price)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"created"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.OdooWebhookURL
	config.Config.OdooWebhookURL = mockServer.URL
	defer func() { config.Config.OdooWebhookURL = originalURL }()

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

	err := SendPaymentToOdoo(555, timestamp, items, vendor, 27000)
	require.NoError(t, err)
}

// TestSendPaymentToOdooPayloadStructure tests that payload structure is correct
func TestSendPaymentToOdooPayloadStructure(t *testing.T) {
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

	originalURL := config.Config.OdooWebhookURL
	config.Config.OdooWebhookURL = mockServer.URL
	defer func() { config.Config.OdooWebhookURL = originalURL }()

	timestamp := time.Date(2025, 12, 10, 16, 0, 0, 0, time.UTC)
	items := []database.OrderEntry{
		{ID: 1, Item: 100, Quantity: 1, Price: 2000, Receiver: 9999},
	}
	vendor := database.Vendor{
		ID:        9999,
		LicenseID: null.StringFrom("structure-test-license"),
	}

	err := SendPaymentToOdoo(666, timestamp, items, vendor, 2000)
	require.NoError(t, err)

	var payload OdooPayload
	err = json.Unmarshal(capturedPayload, &payload)
	require.NoError(t, err)

	require.Equal(t, 666, payload.OrderID)
	require.Equal(t, 2000, payload.Price)
	require.Equal(t, "structure-test-license", payload.LicenseID.String)
	require.True(t, payload.Timestamp.Equal(timestamp))
	require.Len(t, payload.Items, 1)
	require.Equal(t, 100, payload.Items[0].ID)
	require.Equal(t, 1, payload.Items[0].Quantity)
	require.Equal(t, 2000, payload.Items[0].Price)
}

// TestSendPaymentToOdooDonationWithMultipleQuantities tests that a donation item is sent as quantity=1
// with the full price, instead of 200x 1-cent entries.
func TestSendPaymentToOdooDonationWithMultipleQuantities(t *testing.T) {
	originalGetItemByID := getItemByID
	getItemByID = func(id int) (database.Item, error) {
		itemType := "normal_item"
		if id == 2 {
			itemType = "donation"
		}
		return database.Item{
			ID:   id,
			Name: "Test Item",
			Type: itemType,
		}, nil
	}
	defer func() { getItemByID = originalGetItemByID }()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedPayload OdooPayload
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		require.Equal(t, 777, receivedPayload.OrderID)
		require.Equal(t, 100, receivedPayload.Price)
		require.Equal(t, "donation-license-123", receivedPayload.LicenseID.String)

		require.Len(t, receivedPayload.Items, 3)

		// Item 1: Magazine (quantity 5, price 50 cents each)
		require.Equal(t, 1, receivedPayload.Items[0].ID)
		require.Equal(t, 5, receivedPayload.Items[0].Quantity)
		require.Equal(t, 50, receivedPayload.Items[0].Price)

		// Item 2: Donation to charity — collapsed to quantity=1 with full price
		require.Equal(t, 2, receivedPayload.Items[1].ID)
		require.Equal(t, 1, receivedPayload.Items[1].Quantity)
		require.Equal(t, -100, receivedPayload.Items[1].Price)

		// Item 3: Service fee (quantity 1, price -50 cents, receiver is not vendor)
		require.Equal(t, 3, receivedPayload.Items[2].ID)
		require.Equal(t, 1, receivedPayload.Items[2].Quantity)
		require.Equal(t, -50, receivedPayload.Items[2].Price)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	originalURL := config.Config.OdooWebhookURL
	config.Config.OdooWebhookURL = mockServer.URL
	defer func() { config.Config.OdooWebhookURL = originalURL }()

	timestamp := time.Date(2025, 12, 12, 10, 30, 0, 0, time.UTC)
	items := []database.OrderEntry{
		{
			ID:       1,
			Item:     1,
			Quantity: 5,
			Price:    50,
			Receiver: 6240,
		},
		{
			ID:       2,
			Item:     2,
			Quantity: 100,
			Price:    1,
			Receiver: 1,
		},
		{
			ID:       3,
			Item:     3,
			Quantity: 1,
			Price:    50,
			Receiver: 2,
		},
	}
	vendor := database.Vendor{
		ID:        6240,
		FirstName: "Donation",
		LastName:  "Vendor",
		Email:     "donation@example.com",
		LicenseID: null.StringFrom("donation-license-123"),
	}

	err := SendPaymentToOdoo(777, timestamp, items, vendor, 100)
	require.NoError(t, err)
}
