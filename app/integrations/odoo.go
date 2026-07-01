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

	"gopkg.in/guregu/null.v4"
)

func sendJSONToOdooWebhook(payload interface{}) error {
	endpoint := config.Config.OdooWebhookURL

	log.Debugf("sendJSONToOdooWebhook: Sending payload to Odoo webhook: %+v", payload)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	var retryCount int
	backoff := initialBackoff

	for {
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Errorf("sendJSONToOdooWebhook: Failed to create request: %v\n", err)
			return fmt.Errorf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Odoo-Database", "huk_odoo19")
		if token := config.Config.OdooWebhookToken; token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			log.Debug("sendJSONToOdooWebhook: Added Authorization header to request")
		}

		if dump, dumpErr := httputil.DumpRequestOut(req, true); dumpErr == nil {
			log.Debugf("sendJSONToOdooWebhook: HTTP request:\n%s", string(dump))
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("sendJSONToOdooWebhook: Request failed: %v\n", err)
		} else {
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK || resp.StatusCode == 201 {
				log.Info("sendJSONToOdooWebhook: Successfully sent Odoo JSON payload")
				return nil
			}

			if resp.StatusCode == http.StatusUnprocessableEntity {
				log.Infof("sendJSONToOdooWebhook: Received 422 Unprocessable Entity, treating as non-fatal")
				return nil
			}

			log.Infof("sendJSONToOdooWebhook: Received non-200 response: %d\n", resp.StatusCode)
		}

		if retryCount >= maxRetries {
			return errors.New("exceeded maximum retries, aborting")
		}

		retryCount++
		log.Infof("Retrying in %v... (attempt %d)\n", backoff, retryCount)
		time.Sleep(backoff)

		backoff *= 2
	}
}

type OdooPayload struct {
	OrderID   int               `json:"order_id"`
	Price     int               `json:"price"`
	LicenseID null.String       `json:"license_id"`
	Timestamp time.Time         `json:"timestamp"`
	Items     []OdooPayloadItem `json:"items"`
}

type OdooPayloadItem struct {
	ID       int    `json:"id"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
}

func SendPaymentToOdoo(id int, timestamp time.Time, items []database.OrderEntry, vendor database.Vendor, price int) error {
	log.Info("SendPaymentToOdoo: Sending payment to Odoo for order ", id)
	odooItems := make([]OdooPayloadItem, 0)
	totalPrice := 0
	for _, item := range items {
		itemPrice := item.Price
		log.Debugf("SendPaymentToOdoo: Processing item %+v for vendor %+v", item, vendor)
		if item.Receiver <= 5 {
			itemPrice = -itemPrice
		}
		totalPrice += itemPrice * item.Quantity

		itemName := ""
		itemType := ""
		if dbItem, err := getItemByID(item.Item); err != nil {
			log.Errorf("SendPaymentToOdoo: Failed to fetch item name for item %d: %v", item.Item, err)
		} else {
			itemName = dbItem.Name
			itemType = dbItem.Type
		}

		odooQuantity := item.Quantity
		odooPrice := itemPrice
		if itemType == "donation" {
			odooPrice = itemPrice * item.Quantity
			odooQuantity = 1
		}

		odooItems = append(odooItems, OdooPayloadItem{
			ID:       item.Item,
			Quantity: odooQuantity,
			Price:    odooPrice,
			Name:     itemName,
			Type:     itemType,
		})
	}
	payload := OdooPayload{
		OrderID:   id,
		Price:     totalPrice,
		LicenseID: vendor.LicenseID,
		Timestamp: timestamp,
		Items:     odooItems,
	}
	payloadJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Errorf("SendPaymentToOdoo: Failed to marshal payload to JSON: %v", err)
	} else {
		log.Debugf("SendPaymentToOdoo: Payload JSON:\n%s", string(payloadJSON))
	}

	err = sendJSONToOdooWebhook(payload)
	if err != nil {
		log.Errorf("Failed to send JSON to Odoo: %v\n", err)
	}
	return err
}
