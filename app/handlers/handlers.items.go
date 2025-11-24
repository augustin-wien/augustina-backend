package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/guregu/null.v4"
)

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
	items, err := database.Db.ListItemsShop()
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

	items, err := database.Db.ListItemsWithDisabled(skipHiddenItems, skipLicenses)
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
	// handleItemPDF returns -1 when no file was provided; only set PDF when we have a positive id
	if pdfId > 0 {
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

	// Save file with unique name (owner read/write, group/other read)
	err = os.WriteFile(dir+"/"+path, buf.Bytes(), 0644)
	if err != nil {
		log.Error("updateItemImage: failed to write file", err)
	}
	return
}

func handleItemPDF(w http.ResponseWriter, r *http.Request) (pdfId int64, err error) {
	log.Info("handleItemPDF: handling PDF upload")
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
		err = os.Mkdir(dir+"/pdf", 0755)
		if err != nil {
			log.Error("handleItemPDF: failed to create directory", err)
			return
		}
	}
	err = os.WriteFile(dir+"/"+path, buf.Bytes(), 0644)
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

		} else if key == "Disabled" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse Disabled failed ", err)
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
			if err != nil {
				log.Error("updateItemNormal: Parse ItemOrder failed ", err)
			}
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
		utils.ErrorJSON(w, errors.New("nice try! You are not allowed to update this item"), http.StatusBadRequest)
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

	// If some fields are omitted in the multipart form (e.g. Description),
	// preserve existing values from the DB to avoid ent validator failures
	// on partial updates.
	if item.Description == "" {
		if existing, err := database.Db.GetItem(ItemID); err == nil {
			item.Description = existing.Description
		}
	}
	if item.Price == 0 {
		if existing, err := database.Db.GetItem(ItemID); err == nil {
			item.Price = existing.Price
		}
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
	// If a LicenseItem is provided, ensure it's not already assigned to another item.
	if item.LicenseItem.Valid {
		licID := int(item.LicenseItem.ValueOrZero())
		owner, found, err := database.Db.GetItemByLicenseID(licID)
		if err != nil {
			log.Error("UpdateItem: failed to check license ownership", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		if found && owner.ID != ItemID {
			utils.ErrorJSON(w, errors.New("license item is already assigned to another item"), http.StatusBadRequest)
			return
		}
	}
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
