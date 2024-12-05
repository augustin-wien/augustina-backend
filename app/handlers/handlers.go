package handlers

import (
	"augustin/config"
	"augustin/keycloak"
	"augustin/utils"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"gopkg.in/guregu/null.v4"

	"github.com/mitchellh/mapstructure"

	"augustin/database"

	_ "github.com/swaggo/files"        // swagger embed files
	_ "github.com/swaggo/http-swagger" // http-swagger middleware

	"augustin/paymentprovider"
)

var log = utils.GetLogger()

// respond takes care of writing the response to the client
func respond(w http.ResponseWriter, err error, payload interface{}) {
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, payload)
	if err != nil {
		log.Error("respond: ", err)
	}
}

// HelloWorld godoc
//
//	@Summary		Return HelloWorld
//	@Description	Return HelloWorld as sample API call
//	@Tags			Core
//	@Accept			json
//	@Produce		json
//	@Router			/hello/ [get]
//
// HelloWorld API Handler fetching data from database
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	greeting, err := database.Db.GetHelloWorld()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, greeting)
	if err != nil {
		log.Error("HelloWorld: ", err)
	}
}

// HelloWorldAuth godoc
//
//	@Summary		Return HelloWorld
//	@Description	Return HelloWorld as sample API call
//	@Tags			Core
//	@Accept			json
//	@Produce		json
//	@Security		KeycloakAuth
//	@Router			/auth/hello/ [get]
//
// HelloWorld API Handler fetching data from database
func HelloWorldAuth(w http.ResponseWriter, r *http.Request) {
	greeting, err := database.Db.GetHelloWorld()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, greeting)
	if err != nil {
		log.Error("HelloWorldAuth: ", err)
	}
}

// Users ----------------------------------------------------------------------

type checkLicenseIDResponse struct {
	FirstName       string
	AccountProofUrl null.String
}

// CheckVendorsLicenseID godoc
//
//	 	@Summary 		Check for license id
//		@Description	Check if license id exists, return first name of vendor if it does
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//	    @Param		    licenseID path string true "License ID"
//		@Success		200	{string} checkLicenseIDResponse
//		@Response		200	{string} checkLicenseIDResponse
//		@Router			/vendors/check/{licenseID}/ [get]
func CheckVendorsLicenseID(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")
	if licenseID == "" {
		utils.ErrorJSON(w, errors.New("No licenseID provided under /vendors/check/{licenseID}/"), http.StatusBadRequest)
		return
	}

	users, err := database.Db.GetVendorByLicenseID(licenseID)
	if err != nil {
		utils.ErrorJSON(w, errors.New("Wrong license id. No vendor exists with this id"), http.StatusBadRequest)
		return
	}
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	response := checkLicenseIDResponse{
		FirstName:       users.FirstName,
		AccountProofUrl: users.AccountProofUrl,
	}
	if settings.UseVendorLicenseIdInShop {
		response.FirstName = licenseID
	}

	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("CheckVendorsLicenseID: ", err)
	}
}

// ListVendors godoc
//
//	 	@Summary 		List Vendors
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Security		KeycloakAuth
//		@Success		200	{array}	database.Vendor
//		@Router			/vendors/ [get]
func ListVendors(w http.ResponseWriter, r *http.Request) {
	vendors, err := database.Db.ListVendors()
	respond(w, err, vendors)
}

