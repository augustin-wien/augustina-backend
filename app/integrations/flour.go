package integrations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

// Payload is the structure of the JSON payload to be sent.
type Payload struct {
	Message string `json:"message"`
}

// sendJSONToWebhook sends a JSON payload to a webhook and retries on failure.
func sendJSONToFlourWebhook(payload interface{}) error {
	flourwebhookEndpoint := config.Config.FlourWebhookURL

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

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("sendJSONToFlourWebhook: Request failed: %v\n", err)
		} else {
			defer resp.Body.Close()

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
	ID       int `json:"id"`
	Quantity int `json:"quantity"`
	Price    int `json:"price"`
}

func SendPaymentToFlour(id int, timestamp time.Time, items []database.OrderEntry, vendor database.Vendor, price int) error {
	log.Info("SendPaymentToFlour: Sending payment to Flour for order ", id)
	flourItems := make([]FlourPayloadItem, 0)
	for _, item := range items {
		flourItems = append(flourItems, FlourPayloadItem{
			ID:       item.Item,
			Quantity: item.Quantity,
			Price:    item.Price,
		})
	}
	// Create a new payload
	payload := FlourPayload{
		OrderID:   id,
		Price:     price,
		LicenseID: vendor.LicenseID,
		Timestamp: timestamp,
		Items:     flourItems,
	}

	// Send the payload to the webhook
	err := sendJSONToFlourWebhook(payload)
	if err != nil {
		log.Errorf("Failed to send JSON to flour: %v\n", err)
	}
	return err
}
