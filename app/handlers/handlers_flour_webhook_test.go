package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestFlourWebhookUpdateVendorDefinedLicense tests updating a vendor via flour webhook with a defined license ID
func TestFlourWebhookUpdateVendorDefinedLicense(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create a vendor with a defined license ID
	licenseID := "defined-license-123"
	vendor := database.Vendor{
		FirstName:      "TestFirst",
		LastName:       "TestLast",
		Email:          "test-vendor@example.com",
		LicenseID:      null.StringFrom(licenseID),
		HasBankAccount: true,
	}
	vendorID, err := database.Db.CreateVendor(vendor)
	require.NoError(t, err)
	require.NotNil(t, vendorID)

	// Verify vendor was created
	vendor, err = database.Db.GetVendorByLicenseID(licenseID)
	require.NoError(t, err)
	require.NotNil(t, vendor)
	require.Equal(t, licenseID, vendor.LicenseID.String)

	// Create update payload
	updatedVendor := database.Vendor{
		FirstName: "UpdatedFirst",
		LastName:  "UpdatedLast",
		Email:     vendor.Email,
	}

	payloadBytes, err := json.Marshal(updatedVendor)
	require.NoError(t, err)

	// Create request with chi URL parameters
	req := httptest.NewRequest("PUT", "/api/flour/vendors/license/"+licenseID+"/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Name", "flour-system")

	// Set chi URL parameter
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("licenseID", licenseID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	rec := httptest.NewRecorder()

	// Call handler
	UpdateVendorByLicenseID(rec, req)

	// Assert response
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify response contains updated fields
	var updatedVendorResponse database.Vendor
	err = json.Unmarshal(rec.Body.Bytes(), &updatedVendorResponse)
	require.NoError(t, err)
	require.Equal(t, "UpdatedFirst", updatedVendorResponse.FirstName)
	require.Equal(t, "UpdatedLast", updatedVendorResponse.LastName)
}

// TestFlourWebhookUpdateVendorUndefinedLicense tests updating a vendor via flour webhook with an undefined license ID
func TestFlourWebhookUpdateVendorUndefinedLicense(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Try to update a vendor with a license ID that doesn't exist
	undefinedLicenseID := "undefined-license-999"

	updatedVendor := database.Vendor{
		FirstName: "ShouldNotExist",
		LastName:  "DoesNotExist",
		Email:     "test@example.com",
	}

	payloadBytes, err := json.Marshal(updatedVendor)
	require.NoError(t, err)

	// Create request with chi URL parameters for undefined license
	req := httptest.NewRequest("PUT", "/api/flour/vendors/license/"+undefinedLicenseID+"/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Name", "flour-system")

	// Set chi URL parameter
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("licenseID", undefinedLicenseID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	rec := httptest.NewRecorder()

	// Call handler
	UpdateVendorByLicenseID(rec, req)

	// Assert response is error (400 Bad Request - missing license ID parameter)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify error message in response
	var errorResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	require.NoError(t, err)
	require.NotEmpty(t, errorResponse)
}

// TestFlourWebhookGetVendorDefinedLicense tests retrieving a vendor via flour webhook with a defined license ID
func TestFlourWebhookGetVendorDefinedLicense(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create a vendor with a defined license ID
	licenseID := "get-vendor-license-123"
	vendor := database.Vendor{
		FirstName:      "GetTestFirst",
		LastName:       "GetTestLast",
		Email:          "get-test-vendor@example.com",
		LicenseID:      null.StringFrom(licenseID),
		HasBankAccount: true,
	}
	vendorID, err := database.Db.CreateVendor(vendor)
	require.NoError(t, err)
	require.NotNil(t, vendorID)

	vendor, err = database.Db.GetVendorByLicenseID(licenseID)
	require.NoError(t, err)
	require.NotNil(t, vendor)

	// Create request to GET vendor with chi URL parameters
	req := httptest.NewRequest("GET", "/api/flour/vendors/license/"+licenseID+"/", nil)
	req.Header.Set("X-Auth-User-Name", "flour-system")

	// Set chi URL parameter
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("licenseID", licenseID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	rec := httptest.NewRecorder()

	// Call handler
	GetVendorByLicenseID(rec, req)

	// Assert response
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify returned vendor data
	var retrievedVendor database.Vendor
	err = json.Unmarshal(rec.Body.Bytes(), &retrievedVendor)
	require.NoError(t, err)
	require.Equal(t, licenseID, retrievedVendor.LicenseID.String)
	require.Equal(t, vendor.ID, retrievedVendor.ID)
}

// TestFlourWebhookGetVendorUndefinedLicense tests retrieving a vendor via flour webhook with an undefined license ID
func TestFlourWebhookGetVendorUndefinedLicense(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Try to retrieve a vendor with a license ID that doesn't exist
	undefinedLicenseID := "undefined-get-license-999"

	// Create request to GET vendor with chi URL parameters
	req := httptest.NewRequest("GET", "/api/flour/vendors/license/"+undefinedLicenseID+"/", nil)
	req.Header.Set("X-Auth-User-Name", "flour-system")

	// Set chi URL parameter
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("licenseID", undefinedLicenseID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	rec := httptest.NewRecorder()

	// Call handler
	GetVendorByLicenseID(rec, req)

	// Assert response is error (400 Bad Request - missing license ID parameter)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify error message in response
	var errorResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	require.NoError(t, err)
	require.NotEmpty(t, errorResponse)
}

// TestFlourWebhookGetVendorMissingLicenseID tests flour GET webhook with missing license ID parameter
func TestFlourWebhookUpdateVendorMissingLicenseID(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	updatedVendor := database.Vendor{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}

	payloadBytes, err := json.Marshal(updatedVendor)
	require.NoError(t, err)

	// Create request with missing chi URL parameter
	req := httptest.NewRequest("PUT", "/api/flour/vendors/license//", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Name", "flour-system")

	// Don't set chi URL parameter - it will be empty
	chiCtx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	rec := httptest.NewRecorder()

	// Call handler
	UpdateVendorByLicenseID(rec, req)

	// Assert response is error
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var errorResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	require.NoError(t, err)
	require.NotEmpty(t, errorResponse)
}

// TestFlourWebhookGetVendorMissingLicenseID tests flour GET webhook with missing license ID parameter
func TestFlourWebhookGetVendorMissingLicenseID(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create request with missing chi URL parameter
	req := httptest.NewRequest("GET", "/api/flour/vendors/license//", nil)
	req.Header.Set("X-Auth-User-Name", "flour-system")

	// Don't set chi URL parameter - it will be empty
	chiCtx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	rec := httptest.NewRecorder()

	// Call handler
	GetVendorByLicenseID(rec, req)

	// Assert response is error
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var errorResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	require.NoError(t, err)
	require.NotEmpty(t, errorResponse)
}
