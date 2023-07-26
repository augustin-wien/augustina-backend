package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

func main() {
	apiURL := "https://demo-accounts.vivapayments.com"
	resource := "/connect/token"
	jsonPost := []byte(`grant_type=client_credentials`)
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-accounts.vivapayments.com/connect/token"

	body, err := authenticateToVivaWallet(urlStr, jsonPost)
	if err != nil {
		log.Fatalf("Authentication failed: %s", err)
	}
	log.Printf("res body: %s", string(body))

}

func authenticateToVivaWallet(urlStr string, jsonPost []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Fatalf("Building request failed: %s", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic ZTc2cnBldnR1cmZma3RuZTduMTh2MG94eWozbTZzNTMycjFxNHk0azR4eDEzLmFwcHMudml2YXBheW1lbnRzLmNvbTpxaDA4RmtVMGRGOHZNd0g3NmpHQXVCbVdpYjlXc1A=")

	// Create a new client with a 10 second timeout
	// do not forget to set timeout; otherwise, no timeout!
	client := http.Client{Timeout: 10 * time.Second}
	// send the request
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("impossible to send request: %s", err)
	}
	log.Printf("status Code: %d", res.StatusCode)

	// closes the body after the function returns
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body) // Log the request body
	if err != nil {
		log.Printf("Reading body failed: %s", err)
		return nil, err
	}
	return body, nil
}
