package integrations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"

	"gopkg.in/guregu/null.v4"
)

var log = utils.GetLogger()

const (
	maxRetries     = 5               // Maximum number of retries
	initialBackoff = 2 * time.Second // Initial backoff time
)

// getItemByID is a variable to allow mocking in tests
var getItemByID = func(id int) (database.Item, error) {
	return database.Db.GetItem(id)
}

// Payload is the structure of the JSON payload to be sent.
type Payload struct {
	Message string `json:"message"`
}

// sendJSONToWebhook sends a JSON payload to a webhook and retries on failure.
func sendJSONToFlourWebhook(payload interface{}) error {
	flourwebhookEndpoint := config.Config.FlourWebhookURL

	log.Debugf("sendJSONToFlourWebhook: Sending payload to Flour webhook: %+v", payload)

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Retry logic with exponential backoff
	var retryCount int
	backoff := initialBackoff

	for {
		// Create a new HTTP request
		req, err := http.NewRequest("POST", flourwebhookEndpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Errorf("sendJSONToFlourWebhook: Failed to create request: %v\n", err)
			return fmt.Errorf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Odoo-Database", "huk_odoo19")
		if token := config.Config.FlourWebhookToken; token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			log.Debug("sendJSONToFlourWebhook: Added Authorization header to request")
		}

		if dump, dumpErr := httputil.DumpRequestOut(req, true); dumpErr == nil {
			log.Debugf("sendJSONToFlourWebhook: HTTP request:\n%s", string(dump))
		}

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("sendJSONToFlourWebhook: Request failed: %v\n", err)
		} else {
			resp.Body.Close()

			// Check for 200 OK status
			if resp.StatusCode == http.StatusOK || resp.StatusCode == 201 {
				log.Info("sendJSONToFlourWebhook: Successfully sent flour JSON payload")
				return nil
			}

			log.Infof("sendJSONToFlourWebhook: Received non-200 response: %d\n", resp.StatusCode)
		}

		// Check if we've exceeded the retry limit
		if retryCount >= maxRetries {
			return errors.New("exceeded maximum retries, aborting")
		}

		// Increment retry count and apply backoff
		retryCount++
		log.Infof("Retrying in %v... (attempt %d)\n", backoff, retryCount)
		time.Sleep(backoff)

		// Exponentially increase the backoff time
		backoff *= 2
	}
}

type FlourPayload struct {
	OrderID   int                `json:"order_id"`
	Price     int                `json:"price"`
	LicenseID null.String        `json:"license_id"`
	Timestamp time.Time          `json:"timestamp"`
	Items     []FlourPayloadItem `json:"items"`
}

type FlourPayloadItem struct {
	ID       int    `json:"id"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
}

func SendPaymentToFlour(id int, timestamp time.Time, items []database.OrderEntry, vendor database.Vendor, price int) error {
	log.Info("SendPaymentToFlour: Sending payment to Flour for order ", id)
	flourItems := make([]FlourPayloadItem, 0)
	totalPrice := 0
	for _, item := range items {
		itemPrice := item.Price
		// If the receiver of the OrderEntry is not the Vendor, the price should be negative
		log.Debugf("SendPaymentToFlour: Processing item %+v for vendor %+v", item, vendor)
		if item.Receiver <= 5 { // system accounts have IDs 1-5
			itemPrice = -itemPrice
		}
		// Total price includes the quantity: price * quantity
		totalPrice += itemPrice * item.Quantity

		itemName := ""
		itemType := ""
		if dbItem, err := getItemByID(item.Item); err != nil {
			log.Errorf("SendPaymentToFlour: Failed to fetch item name for item %d: %v", item.Item, err)
		} else {
			itemName = dbItem.Name
			itemType = dbItem.Type
		}

		flourItems = append(flourItems, FlourPayloadItem{
			ID:       item.Item,
			Quantity: item.Quantity,
			Price:    itemPrice,
			Name:     itemName,
			Type:     itemType,
		})
	}
	// Create a new payload
	payload := FlourPayload{
		OrderID:   id,
		Price:     totalPrice,
		LicenseID: vendor.LicenseID,
		Timestamp: timestamp,
		Items:     flourItems,
	}
	// print payload as JSON for debugging
	payloadJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Errorf("SendPaymentToFlour: Failed to marshal payload to JSON: %v", err)
	} else {
		log.Debugf("SendPaymentToFlour: Payload JSON:\n%s", string(payloadJSON))
	}

	// Send the payload to the webhook
	err = sendJSONToFlourWebhook(payload)
	if err != nil {
		log.Errorf("Failed to send JSON to flour: %v\n", err)
	}
	return err
}