// CreateVendor godoc
//
//	 	@Summary 		Create Vendor
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//	    @Param		    data body database.Vendor true "Vendor Representation"
//		@Router			/vendors/ [post]
func CreateVendor(w http.ResponseWriter, r *http.Request) {
	var vendor database.Vendor
	err := utils.ReadJSON(w, r, &vendor)
	if err != nil {
		log.Error("CreateVendor: ReadJSON failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name") + " is creating a vendor for" + vendor.Email)

	// Create user in keycloak
	user, err := keycloak.KeycloakClient.GetOrCreateUser(vendor.Email)
	if err != nil {
		log.Error("CreateVendor: Create keycloak user failed ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info("Created user in keycloak: ", user)
	vendor.KeycloakID = user

	err = keycloak.KeycloakClient.AssignGroup(user, config.Config.KeycloakVendorGroup)
	if err != nil {
		log.Error("CreateVendor: Assigning user to vendor group failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	id, err := database.Db.CreateVendor(vendor)
	if err != nil {
		log.Error("CreateVendor: Create vendor in db failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, err, id)
}

// GetVendor godoc
//
//	 	@Summary 		Get Vendor
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//		@Param          id   path int  true  "Vendor ID"
//		@Router			/vendors/{id}/ [get]
func GetVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendor, err := database.Db.GetVendorWithBalanceUpdate(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, err, vendor)
}

type VendorOverview struct {
	ID           int
	FirstName    string
	LastName     string
	Email        string
	LicenseID    string
	UrlID        string
	LastPayout   null.Time `swaggertype:"string" format:"date-time"`
	Balance      int
	Address      string
	PLZ          string
	Location     string
	Telephone    string
	OpenPayments []database.Payment
}

// GetVendorOverview godoc
//
//	 	@Summary 		Get Vendor overview
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200 {object} VendorOverview
//		@Security		KeycloakAuth
//		@Router			/vendors/me/ [get]
func GetVendorOverview(w http.ResponseWriter, r *http.Request) {

	// Get vendors email from keycloak header
	vendorEmail := r.Header.Get("X-Auth-User-Email")
	if vendorEmail == "" {
		utils.ErrorJSON(w, fmt.Errorf("user has no email defined"), http.StatusBadRequest)
		return
	}

	// Get vendor information from database
	vendor, err := database.Db.GetVendorByEmail(vendorEmail)
	if err != nil {
		if err.Error() == "no rows in result set" {
			utils.ErrorJSON(w, fmt.Errorf("User is not a vendor"), http.StatusBadRequest)
			return
		}
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get open payments of vendor from database
	minDate := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Now()
	payments, err := database.Db.ListPaymentsForPayout(minDate, maxDate, vendor.LicenseID.String)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Create response
	response := VendorOverview{
		ID:           vendor.ID,
		FirstName:    vendor.FirstName,
		LastName:     vendor.LastName,
		Email:        vendor.Email,
		LicenseID:    vendor.LicenseID.String,
		UrlID:        vendor.UrlID,
		LastPayout:   vendor.LastPayout,
		Balance:      vendor.Balance,
		Address:      vendor.Address,
		PLZ:          vendor.PLZ,
		Location:     vendor.Location,
		Telephone:    vendor.Telephone,
		OpenPayments: payments,
	}

	// Return response
	respond(w, err, response)
}

// UpdateVendor godoc
//
//	 	@Summary 		Update Vendor
//		@Description	Warning: Unfilled fields will be set to default values
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//	    @Param          id   path int  true  "Vendor ID"
//		@Param		    data body database.Vendor true "Vendor Representation"
//		@Router			/vendors/{id}/ [put]
func UpdateVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("UpdateVendor: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name")+" is updating vendor with id: ", vendorID)
	var vendor database.Vendor
	err = utils.ReadJSON(w, r, &vendor)
	if err != nil {
		log.Error("UpdateVendor: ReadJSON failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	oldVendor, err := database.Db.GetVendor(vendorID)
	if err != nil {
		log.Error("UpdateVendor: "+fmt.Sprint(vendorID)+"failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	keycloakId, err := keycloak.KeycloakClient.UpdateVendor(oldVendor.Email, vendor.Email, vendor.LicenseID.String, vendor.FirstName, vendor.LastName)
	if err != nil {
		log.Error("UpdateVendor: update user in keycloak for "+fmt.Sprint(vendorID)+" failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendor.KeycloakID = keycloakId

	err = database.Db.UpdateVendor(vendorID, vendor)
	if err != nil {
		log.Error("UpdateVendor: update vendor in db for "+fmt.Sprint(vendorID)+" failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, err, vendor)
}

// DeleteVendor godoc
//
//		@Summary 		Delete Vendor
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//	    @Param          id   path int  true  "Vendor ID"
//		@Router			/vendors/{id}/ [delete]
func DeleteVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name")+" is deleting vendor with id: ", vendorID)
	vendor, err := database.Db.GetVendor(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Delete user in keycloak
	err = keycloak.KeycloakClient.DeleteUser(vendor.Email)
	if err != nil {
		log.Error("Deleting user "+vendor.Email+" failed in keycloak failed: ", err)
		// ignore because not each legacy vendor is in keycloak
	}

	err = database.Db.DeleteVendor(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func UpdateVendorByLicenseID(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")
	if licenseID == "" {
		utils.ErrorJSON(w, errors.New("No licenseID provided under /vendors/license/{licenseID}/"), http.StatusBadRequest)
		return
	}
	vendor, err := database.Db.GetVendorByLicenseID(licenseID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var updatedVendor database.Vendor
	err = utils.ReadJSON(w, r, &updatedVendor)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	keycloakId, err := keycloak.KeycloakClient.UpdateVendor(vendor.Email, updatedVendor.Email, vendor.LicenseID.String, updatedVendor.FirstName, updatedVendor.LastName)
	if err != nil {
		log.Error("UpdateVendor: update user in keycloak for "+fmt.Sprint(vendor.ID)+" failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendor.KeycloakID = keycloakId
	err = database.Db.UpdateVendor(vendor.ID, updatedVendor)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name") + " is updating vendor via flour with license id: " + licenseID)
	respond(w, err, vendor)
}

func GetVendorByLicenseID(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")
	if licenseID == "" {
		utils.ErrorJSON(w, errors.New("No licenseID provided under /vendors/license/{licenseID}/"), http.StatusBadRequest)
		return
	}
	vendor, err := database.Db.GetVendorByLicenseID(licenseID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, err, vendor)
}

// Items (that can be sold) ---------------------------------------------------

// ListItems godoc
//
//	 	@Summary 		List Items
//		@Tags			Items
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	database.Item
//		@Router			/items/ [get]
func ListItems(w http.ResponseWriter, r *http.Request) {
	items, err := database.Db.ListItems(true, true)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, items)
	if err != nil {
		log.Error("ListItems", err)
	}
}

// ListItemsBackoffice godoc
//
//	 	@Summary 		List Items for backoffice overview
//		@Tags			Items
//		@Accept			json
//		@Produce		json
//	    @Param			skipHiddenItems query bool false "No donation and transaction cost items"
//		@Param 			skipLicenses query bool false "No license items"
//		@Success		200	{array}	database.Item
//		@Security		KeycloakAuth
//		@Router			/items/backoffice [get]
func ListItemsBackoffice(w http.ResponseWriter, r *http.Request) {

	// Get filter parameters
	skipHiddenItemsRaw := r.URL.Query().Get("skipHiddenItems")
	skipLicensesRaw := r.URL.Query().Get("skipLicenses")

	// Parse filter parameters
	skipHiddenItems, err := parseBool(skipHiddenItemsRaw)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	skipLicenses, err := parseBool(skipLicensesRaw)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	items, err := database.Db.ListItems(skipHiddenItems, skipLicenses)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, items)
	if err != nil {
		log.Error("ListItemsBackoffice: ", err)
	}
}

// CreateItem godoc
//
//	 	@Summary 		Create Item
//		@Tags			Items
//		@Accept			json
//		@Produce		json
//	    @Param		    data body database.Item true "Item Representation"
//		@Success		200	 {integer}	id
//		@Security		KeycloakAuth
//		@Router			/items/ [post]
func CreateItem(w http.ResponseWriter, r *http.Request) {
	// Read multipart form
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Error("CreateItem: failed to parse Multipartform ", err)
		return
	}

	mForm := r.MultipartForm
	if mForm == nil {
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}

	// Handle normal fields
	item, err := updateItemNormal(mForm.Value)
	if err != nil {
		log.Error("CreateItem: updateItemNormal failed ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Handle image field
	path, _ := updateItemImage(w, r)
	if path != "" {
		item.Image = path
	}

	// Handle pdf field
	pdfId, err := handleItemPDF(w, r)
	if err != nil {
		log.Error("CreateItem: handleItemPDF failed ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if pdfId != 0 {
		item.PDF = null.IntFrom(pdfId)
	}

	// Save item to database
	id, err := database.Db.CreateItem(item)
	if err != nil {
		log.Error("CreateItem: Database call create item failed", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, id)
	if err != nil {
		log.Error("CreateItem: WriteJSON failed", err)
	}
}

func updateItemImage(w http.ResponseWriter, r *http.Request) (path string, err error) {
	// Get file from image field
	file, header, err := r.FormFile("Image")
	if err != nil {
		return // No file passed, which is ok
	}
	defer file.Close()

	// Debugging
	name := strings.Split(header.Filename, ".")
	if len(name) < 2 {
		log.Error("updateItemImage: image name is wrong")
		utils.ErrorJSON(w, errors.New("invalid filename"), http.StatusBadRequest)
		return
	}

	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, file); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	dir, err := os.Getwd()
	if err != nil {
		log.Error("updateItemImage: ", err)
	}
	// Generate unique filename
	i := 0
	for {
		path = "img/" + name[0] + "_" + strconv.Itoa(i) + "." + name[len(name)-1]
		_, err = os.Stat(dir + "/" + path)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		i++
		if i > 1000 {
			log.Error("updateItemImage: too many files with same name", err)
			utils.ErrorJSON(w, errors.New("too many files with same name"), http.StatusBadRequest)
			return
		}
	}
	// current file path from os

	// Save file with unique name
	err = os.WriteFile(dir+"/"+path, buf.Bytes(), 0666)
	if err != nil {
		log.Error("updateItemImage: failed to write file", err)
	}
	return
}

func handleItemPDF(w http.ResponseWriter, r *http.Request) (pdfId int64, err error) {
	pdfId = -1
	// Check if a digit is sent instead of pdf
	// contentType := r.Header.Get("Content-Type")
	// if contentType != "multipart/form-data" {
	// 	log.Info("handleItemPDF: Content-Type is not multipart/form-data", contentType)
	// 	// Assuming that if it's not multipart/form-data, it's a digit being sent
	// 	// Handle the case where a digit is sent
	// 	pdfId, err = strconv.ParseInt(r.FormValue("pdfId"), 10, 64)
	// 	if err != nil {
	// 		log.Error("handleItemPDF: parsing the id failed", err)
	// 	}
	// 	return pdfId, nil
	// }

	// Get file from pdf field
	file, header, err := r.FormFile("PDF")
	if err != nil {
		return pdfId, nil // No file passed, which is ok
	}
	defer file.Close()

	// Debugging
	name := strings.Split(header.Filename, ".")
	if len(name) < 2 {
		log.Error("handleItemPDF: pdf name is wrong")
		utils.ErrorJSON(w, errors.New("invalid filename"), http.StatusBadRequest)
		return
	}

	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, file); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	// create base dir if not exist

	dir, err := os.Getwd()
	if err != nil {
		log.Error("UploadPDF: Failed to get current directory ", err)
		return
	}
	// Add a human readable timestamp to the filename to make it unique
	timeStamp := time.Now().Format("2006-01-02_15-04-05")
	path := "pdf/" + timeStamp + "_" + name[0] + "." + name[len(name)-1]
	_, err = os.Stat(dir + "/pdf")
	if errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(dir+"/pdf", 0777)
		if err != nil {
			log.Error("handleItemPDF: failed to create directory", err)
			return
		}
	}
	err = os.WriteFile(dir+"/"+path, buf.Bytes(), 0666)
	if err != nil {
		log.Error("handleItemPDF: failed to write file", err)
		return
	}
	pdf := database.PDF{
		Path:      path,
		Timestamp: time.Now(),
	}

	pdfId, err = database.Db.CreatePDF(pdf)
	if err != nil {
		log.Error("handleItemPDF: failed to create db entry", err)
	}
	return
}

// fields := mForm.Value               // Values are stored in []string
func updateItemNormal(fields map[string][]string) (item database.Item, err error) {
	fieldsClean := make(map[string]any) // Values are stored in string
	for key, value := range fields {
		if key == "Price" {
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse Price failed ", err)
				return
			}
		} else if key == "IsLicenseItem" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse IfLicenseItem failed ", err)
				return
			}
		} else if key == "IsPDFItem" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse IsPDFItem failed ", err)
				return
			}
		} else if key == "Archived" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse Archived failed ", err)
				return
			}
		} else if key == "LicenseItem" {
			licensitem, err := strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse LicenseItem failed ", err)
				return item, err
			}
			fieldsClean[key] = null.IntFrom(int64(licensitem))

		} else if key == "ID" {
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse ID failed ", err)
				return
			}
		} else if key == "ItemColor" {
			fieldsClean[key] = null.StringFrom(value[0])
		} else if key == "ItemTextColor" {
			fieldsClean[key] = null.StringFrom(value[0])
		} else if key == "PDF" {
			pdf, err := strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse PDF failed ", err)
				return item, err
			}
			fieldsClean[key] = null.IntFrom(int64(pdf))
		} else if key == "LicenseGroup" {
			fieldsClean[key] = null.StringFrom(value[0])
		} else if key == "ItemOrder" {
			fieldsClean[key], err = strconv.Atoi(value[0])
		} else {
			fieldsClean[key] = value[0]
		}
	}
	err = mapstructure.Decode(fieldsClean, &item)
	if err != nil {
		log.Error("updateItemNormal: Decoding fields failed", err)
		return
	}
	return
}

// UpdateItem godoc
//
//	 	@Summary 		Update Item
//		@Description	Requires multipart form (for image)
//		@Tags			Items
//		@Accept			json
//		@Produce		json
//		@Param			id path int true "Item ID"
//	    @Param		    data body database.Item true "Item Representation"
//		@Success		200
//		@Security		KeycloakAuth
//		@Router			/items/{id}/ [put]
//
// UpdateItem requires a multipart form
// https://www.sobyte.net/post/2022-03/go-multipart-form-data/
func UpdateItem(w http.ResponseWriter, r *http.Request) {
	ItemID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("UpdateItem: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Security check to disable updating Item of ID 2 and 3, which are essential for donations and transaction costs
	if ItemID == 2 || ItemID == 3 {
		utils.ErrorJSON(w, errors.New("Nice try! You are not allowed to update this item"), http.StatusBadRequest)
		return
	}

	// Read multipart form
	log.Info("Request Content Length:", r.ContentLength)
	err = r.ParseMultipartForm(64 << 20)
	log.Info("Parsed Form Content Length:", r.ContentLength)
	if err != nil {
		log.Error("UpdateItem: ParseMultipartForm failed ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	mForm := r.MultipartForm
	if mForm == nil {
		log.Error("UpdateItem: ", errors.New("invalid form"))
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}

	// Handle normal fields
	item, err := updateItemNormal(mForm.Value)
	if err != nil {
		log.Error("UpdateItem: UpdateItemNormal failed ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	path, _ := updateItemImage(w, r)
	if path != "" {
		log.Info("UpdateItem: Path is empty", path)
		item.Image = path
	}

	pdfId, err := handleItemPDF(w, r)
	if err != nil {
		log.Error("CreateItem: handleItemPDF failed ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if pdfId != -1 {
		item.PDF = null.IntFrom(pdfId)
	}

	// Save item to database
	err = database.Db.UpdateItem(ItemID, item)
	if err != nil {
		log.Error("UpdateItem: db update", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
	}
	err = utils.WriteJSON(w, http.StatusOK, err)
	if err != nil {
		log.Error("UpdateItem: ", err)
	}
}

// DeleteItem godoc
//
//		 	@Summary 		Delete Item
//			@Tags			Items
//			@Accept			json
//			@Produce		json
//			@Success		200
//			@Security		KeycloakAuth
//	     	@Param          id   path int  true  "Item ID"
//			@Router			/items/{id}/ [delete]
func DeleteItem(w http.ResponseWriter, r *http.Request) {
	ItemID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.DeleteItem(ItemID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Orders ---------------------------------------------------------------------

type createOrderRequestEntry struct {
	Item     int
	Quantity int
}

type createOrderRequest struct {
	Entries         []createOrderRequestEntry
	User            string
	VendorLicenseID string
	CustomerEmail   null.String
}

type createOrderResponse struct {
	SmartCheckoutURL string
}

// PaymentOrders ---------------------------------------------------------------------

// hasDuplicitValues checks if a map has duplicate values
// Credit to: https://stackoverflow.com/a/57237165/19932351
func hasDuplicitValues(m map[int]int) bool {
	// Create empty map
	x := make(map[int]struct{})

	// Iterate over map
	for _, v := range m {
		// Add value to map by using it as key
		if _, has := x[v]; has {
			// Return true if value is already in map
			return true
		}
		// Add empty struct to map
		x[v] = struct{}{}
	}

	return false
}

// CreatePaymentOrder godoc
//
//	 	@Summary 		Create Payment Order
//		@Description	Submits payment order to provider & saves it to database. Entries need to have an item id and a quantity (for entries without a price like tips, the quantity is the amount of cents). If no user is given, the order is anonymous.
//		@Tags			Orders
//		@Accept			json
//		@Produce		json
//	    @Param		    data body createOrderRequest true "Payment Order"
//		@Success		200 {object} createOrderResponse
//		@Router			/orders/ [post]
func CreatePaymentOrder(w http.ResponseWriter, r *http.Request) {

	// Read payment order from request
	var requestData createOrderRequest
	var order database.Order
	err := utils.ReadJSON(w, r, &requestData)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Security checks for entries
	for _, entry := range requestData.Entries {

		// 1. Check: Quantity has to be > 0 for any item except donation
		if entry.Quantity <= 0 && entry.Item != 2 {
			utils.ErrorJSON(w, errors.New("Nice try! Quantity has to be greater than 0"), http.StatusBadRequest)
			return
		}

		// 2. Check: All items have to exist
		item, err := database.Db.GetItem(entry.Item)
		if err != nil {
			utils.ErrorJSON(w, errors.New("Nice try! Item does not exist"), http.StatusBadRequest)
			return
		}

		// 3. Check: Transaction costs (id == 3) are not allowed to be in entries
		if entry.Item == 3 {
			utils.ErrorJSON(w, errors.New("Nice try! You are not allowed to purchase this item"), http.StatusBadRequest)
			return
		}

		// 4. Check: If there is a item that needs a customerEmail, the user has to be given

		if item.LicenseItem.Valid {
			if !requestData.CustomerEmail.Valid || requestData.CustomerEmail.String == "" {
				utils.ErrorJSON(w, errors.New("you are not allowed to purchase this item without a customer email"), http.StatusBadRequest)
				return
			}
			order.CustomerEmail = requestData.CustomerEmail
		}
	}

	// 5. Check: If there is more than one entry, each item id has to be unique
	if len(requestData.Entries) > 1 {
		// Create map with item ids as keys
		uniqueItemIDs := make(map[int]int)
		for idx, entry := range requestData.Entries {
			uniqueItemIDs[idx] = entry.Item
		}
		// Check if there are duplicate item ids
		if hasDuplicitValues(uniqueItemIDs) {
			utils.ErrorJSON(w, errors.New("Nice try! You are not supposed to have duplicate item ids in your order request"), http.StatusBadRequest)
			return
		}
	}

	// 6. Check: If item 2 (donation) is ordered without another item
	if len(requestData.Entries) == 1 && requestData.Entries[0].Item == 2 {
		// Throw error
		utils.ErrorJSON(w, errors.New("Nice try! You are not allowed to purchase this item without another item"), http.StatusBadRequest)
		return
	}

	// Create slice of order entries depending on size of requestData.Entries
	order.Entries = make([]database.OrderEntry, len(requestData.Entries))

	// Add entries to each ordered item
	for idx, entry := range requestData.Entries {
		order.Entries[idx].Item = entry.Item
		order.Entries[idx].Quantity = entry.Quantity
	}

	// Get vendor id from license id
	vendor, err := database.Db.GetVendorByLicenseID(requestData.VendorLicenseID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	order.Vendor = vendor.ID

	var settings database.Settings
	if settings, err = database.Db.GetSettings(); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Add user to order
	// TODO-Question: This line is not necessary anymore, since the user is already in the request?
	order.User.String = requestData.User

	// Get accounts
	var buyerAccountID int
	authenticatedUserID := r.Header.Get("X-Auth-User-Name")
	if authenticatedUserID != "" {
		buyerAccount, err := database.Db.GetOrCreateAccountByUserID(authenticatedUserID)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		buyerAccountID = buyerAccount.ID
	} else {
		buyerAccountID, err = database.Db.GetAccountTypeID("UserAnon")
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}

	vendorAccount, err := database.Db.GetAccountByVendorID(order.Vendor)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	orgaAccount, err := database.Db.GetAccountByType("Orga")
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Amount of added license items
	// Since for each license item an additional entry is added,
	// Therefore, the index of the for loop has to be increased by 1
	licenseItemAdded := 0

	// Extend order entries
	for idx, entry := range order.Entries {
		// Increase index depending on how many license items were added
		idx = idx + licenseItemAdded
		// Get item from database
		item, err := database.Db.GetItem(entry.Item)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		// Define flow of money from buyer to vendor
		order.Entries[idx].Sender = buyerAccountID
		order.Entries[idx].Receiver = vendorAccount.ID
		order.Entries[idx].Price = item.Price // Take current item price
		order.Entries[idx].IsSale = true      // Will be used for sales payment

		// If there is a license item, prepend it before the actual item
		if item.LicenseItem.Valid {
			// Get license item from database
			licenseItem, err := database.Db.GetItem(int(item.LicenseItem.Int64))
			if err != nil {
				utils.ErrorJSON(w, err, http.StatusBadRequest)
				return
			}
			// Define flow of money from vendor to orga
			licenseItemEntry := database.OrderEntry{
				Item:         int(item.LicenseItem.Int64),
				Quantity:     entry.Quantity,
				Price:        licenseItem.Price,
				Sender:       vendorAccount.ID,
				Receiver:     orgaAccount.ID,
				SenderName:   vendorAccount.Name,
				ReceiverName: orgaAccount.Name,
			}
			// Prepend license item without overwriting next entries
			order.Entries = append([]database.OrderEntry{licenseItemEntry}, order.Entries...)
			// Add customer email to order
			order.CustomerEmail = requestData.CustomerEmail
			// Increase licenseItemAdded by one
			licenseItemAdded++

		}

	}
	// ignore MaxOrderAmount if its 0
	if settings.MaxOrderAmount != 0 && order.GetTotal() >= settings.MaxOrderAmount {
		utils.ErrorJSON(w, errors.New("Order amount is too high"), http.StatusBadRequest)
		return
	}
	// Submit order to vivawallet (disabled in tests)
	var OrderCode int
	if database.Db.IsProduction {
		accessToken, err := paymentprovider.AuthenticateToVivaWallet()
		if err != nil {
			log.Error("Authentication failed: ", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		OrderCode, err = paymentprovider.CreatePaymentOrder(accessToken, order, requestData.VendorLicenseID)
		if err != nil {
			log.Errorf("Creating payment order failed for %+v with order id %+v failed", requestData.VendorLicenseID, order.ID, err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}

	// Save order to database
	order.OrderCode.String = strconv.Itoa(OrderCode)
	order.OrderCode.Valid = true // This means that it is not null
	_, err = database.Db.CreateOrder(order)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Check if VivaWalletSmartCheckoutURL is set
	if config.Config.VivaWalletSmartCheckoutURL == "" {
		utils.ErrorJSON(w, errors.New("VivaWalletSmartCheckoutURL is not set"), http.StatusBadRequest)
		return
	}

	// Create response
	checkoutURL := config.Config.VivaWalletSmartCheckoutURL + strconv.Itoa(OrderCode)

	// Add color code to URL
	if settings.Color == "" {
		log.Info("Color code is not set")
	} else {

		var colorCode string
		// Check if color code is valid with # at the beginning
		if settings.Color[0] == '#' {
			// Remove # from color code due to VivaWallet's policy
			colorCode = settings.Color[1:]
		} else {
			log.Info("Color code is not valid: ", settings.Color)
		}
		// Make color code lowercase
		colorCode = strings.ToLower(colorCode)

		// Add color code and necessary attachment to URL
		colorCodeAttachment := fmt.Sprintf("%s%s", "&color=", colorCode)

		// Add color code to URL
		checkoutURL = fmt.Sprintf("%s%s", checkoutURL, colorCodeAttachment)
	}

	response := createOrderResponse{
		SmartCheckoutURL: checkoutURL,
	}
	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("CreatePaymentOrder: ", err)
	}
}

// VerifyPaymentOrderResponse is the response to VerifyPaymentOrder
type VerifyPaymentOrderResponse struct {
	TimeStamp        time.Time
	FirstName        string
	PurchasedItems   []database.OrderEntry
	TotalSum         int
	PDFDownloadLinks *[]database.PDFDownloadLinks
}

// VerifyPaymentOrder godoc
//
//	 	@Summary 		Verify Payment Order
//		@Description	Verifies order and creates payments
//		@Tags			Orders
//		@Accept			json
//		@Produce		json
//		@Success		200 {object} VerifyPaymentOrderResponse
//		@Param			s query string true "Order Code" Format(3043685539722561)
//		@Param			t query string true "Transaction ID" Format(882d641c-01cc-442f-b894-2b51250340b5)
//		@Router			/orders/verify/ [get]
func VerifyPaymentOrder(w http.ResponseWriter, r *http.Request) {

	// Get transaction ID from URL parameter
	OrderCode := r.URL.Query().Get("s")
	if OrderCode == "" {
		utils.ErrorJSON(w, errors.New("missing parameter s"), http.StatusBadRequest)
		return
	}
	TransactionID := r.URL.Query().Get("t")
	if TransactionID == "" {
		utils.ErrorJSON(w, errors.New("missing parameter t"), http.StatusBadRequest)
		return
	}

	// Get payment order from database
	order, err := database.Db.GetOrderByOrderCode(OrderCode)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	if database.Db.IsProduction && !config.Config.Development {
		// Verify transaction
		_, err := paymentprovider.VerifyTransactionID(TransactionID, true)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}

	if config.Config.Development {
		// Verify transaction
		err = database.Db.VerifyOrderAndCreatePayments(order.ID, 0)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}

	// Make sure that transaction timestamp is not older than 15 minutes (900 seconds) to time.Now()
	if time.Since(order.Timestamp) > 900*time.Second {
		utils.ErrorJSON(w, errors.New("Transaction timestamp is older than 15 minutes"), http.StatusBadRequest)
		return
	}

	var verifyPaymentOrderResponse VerifyPaymentOrderResponse

	// Declare timestamp from order
	verifyPaymentOrderResponse.TimeStamp = order.Timestamp

	for _, entry := range order.Entries {
		if entry.IsSale {
			verifyPaymentOrderResponse.PurchasedItems = append(verifyPaymentOrderResponse.PurchasedItems, entry)
		} else {
			continue
		}
	}

	// Declare total sum from order
	verifyPaymentOrderResponse.TotalSum = order.GetTotal()
	verifyPaymentOrderResponse.PDFDownloadLinks = order.GetPDFDownloadLinks()

	// Get first name of vendor from vendor id in order
	vendor, err := database.Db.GetVendor(order.Vendor)
	if err != nil {
		log.Error("Getting vendor's first name failed: ", err)
		return
	}
	// Declare first name from vendor
	verifyPaymentOrderResponse.FirstName = vendor.FirstName

	// Create response
	err = utils.WriteJSON(w, http.StatusOK, verifyPaymentOrderResponse)
	if err != nil {
		log.Error("VerifyPaymentOrder: ", err)
	}
}

// Payments (from one account to another account) -----------------------------

func parseBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	return strconv.ParseBool(value)
}

// ListPaymentsForPayout godoc
//
//	 	@Summary 		Get list of all payments for payout
//		@Description 	Payments that do not have an associated payout
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Param			vendor query string false "Vendor LicenseID"
//		@Success		200	{array}	database.Payment
//		@Security		KeycloakAuth
//		@Security		KeycloakAuth
//		@Router			/payments/forpayout/ [get]
func ListPaymentsForPayout(w http.ResponseWriter, r *http.Request) {
	var err error
	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")
	vendor := r.URL.Query().Get("vendor")
	var minDate, maxDate time.Time
	if minDateRaw != "" {
		minDate, err = time.Parse(time.RFC3339, minDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}
	if maxDateRaw != "" {
		maxDate, err = time.Parse(time.RFC3339, maxDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}
	payments, err := database.Db.ListPaymentsForPayout(minDate, maxDate, vendor)
	respond(w, err, payments)
}

// ListPayments godoc
//
//		 	@Summary 		Get list of all payments
//			@Description 	Filter by date, vendor, payouts, sales. If payouts set true, all payments are removed that are not payouts. Same for sales. So sales and payouts can't be true at the same time.
//			@Tags			Payments
//			@Accept			json
//			@Produce		json
//			@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//			@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//			@Param			vendor query string false "Vendor LicenseID"
//	     @Param			payouts query bool false "Payouts only"
//	     @Param          sales query bool false "Sales only"
//			@Success		200	{array}	database.Payment
//			@Security		KeycloakAuth
//			@Security		KeycloakAuth
//			@Router			/payments/ [get]
func ListPayments(w http.ResponseWriter, r *http.Request) {
	var err error

	// Get filter parameters
	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")
	payoutRaw := r.URL.Query().Get("payouts")
	salesRaw := r.URL.Query().Get("sales")
	vendor := r.URL.Query().Get("vendor")

	// Parse filter parameters
	payout, err := parseBool(payoutRaw)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	sales, err := parseBool(salesRaw)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var minDate, maxDate time.Time
	if minDateRaw != "" {
		minDate, err = time.Parse(time.RFC3339, minDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}
	if maxDateRaw != "" {
		maxDate, err = time.Parse(time.RFC3339, maxDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}

	// Get payments with filter parameters
	payments, err := database.Db.ListPayments(minDate, maxDate, vendor, payout, sales, false)
	respond(w, err, payments)
}

type ItemStatistics struct {
	ID          int
	Name        string
	SumAmount   int
	SumQuantity int
}

// PaymentsStatistics is the response to ListPaymentsStatistics
type PaymentsStatistics struct {
	From  time.Time
	To    time.Time
	Items []ItemStatistics
}

// ListPaymentsStatistics godoc
//
//	 	@Summary 		Calculate statistics of items & payments
//		@Description 	Filter by date, get statistical information, sorted by item.
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Success		200	{array}	PaymentsStatistics
//		@Security		KeycloakAuth
//		@Security		KeycloakAuth
//		@Router			/payments/statistics/ [get]
func ListPaymentsStatistics(w http.ResponseWriter, r *http.Request) {
	var err error

	// Get filter parameters
	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")

	// Parse filter parameters
	var minDate, maxDate time.Time
	if minDateRaw != "" {
		minDate, err = time.Parse(time.RFC3339, minDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}
	if maxDateRaw != "" {
		maxDate, err = time.Parse(time.RFC3339, maxDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}

	// Get items
	items, err := database.Db.ListItems(false, false)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get payments with filter parameters
	payments, err := database.Db.ListPayments(minDate, maxDate, "", false, false, false)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Create map of items
	itemsMap := make(map[int]ItemStatistics)
	for _, item := range items {
		itemsMap[item.ID] = ItemStatistics{
			ID:          item.ID,
			Name:        item.Name,
			SumAmount:   0,
			SumQuantity: 0,
		}
	}

	// Create sums per item
	for _, payment := range payments {
		if !payment.Item.Valid {
			continue
		}
		itemID := int(payment.Item.Int64)
		if entry, ok := itemsMap[itemID]; ok {
			entry.SumAmount += payment.Amount
			entry.SumQuantity += payment.Quantity
			itemsMap[itemID] = entry
		} else {
			utils.ErrorJSON(w, errors.New("item not found"), http.StatusBadRequest)
			return
		}
	}

	// Create payment statistics
	var paymentsStatistics PaymentsStatistics
	paymentsStatistics.From = minDate
	paymentsStatistics.To = maxDate
	for _, item := range itemsMap {
		paymentsStatistics.Items = append(paymentsStatistics.Items, item)
	}

	respond(w, err, paymentsStatistics)
}

// CreatePayment godoc
//
//	 	@Summary 		Create a payment
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			amount body database.Payment true " Create Payment"
//		@Success		200
func CreatePayment(w http.ResponseWriter, r *http.Request) {
	var payment database.Payment
	err := utils.ReadJSON(w, r, &payment)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	paymentID, err := database.Db.CreatePayment(payment)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, paymentID)
	if err != nil {
		log.Error("CreatePayment: ", err)
	}
}

type createPaymentsRequest struct {
	Payments []database.Payment
}

// CreatePayments godoc
//
//	 	@Summary 		Create a set of payments
//		@Description 	TODO: This handler is not working right now and to be done for manually setting payments
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			amount body createPaymentsRequest true "Create Payment"
//		@Success		200 {integer} id
//		@Security		KeycloakAuth
//		@Router			/payments/ [post]
func CreatePayments(w http.ResponseWriter, r *http.Request) {
	var paymentBatch createPaymentsRequest
	err := utils.ReadJSON(w, r, &paymentBatch)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.CreatePayments(paymentBatch.Payments)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
}

type createPaymentPayoutRequest struct {
	VendorLicenseID string
	From            time.Time
	To              time.Time
}

// CreatePaymentPayout godoc
//
//	 	@Summary 		Create a payment from a vendor account to cash
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			amount body createPaymentPayoutRequest true "Create Payment"
//		@Success		200 {integer} id
//		@Security		KeycloakAuth
//		@Router			/payments/payout/ [post]
func CreatePaymentPayout(w http.ResponseWriter, r *http.Request) {

	// Read data from request
	var payoutData createPaymentPayoutRequest
	err := utils.ReadJSON(w, r, &payoutData)
	if err != nil {
		log.Error("CreatePaymentPayout: parse JSON ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get vendor
	vendor, err := database.Db.GetVendorByLicenseID(payoutData.VendorLicenseID)
	if err != nil {
		log.Error("CreatePaymentPayout: get vendor ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get vendor account
	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	if err != nil {
		log.Error("CreatePaymentPayout: get vendor account ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get amount of money for payout
	paymentsToBePaidOut, err := database.Db.ListPaymentsForPayout(payoutData.From, payoutData.To, payoutData.VendorLicenseID)
	if err != nil {
		log.Error("CreatePaymentPayout: list payments for payout ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	var amount int
	for _, payment := range paymentsToBePaidOut {
		if payment.Receiver == vendorAccount.ID {
			amount += payment.Amount
		}
		if payment.Sender == vendorAccount.ID {
			amount -= payment.Amount
		}
	}

	// Check that amount is bigger than 0
	if amount <= 0 {
		utils.ErrorJSON(w, errors.New("payout amount must be bigger than 0"), http.StatusBadRequest)
		return
	}

	// Check if vendor has enough money
	if vendorAccount.Balance < amount {
		log.Error("CreatePaymentPayout: payout amount bigger than vendor account balance")
		utils.ErrorJSON(w, errors.New("payout amount bigger than vendor account balance"), http.StatusBadRequest)
		return
	}

	// Get authenticated user
	authenticatedUserID := r.Header.Get("X-Auth-User-Name")

	// Execute payout
	paymentID, err := database.Db.CreatePaymentPayout(vendor, vendorAccount.ID, authenticatedUserID, amount, paymentsToBePaidOut)
	if err != nil {
		log.Error("CreatePaymentPayout: db", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Return success with paymentID
	err = utils.WriteJSON(w, http.StatusOK, paymentID)
	if err != nil {
		log.Error("CreatePaymentPayout: finish:", err)
	}
	log.Infof("Payout of %d cents for vendor %s was successful", amount, vendor.LicenseID)

}

type webhookResponse struct {
	Status string
}

// VivaWalletWebhookSuccess godoc
//
//	@Summary		Webhook for VivaWallet successful transaction
//	@Description	Webhook for VivaWallet successful transaction
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200
//	@Param			data body paymentprovider.TransactionSuccessRequest true "Payment Successful Response"
//	@Router			/webhooks/vivawallet/success/ [post]
func VivaWalletWebhookSuccess(w http.ResponseWriter, r *http.Request) {

	// Message to console that handler was entered
	log.Info("Transaction Success Webhook entered")

	var paymentSuccessful paymentprovider.TransactionSuccessRequest
	err := utils.ReadJSON(w, r, &paymentSuccessful)
	if err != nil {
		log.Info("Reading JSON failed for webhook: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = paymentprovider.HandlePaymentSuccessfulResponse(paymentSuccessful)
	if err != nil {
		log.Error("VivaWalletWebhookSuccess: handle payment failed: ", err)
		return
	}

	var response webhookResponse
	response.Status = "OK"

	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("VivaWalletWebhookSuccess: write json: ", err)
	}
}

// VivaWalletWebhookFailure godoc
//
//	@Summary		Webhook for VivaWallet failed transaction
//	@Description	Webhook for VivaWallet failed transaction
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200
//	@Param			data body paymentprovider.TransactionSuccessRequest true "Payment Failure Response"
//	@Router			/webhooks/vivawallet/failure/ [post]
func VivaWalletWebhookFailure(w http.ResponseWriter, r *http.Request) {
	var paymentFailure paymentprovider.TransactionSuccessRequest
	err := utils.ReadJSON(w, r, &paymentFailure)
	if err != nil {
		log.Info("Reading JSON failed for webhook: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = paymentprovider.HandlePaymentFailureResponse(paymentFailure)
	if err != nil {
		log.Error("VivaWalletWebhookFailure: ", err)
		return
	}

	var response webhookResponse
	response.Status = "OK"

	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("VivaWalletWebhookFailure: ", err)
	}
}

// VivaWalletWebhookPrice godoc
//
//	@Summary		Webhook for VivaWallet transaction prices
//	@Description	Webhook for VivaWallet transaction prices
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200
//	@Param			data body paymentprovider.TransactionPriceRequest true "Payment Price Response"
//	@Router			/webhooks/vivawallet/price/ [post]
func VivaWalletWebhookPrice(w http.ResponseWriter, r *http.Request) {

	// Message to console that handler was entered
	log.Info("Transaction Price Webhook entered")

	var paymentPrice paymentprovider.TransactionPriceRequest
	err := utils.ReadJSON(w, r, &paymentPrice)
	if err != nil {
		log.Info("Reading JSON failed for webhook: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = paymentprovider.HandlePaymentPriceResponse(paymentPrice)
	if err != nil {
		log.Error("VivaWalletWebhookPrice: ", err)
		return
	}

	var response webhookResponse
	response.Status = "OK"

	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("VivaWalletWebhookPrice: ", err)
	}
}

// VivaWalletVerificationKey godoc
//
//	@Summary		Return VivaWallet verification key
//	@Description	Return VivaWallet verification key
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200	{array}	paymentprovider.VivaWalletVerificationKeyResponse
//	@Router			/webhooks/vivawallet/price/ [get]
//	@Router 		/webhooks/vivawallet/success/ [get]
//	@Router 		/webhooks/vivawallet/failure/ [get]
func VivaWalletVerificationKey(w http.ResponseWriter, r *http.Request) {
	key := config.Config.VivaWalletVerificationKey
	if key == "" {
		log.Error("VIVA_WALLET_VERIFICATION_KEY not set or can't be found")
		utils.ErrorJSON(w, errors.New("VIVA_WALLET_VERIFICATION_KEY not set or can't be found"), http.StatusBadRequest)
		return
	}
	response := paymentprovider.VivaWalletVerificationKeyResponse{Key: key}
	err := utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("VivaWalletVerificationKey: ", err)
	}
}

type KeycloakSettings struct {
	Realm string
	URL   string
}
type ExtendedSettings struct {
	database.Settings
	Keycloak KeycloakSettings
}

// Settings -------------------------------------------------------------------

// getSettings godoc
//
//	 	@Summary 		Return settings
//		@Description	Return configuration data of the system
//		@Tags			Core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	database.Settings
//		@Router			/settings/ [get]
func getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	exSettings := ExtendedSettings{
		Settings: settings,
		Keycloak: KeycloakSettings{
			Realm: config.Config.KeycloakRealm,
			URL:   config.Config.KeycloakHostname,
		},
	}
	err = utils.WriteJSON(w, http.StatusOK, exSettings)
	if err != nil {
		log.Error("getSettings: ", err)
	}
}

type Imagetype string

const (
	ImagetypeLogo    Imagetype = "Logo"
	ImagetypeFavicon Imagetype = "Favicon"
	ImagetypeQrCode  Imagetype = "QRCodeLogoImgUrl"
)

func updateSettingsImg(w http.ResponseWriter, r *http.Request, fileType Imagetype) (path string, err error) {

	// Get file from image field
	file, header, err := r.FormFile(string(fileType))
	if err != nil {
		// Do not return error, as not passing a file is ok
		// Could be improved by differentiating between not passed and invalid file
		err = nil
		return
	}
	defer file.Close()

	// Debugging
	name := strings.Split(header.Filename, ".")
	if len(name) < 2 {
		log.Error("updateSettingsLogo: file name to short", err)
		utils.ErrorJSON(w, errors.New("invalid filename"), http.StatusBadRequest)
		return
	}
	if name[len(name)-1] != "png" {
		log.Error("updateSettingsLogo: wrong file ending: ", name[1])
		utils.ErrorJSON(w, errors.New("file type must be png"), http.StatusBadRequest)
		return
	}

	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, file); err != nil {
		log.Error("updateSettingsLogo: copying file failed", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Save file with name "logo"
	switch fType := fileType; fType {
	case "Logo":
		path = "/img/logo.png"
	case "Favicon":
		path = "/img/favicon.png"
	case "QRCodeLogoImgUrl":
		path = "/img/qrcode.png"
	}
	dir, err := os.Getwd()
	if err != nil {
		log.Error("updateSettingsLogo: couldn't get wd", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = os.WriteFile(dir+"/"+path, buf.Bytes(), 0666)
	if err != nil {
		log.Error("updateSettingsLogo: saving failed", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
	}
	log.Info("updateSettingsLogo: saved file to ", dir+path)
	return
}

var updateSettingsMutex sync.Mutex

// updateSettings godoc
//
//	 	@Summary 		Update settings
//		@Description	Update configuration data of the system. Requires multipart form. Logo has to be a png and will always be saved under "img/logo.png"
//		@Tags			Core
//		@Accept			json
//		@Produce		json
//	    @Param		    data body database.Settings true "Settings Representation"
//		@Success		200
//		@Security		KeycloakAuth
//		@Router			/settings/ [put]
func updateSettings(w http.ResponseWriter, r *http.Request) {
	updateSettingsMutex.Lock()
	defer updateSettingsMutex.Unlock()
	var err error

	// Read multipart form
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Error("updateSettings: parse multipart:", err)
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}
	mForm := r.MultipartForm
	if mForm == nil {
		log.Error("updateSettings: ", errors.New("form is nil"))
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}

	// Handle normal fields
	var settings database.Settings
	fields := mForm.Value               // Values are stored in []string
	fieldsClean := make(map[string]any) // Values are stored in string
	for key, value := range fields {
		if key == "MaxOrderAmount" {
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("MaxOrderAmount is not an integer")
				utils.ErrorJSON(w, errors.New("MaxOrderAmount is not an integer"), http.StatusBadRequest)
				return
			}
		} else if key == "OrgaCoversTransactionCosts" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("OrgaCoversTransactionCosts is not a boolean")
				utils.ErrorJSON(w, errors.New("OrgaCoversTransactionCosts is not a boolean"), http.StatusBadRequest)
				return
			}
		} else if key == "WebshopIsClosed" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("WebShopIsClosed is not a boolean")
				utils.ErrorJSON(w, errors.New("WebShopIsClosed is not a boolean"), http.StatusBadRequest)
				return
			}
		} else if key == "QRCodeEnableLogo" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("QRCodeEnableLogo is not a boolean")
				utils.ErrorJSON(w, errors.New("QRCodeEnableLogo is not a boolean"), http.StatusBadRequest)
				return
			}
		} else if key == "MainItem" {
			value, err := strconv.Atoi(value[0])
			if err != nil {
				log.Error("MainItem is not an integer")
				utils.ErrorJSON(w, errors.New("MainItem is not an integer"), http.StatusBadRequest)
				return
			}
			fieldsClean[key] = null.NewInt(int64(value), true)
		} else if key == "MapCenterLat" || key == "MapCenterLong" {
			if s, err := strconv.ParseFloat(value[0], 64); err == nil {
				fieldsClean[key] = (s)
			} else {
				fieldsClean[key] = 0.1
			}
		} else if key == "UseVendorLicenseIdInShop" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("WebShopIsClosed is not a boolean")
				utils.ErrorJSON(w, errors.New("UseVendorLicenseIdInShop is not a boolean"), http.StatusBadRequest)
				return
			}
		} else {
			fieldsClean[key] = value[0]
		}
	}

	err = mapstructure.Decode(fieldsClean, &settings)
	if err != nil {
		log.Error("updateSettings: ", err)
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}
	// update the logo
	logoPath, err := updateSettingsImg(w, r, ImagetypeLogo)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if logoPath != "" {
		settings.Logo = logoPath
		log.Info("updateSettings: settings.Logo is ", settings.Logo)
	}

	// update the favicon
	faviconPath, err := updateSettingsImg(w, r, ImagetypeFavicon)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if faviconPath != "" {
		settings.Favicon = faviconPath
		log.Info("updateSettings: settings.Favicon is ", settings.Favicon)
	}

	// update the qrcode logo
	qrcodePath, err := updateSettingsImg(w, r, ImagetypeQrCode)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if qrcodePath != "" {
		settings.QRCodeLogoImgUrl = qrcodePath
		log.Info("updateSettings: settings.QRCodeLogoImgUrl is ", settings.QRCodeLogoImgUrl)
	}

	// Save settings to database
	err = database.Db.UpdateSettings(settings)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, settings)
	if err != nil {
		log.Error("updateSettings: ", err)
	}
	log.Info("updateSettings: settings updated")
}

// Online Map -----------------------------------------------------------------

// GetVendorLocations godoc
//
//	 	@Summary 		Get longitudes and latitudes of all vendors for online map
//		@Description	Get longitudes and latitudes of all vendors for online map
//		@Tags			Map
//		@Accept			json
//		@Produce		json
//		@Security		KeycloakAuth
//		@Success		200	{array}	database.LocationData
//		@Router			/map/ [get]
func GetVendorLocations(w http.ResponseWriter, r *http.Request) {
	locationData, err := database.Db.GetVendorLocations()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, locationData)
	if err != nil {
		log.Error("GetVendorLocations: ", err)
	}
}

// GetPDF godoc
//
//	@Summary		Get PDF path
//	@Description	Get PDF path
//	@Tags			PDF
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/pdf/ [get]
func GetPDF(w http.ResponseWriter, r *http.Request) {
	pdf, err := database.Db.GetPDF()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, pdf)
	if err != nil {
		log.Error("GetPDF: ", err)
	}
}

// Download PDF from id
func downloadPDF(w http.ResponseWriter, r *http.Request) {
	// Get id from URL
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Error("DownloadPDF: No id passed")
		utils.ErrorJSON(w, errors.New("missing parameter id"), http.StatusBadRequest)
		return
	}
	tx, err := database.Db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("UpdatePdfDownload: failed to start transaction ", err)
		return
	}
	defer func() {
		err = database.DeferTx(tx, err)
		if err != nil {
			log.Error("DownloadPDF: failed to defer transaction ", err)
		}
	}()

	// Get PDF from database
	pdfDownload, err := database.Db.GetPDFDownloadTx(tx, id)
	if err != nil {
		log.Error("DownloadPDF: Failed to get PDF download from database ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if pdfDownload.Timestamp.IsZero() {
		log.Error("DownloadPDF: Timestamp is zero")
		utils.ErrorJSON(w, errors.New("timestamp is zero"), http.StatusBadRequest)
		return
	}
	// check for expiration < 6 weeks
	if time.Until(pdfDownload.Timestamp).Hours() < -6*7*24 {
		log.Error("DownloadPDF: PDF is expired")
		utils.ErrorJSON(w, errors.New("pdf is expired"), http.StatusBadRequest)
		return
	}
	// Get PDF from database
	pdf, err := database.Db.GetPDFByID(int64(pdfDownload.PDF))
	if err != nil {
		log.Error("DownloadPDF: Failed to get PDF from database ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	pdfDownload.DownloadCount = pdfDownload.DownloadCount + 1
	pdfDownload.LastDownload = time.Now()
	err = database.Db.UpdatePdfDownloadTx(tx, pdfDownload)
	pdfDownload, err = database.Db.GetPDFDownloadTx(tx, id)

	if err != nil {
		log.Error("DownloadPDF: Failed to update downloadpdf ", err)
	}
	// send file
	http.ServeFile(w, r, pdf.Path)
}
func validatePDFLink(w http.ResponseWriter, r *http.Request) {
	// Get id from URL
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Error("DownloadPDF: No id passed")
		utils.ErrorJSON(w, errors.New("missing parameter id"), http.StatusBadRequest)
		return
	}

	// Get PDF from database
	pdfDownload, err := database.Db.GetPDFDownload(id)
	if err != nil {
		log.Error("DownloadPDF: Failed to get PDF from database ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if pdfDownload.Timestamp.IsZero() {
		log.Error("DownloadPDF: Timestamp is zero")
		utils.ErrorJSON(w, errors.New("timestamp is zero"), http.StatusBadRequest)
		return
	}
	// check for expiration < 6 weeks
	if time.Until(pdfDownload.Timestamp).Hours() < -6*7*24 {
		log.Error("DownloadPDF: PDF is expired")
		utils.ErrorJSON(w, errors.New("pdf is expired"), http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, "valid")
	if err != nil {
		log.Error("validatePDFLink: ", err)
	}
}

// UpdateCss godoc
//
//	@Summary 		Update CSS
//	@Description	Gets a css as a string and saves it to the disk
//	@Tags			Settings
//	@Accept			txt
//	@Produce		txt
//	@Success		200
//	@Router			/settings/css [put]

func updateCSS(w http.ResponseWriter, r *http.Request) {

	// Read data from request
	body, err := io.ReadAll(r.Body)

	if err != nil {
		log.Error("Reading body failed for css: ", err)
		err = errors.New("failed to update css")

		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Save file with name "style.css"
	path := "/public/style.css"
	dir, err := os.Getwd()
	if err != nil {
		log.Error("updateCSS: couldn't get wd", err)
		err = errors.New("failed to update css")

		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = os.WriteFile(dir+path, body, 0666)
	if err != nil {
		log.Error("updateCSS: saving failed", err)
		err = errors.New("failed to update css")
		utils.ErrorJSON(w, err, http.StatusBadRequest)
	}
	log.Info("updateCSS: success")
}
