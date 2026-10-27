package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/augustin-wien/augustina-backend/wordpress"
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
//		@Param 			includeArchived query bool false "Also return archived (deleted) items, e.g. to resolve item names in exports"
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

	includeArchived, err := parseBool(r.URL.Query().Get("includeArchived"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	var items []database.Item
	if includeArchived {
		items, err = database.Db.ListItemsIncludingArchived(skipHiddenItems, skipLicenses)
	} else {
		items, err = database.Db.ListItemsWithDisabled(skipHiddenItems, skipLicenses)
	}
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, items)
	if err != nil {
		log.Error("ListItemsBackoffice: ", err)
	}
}

// ListArchivedItems godoc
//
//		@Summary		List archived (deleted) items
//		@Tags			Items
//		@Produce		json
//		@Success		200	{array}	database.Item
//		@Security		KeycloakAuth
//		@Router			/items/archived/ [get]
func ListArchivedItems(w http.ResponseWriter, r *http.Request) {
	items, err := database.Db.ListArchivedItems()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if err := utils.WriteJSON(w, http.StatusOK, items); err != nil {
		log.Error("ListArchivedItems: ", err)
	}
}

// ListLicenseGroups godoc
//
//		@Summary		List distinct license groups from issue items
//		@Tags			Items
//		@Produce		json
//		@Success		200	{array}	string
//		@Security		KeycloakAuth
//		@Router			/items/licensegroups/ [get]
func ListLicenseGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := database.Db.ListLicenseGroups()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if err := utils.WriteJSON(w, http.StatusOK, groups); err != nil {
		log.Error("ListLicenseGroups: ", err)
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

	if err := validateLicenseGroup(item); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	if item.Type == "online_issue" || item.Type == "abonement" {
		licenseCost, err := parseLicenseCost(mForm.Value)
		if err != nil {
			log.Error("CreateItem: parse license_cost failed", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		var id int
		if item.Type == "online_issue" {
			id, _, err = database.Db.CreateOnlineIssueWithLicense(item, licenseCost)
		} else {
			id, _, err = database.Db.CreateAbonementItemWithLicense(item, licenseCost)
		}
		if err != nil {
			log.Error("CreateItem: item with license creation failed", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		err = utils.WriteJSON(w, http.StatusOK, id)
		if err != nil {
			log.Error("CreateItem: WriteJSON failed", err)
		}
		return
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

// sanitizeUploadFilename derives a safe base name and extension from a client-supplied
// upload filename. It strips any directory components and rejects names that still contain
// path separators or traversal sequences, preventing writes outside the intended directory.
func sanitizeUploadFilename(filename string) (base, ext string, err error) {
	clean := filepath.Base(filename) // strips any directory components
	parts := strings.Split(clean, ".")
	if len(parts) < 2 {
		return "", "", errors.New("invalid filename")
	}
	base = parts[0]
	ext = parts[len(parts)-1]
	if base == "" || ext == "" || strings.ContainsAny(base, `/\`) || strings.Contains(base, "..") {
		return "", "", errors.New("invalid filename")
	}
	return base, ext, nil
}

func updateItemImage(w http.ResponseWriter, r *http.Request) (path string, err error) {
	// Get file from image field
	file, header, err := r.FormFile("Image")
	if err != nil {
		return // No file passed, which is ok
	}
	defer func() { _ = file.Close() }()

	// Sanitize the client-supplied filename to prevent path traversal
	base, ext, err := sanitizeUploadFilename(header.Filename)
	if err != nil {
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
		path = "img/" + base + "_" + strconv.Itoa(i) + "." + ext
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
	defer func() { _ = file.Close() }()

	// Sanitize the client-supplied filename to prevent path traversal
	base, ext, err := sanitizeUploadFilename(header.Filename)
	if err != nil {
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
	timeStamp := time.Now().Format("2006-01-02_15-04-05.000")
	path := "pdf/" + timeStamp + "_" + base + "." + ext
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
		switch key {
		case "Price":
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse Price failed ", err)
				return
			}
		case "IsLicenseItem":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse IfLicenseItem failed ", err)
				return
			}
		case "IsPDFItem":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse IsPDFItem failed ", err)
				return
			}
		case "Archived":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse Archived failed ", err)
				return
			}

		case "Disabled":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse Disabled failed ", err)
				return
			}
		case "LicenseItem":
			licensitem, err := strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse LicenseItem failed ", err)
				return item, err
			}
			fieldsClean[key] = null.IntFrom(int64(licensitem))

		case "ID":
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse ID failed ", err)
				return
			}
		case "ItemColor":
			fieldsClean[key] = null.StringFrom(value[0])
		case "ItemTextColor":
			fieldsClean[key] = null.StringFrom(value[0])
		case "PDF":
			pdf, err := strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse PDF failed ", err)
				return item, err
			}
			fieldsClean[key] = null.IntFrom(int64(pdf))
		case "LicenseGroup":
			fieldsClean[key] = null.StringFrom(value[0])
		case "ItemOrder":
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("updateItemNormal: Parse ItemOrder failed ", err)
			}
		default:
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

func parseLicenseCost(fields map[string][]string) (licenseCost int, err error) {
	licenseCostRaw := ""
	if values, ok := fields["license_cost"]; ok && len(values) > 0 {
		licenseCostRaw = values[0]
	} else if values, ok := fields["LicenseCost"]; ok && len(values) > 0 {
		licenseCostRaw = values[0]
	}
	if licenseCostRaw == "" {
		return 0, errors.New("license_cost is required for online_issue")
	}
	licenseCost, err = strconv.Atoi(licenseCostRaw)
	if err != nil {
		return 0, errors.New("license_cost must be an integer")
	}
	if licenseCost <= 0 {
		return 0, errors.New("license_cost must be greater than 0")
	}
	return licenseCost, nil
}

// validateLicenseGroup rejects digital items without a license group. Keycloak
// gates access to online issues and abonements by that group, so without it
// customers can buy the item but never read it. PDF items are delivered as a
// download link and need no group.
func validateLicenseGroup(item database.Item) error {
	if item.Type != "online_issue" && item.Type != "abonement" {
		return nil
	}
	if item.IsPDFItem {
		return nil
	}
	if strings.TrimSpace(item.LicenseGroup.String) == "" {
		return errors.New("license group is required for " + item.Type)
	}
	return nil
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

	existingItem, err := database.Db.GetItemIncludingDisabled(ItemID)
	if err != nil {
		log.Error("UpdateItem: failed to load current item", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
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
	if item.Name == "" {
		item.Name = existingItem.Name
	}
	if item.Description == "" {
		item.Description = existingItem.Description
	}
	if item.Price == 0 {
		item.Price = existingItem.Price
	}
	if item.Image == "" {
		item.Image = existingItem.Image
	}
	if _, ok := mForm.Value["Disabled"]; !ok {
		item.Disabled = existingItem.Disabled
	}
	if _, ok := mForm.Value["Type"]; !ok {
		item.Type = existingItem.Type
	}

	path, _ := updateItemImage(w, r)
	if path != "" {
		log.Info("UpdateItem: Path is empty", path)
		item.Image = path
	}

	pdfId, err := handleItemPDF(w, r)
	if err != nil {
		log.Error("UpdateItem: handleItemPDF failed ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if pdfId != -1 {
		item.PDF = null.IntFrom(pdfId)
	}

	if err := validateLicenseGroup(item); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
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
		return
	}

	// Publishing an online_issue neither mails the abonnents nor grants it to them - the
	// backoffice does both explicitly via NotifyAbonementsOnlineIssue.

	err = utils.WriteJSON(w, http.StatusOK, err)
	if err != nil {
		log.Error("UpdateItem: ", err)
	}
}

// NotifyAbonementsOnlineIssue godoc
//
//	@Summary		Send an online issue to the abonnents
//	@Description	Checks the issue in WordPress (when the WordPress invite is set up) and makes sure its Keycloak group exists, then grants the online issue's license group to every customer with an active abonement and mails them the issue. Granting and mailing happen in the background, the access first; the response contains the number of recipients.
//	@Tags			Items
//	@Produce		json
//	@Param			id	path		int	true	"Item ID"
//	@Success		200	{object}	NotifyAbonementsResponse
//	@Security		KeycloakAuth
//	@Router			/items/{id}/notify-abonements/ [post]
func NotifyAbonementsOnlineIssue(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, errors.New("invalid item id"), http.StatusBadRequest)
		return
	}
	issue, err := database.Db.GetItemIncludingDisabled(itemID)
	if err != nil {
		utils.ErrorJSON(w, errors.New("item not found"), http.StatusNotFound)
		return
	}
	if issue.Type != "online_issue" {
		utils.ErrorJSON(w, errors.New("only an online issue can be sent to the abonnents"), http.StatusBadRequest)
		return
	}
	if issue.Disabled || issue.Archived {
		utils.ErrorJSON(w, errors.New("the online issue has to be enabled before it is sent to the abonnents"), http.StatusBadRequest)
		return
	}
	licenseGroup := strings.TrimSpace(issue.LicenseGroup.String)
	if licenseGroup == "" {
		utils.ErrorJSON(w, errors.New("the online issue has no license group"), http.StatusBadRequest)
		return
	}

	// With the WordPress invite set up, only an issue readers can open in WordPress is sent
	wpCheck, err := checkWordPressIssue(licenseGroup)
	if err != nil {
		log.Error("NotifyAbonementsOnlineIssue: ", err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if wpCheck.Configured && !wpCheck.OK {
		utils.ErrorJSON(w, errors.New(wpCheck.Message), http.StatusBadRequest)
		return
	}

	groupPath := "/" + keycloak.KeycloakClient.CustomerGroup + "/" + keycloak.KeycloakClient.NewspaperGroup + "/" + licenseGroup
	if err = keycloak.KeycloakClient.EnsureGroupPath(groupPath); err != nil {
		log.Error("NotifyAbonementsOnlineIssue: failed to ensure keycloak group ", groupPath, ": ", err)
		utils.ErrorJSON(w, errors.New("Die Keycloak-Gruppe "+groupPath+" konnte nicht angelegt werden"), http.StatusInternalServerError)
		return
	}

	recipients, err := activeAbonementRecipients()
	if err != nil {
		log.Error("NotifyAbonementsOnlineIssue: ", err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// Granting access and sending one mail per recipient easily take longer than the request
	// may, so both happen in the background. Access comes first so the mailed link works.
	issueImageURL := resolveIssueImageURL(r, issue.Image)
	go func() {
		svc := database.NewAbonementService(&database.Db)
		if err := svc.ProcessAbonementLicenseGroupsForDate(time.Now(), licenseGroup); err != nil {
			log.Error("NotifyAbonementsOnlineIssue: failed to grant the online issue to the abonnents: ", err)
		}
		sendOnlineIssueMails(issue, issueImageURL, recipients)
	}()

	respond(w, nil, NotifyAbonementsResponse{Recipients: len(recipients)})
}

// NotifyAbonementsResponse tells the backoffice how many abonnents get the mail
type NotifyAbonementsResponse struct {
	Recipients int `json:"recipients"`
}

// activeAbonementRecipients returns the distinct mail addresses of all customers with an
// active abonement
func activeAbonementRecipients() ([]string, error) {
	abonements, err := database.Db.GetActiveAbonementsByDate(time.Now())
	if err != nil {
		return nil, err
	}

	recipientSet := make(map[string]struct{})
	for _, a := range abonements {
		customer, customerErr := database.Db.GetCustomerByID(a.CustomerID)
		if customerErr != nil {
			continue
		}
		email := strings.TrimSpace(customer.Email)
		if email == "" {
			continue
		}
		recipientSet[email] = struct{}{}
	}

	recipients := make([]string, 0, len(recipientSet))
	for email := range recipientSet {
		recipients = append(recipients, email)
	}
	return recipients, nil
}

// sendOnlineIssueMails mails the online issue to each recipient
func sendOnlineIssueMails(issue database.Item, issueImageURL string, recipients []string) {
	pdfID, pdfItemID, isPDF := resolveOnlineIssuePDF(issue)
	failed := 0
	for _, recipient := range recipients {
		issueURL := config.Config.OnlinePaperUrl
		if isPDF {
			// Each recipient gets their own download link, like PDF purchases do.
			pdfDownload, dlErr := database.Db.CreatePDFDownloadForItem(pdfID, pdfItemID)
			if dlErr != nil {
				log.Error("sendOnlineIssueMails: failed to create pdf download", dlErr)
			} else {
				issueURL = config.Config.FrontendURL + "/pdf/" + pdfDownload.LinkID
			}
		}
		templateData := map[string]interface{}{
			"IssueName": issue.Name,
			"ImageURL":  issueImageURL,
			"URL":       issueURL,
		}

		mailReq, reqErr := database.BuildEmailRequestFromTemplate("onlineIssuePublished", []string{recipient}, templateData)
		if reqErr != nil {
			fallbackBody := fmt.Sprintf(
				"<p>Eine neue Online-Ausgabe ist verfügbar: <strong>%s</strong></p><p><img src=\"%s\" alt=\"%s\" style=\"max-width:100%%;height:auto;\"></p><p><a href=\"%s\">Hier klicken</a> um die Zeitung zu lesen.</p>",
				issue.Name,
				issueImageURL,
				issue.Name,
				issueURL,
			)
			mailReq, reqErr = mailer.NewRequest([]string{recipient}, "Neue Online-Ausgabe verfügbar", fallbackBody)
			if reqErr != nil {
				log.Error("sendOnlineIssueMails: failed to build fallback mail", reqErr)
				failed++
				continue
			}
		}

		if _, sendErr := mailer.Send(mailReq); sendErr != nil {
			log.Error("sendOnlineIssueMails: sending failed", sendErr)
			failed++
		}
	}
	log.Infof("sendOnlineIssueMails: online issue %d sent to %d of %d abonnents", issue.ID, len(recipients)-failed, len(recipients))
}

// GetItemPDF godoc
//
//	@Summary		Get the PDF of an item
//	@Description	Serves the PDF attached to an item, so admins can check what customers receive
//	@Tags			Items
//	@Produce		application/pdf
//	@Success		200
//	@Security		KeycloakAuth
//	@Param			id	path	int	true	"Item ID"
//	@Router			/items/{id}/pdf/ [get]
func GetItemPDF(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	item, err := database.Db.GetItemIncludingDisabled(itemID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}
	if !item.PDF.Valid {
		utils.ErrorJSON(w, errors.New("item has no pdf"), http.StatusNotFound)
		return
	}
	pdf, err := database.Db.GetPDFByID(item.PDF.Int64)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(pdf.Path)))
	http.ServeFile(w, r, pdf.Path)
}

// resolveOnlineIssuePDF returns the PDF of an online issue, either attached to
// the issue itself or to its linked license item.
func resolveOnlineIssuePDF(issue database.Item) (pdfID int, itemID int, ok bool) {
	if issue.IsPDFItem && issue.PDF.Valid {
		return int(issue.PDF.ValueOrZero()), issue.ID, true
	}
	if issue.LicenseItem.Valid {
		licenseItem, err := database.Db.GetItemIncludingDisabled(int(issue.LicenseItem.ValueOrZero()))
		if err != nil {
			log.Error("resolveOnlineIssuePDF: failed to load license item", err)
			return 0, 0, false
		}
		if licenseItem.IsPDFItem && licenseItem.PDF.Valid {
			return int(licenseItem.PDF.ValueOrZero()), licenseItem.ID, true
		}
	}
	return 0, 0, false
}

func resolveIssueImageURL(r *http.Request, imagePath string) string {
	trimmed := strings.TrimSpace(imagePath)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.Config.FrontendURL), "/")
	if baseURL == "" && r != nil {
		baseURL = "http://" + strings.TrimSpace(r.Host)
	}
	if baseURL == "" {
		return trimmed
	}
	return baseURL + "/" + strings.TrimLeft(trimmed, "/")
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
//	     	@Param          deleteLicense   query bool  false  "Also delete the linked license item"
//			@Failure		409
//			@Router			/items/{id}/ [delete]
func DeleteItem(w http.ResponseWriter, r *http.Request) {
	ItemID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	deleteLicense := r.URL.Query().Get("deleteLicense") == "true"
	err = database.Db.DeleteItemChecked(ItemID, deleteLicense)
	if errors.Is(err, database.ErrLicenseItemInUse) {
		utils.ErrorJSON(w, err, http.StatusConflict)
		return
	}
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RestoreItem godoc
//
//		 	@Summary 		Restore an archived (deleted) item
//			@Tags			Items
//			@Produce		json
//			@Success		204
//			@Security		KeycloakAuth
//	     	@Param          id   path int  true  "Item ID"
//			@Router			/items/{id}/restore/ [post]
func RestoreItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.RestoreItem(itemID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// WordPressIssueCheck tells the backoffice whether a license group names an
// issue in WordPress and whether a new subscriber would get it
type WordPressIssueCheck struct {
	// Configured is false when the WordPress invite settings are empty
	Configured bool `json:"configured"`
	// OK is true when the issue exists, is published and is no demo
	OK      bool            `json:"ok"`
	Message string          `json:"message"`
	Issue   wordpress.Issue `json:"issue"`
}

// CheckWordPressIssue godoc
//
//	@Summary		Check a license group in WordPress
//	@Description	Asks the augustin-ki plugin whether an issue with this number (the license group) exists, is published and is a demo issue. Uses the WordPress invite settings.
//	@Tags			Items
//	@Produce		json
//	@Param			licenseGroup	query		string	true	"License group (issue number)"
//	@Success		200	{object}	WordPressIssueCheck
//	@Security		KeycloakAuth
//	@Router			/items/wordpress-issue/ [get]
func CheckWordPressIssue(w http.ResponseWriter, r *http.Request) {
	res, err := checkWordPressIssue(strings.TrimSpace(r.URL.Query().Get("licenseGroup")))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if err = utils.WriteJSON(w, http.StatusOK, res); err != nil {
		log.Error("CheckWordPressIssue: ", err)
	}
}

// checkWordPressIssue asks WordPress about the issue with this license group.
// The error is only set when the settings can't be read.
func checkWordPressIssue(licenseGroup string) (WordPressIssueCheck, error) {
	res := WordPressIssueCheck{Issue: wordpress.Issue{IssueNumber: licenseGroup}}
	settings, err := database.Db.GetSettings()
	if err != nil {
		return res, err
	}

	switch {
	case settings.WordPressInviteURL == "" || settings.WordPressInviteAPIKey == "":
		res.Message = "Die WordPress-Einladung ist in den Einstellungen nicht eingerichtet"
	case licenseGroup == "":
		res.Configured = true
		res.Message = "Keine Lizenzgruppe eingetragen"
	case wordpress.IssueNumber(licenseGroup) == "":
		res.Configured = true
		res.Message = "Die Lizenzgruppe darf nur Buchstaben, Ziffern, _ und - enthalten (höchstens 32 Zeichen)"
	default:
		res.Configured = true
		issue, err := wordpress.LookupIssue(settings.WordPressInviteURL, settings.WordPressInviteAPIKey, licenseGroup)
		var statusErr *wordpress.StatusError
		switch {
		case errors.As(err, &statusErr) && statusErr.Status == http.StatusNotFound && statusErr.Code == "rest_no_route":
			res.Message = "Das augustin-ki-Plugin in WordPress ist zu alt für diese Prüfung"
		case errors.As(err, &statusErr):
			res.Message = statusErr.Hint()
		case err != nil:
			log.Info("CheckWordPressIssue: ", err)
			res.Message = "WordPress ist nicht erreichbar: " + err.Error()
		case !issue.Exists:
			res.Issue = issue
			res.Message = "In WordPress gibt es keine Ausgabe mit der Nummer " + licenseGroup
		case issue.IsDemo:
			res.Issue = issue
			res.Message = "„" + issue.Title + "“ ist in WordPress eine Demo-Ausgabe – Abonnent·innen bekommen sie nicht"
		case !issue.Published:
			res.Issue = issue
			res.Message = "„" + issue.Title + "“ existiert in WordPress, ist aber noch nicht veröffentlicht"
		default:
			res.Issue = issue
			res.OK = true
			res.Message = "„" + issue.Title + "“ ist in WordPress veröffentlicht"
		}
	}
	return res, nil
}
