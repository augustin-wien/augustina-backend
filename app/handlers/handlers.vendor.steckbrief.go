package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

const (
	maxSteckbriefImageSize = 10 << 20
	maxSteckbriefTextLen   = 2000
)

var errSteckbriefDisabled = errors.New("the Steckbrief is disabled")

// steckbriefImageTypes maps the accepted photo content types to the file extension they are saved with
var steckbriefImageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

// steckbriefVendorID checks that the Steckbrief is enabled and returns the vendor ID from
// the URL. On error it has already written the response.
func steckbriefVendorID(w http.ResponseWriter, r *http.Request) (int, bool) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return 0, false
	}
	if !settings.SteckbriefEnabled {
		utils.ErrorJSON(w, errSteckbriefDisabled, http.StatusForbidden)
		return 0, false
	}
	vendorID, err := strconv.Atoi(chi.URLParam(r, "vendorid"))
	if err != nil {
		utils.ErrorJSON(w, errors.New("invalid vendor id"), http.StatusBadRequest)
		return 0, false
	}
	return vendorID, true
}

func respondSteckbrief(w http.ResponseWriter, s database.VendorSteckbrief, err error) {
	if errors.Is(err, database.ErrVendorNotFound) {
		utils.ErrorJSON(w, errors.New("vendor not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("Steckbrief: ", err)
		utils.ErrorJSON(w, errors.New("failed to load or save the Steckbrief"), http.StatusInternalServerError)
		return
	}
	respond(w, nil, s)
}

// GetVendorSteckbrief godoc
//
//	 	@Summary 		Get the vendor's Steckbrief
//		@Tags			Vendors
//		@Produce		json
//		@Success		200	{object}	database.VendorSteckbrief
//		@Failure		403
//		@Failure		404
//		@Security		KeycloakAuth
//	    @Param          vendorid   path int  true  "Vendor ID"
//		@Router			/vendors/{vendorid}/steckbrief/ [get]
func GetVendorSteckbrief(w http.ResponseWriter, r *http.Request) {
	vendorID, ok := steckbriefVendorID(w, r)
	if !ok {
		return
	}
	s, err := database.Db.GetVendorSteckbrief(vendorID)
	respondSteckbrief(w, s, err)
}

// UpdateVendorSteckbrief godoc
//
//	 	@Summary 		Update the text of the vendor's Steckbrief
//		@Description	Stores Description, Family, Residence, Education, CareerGoal and Support. All other fields are ignored.
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Param			data body database.VendorSteckbrief true "Steckbrief"
//		@Success		200	{object}	database.VendorSteckbrief
//		@Failure		400
//		@Failure		403
//		@Failure		404
//		@Security		KeycloakAuth
//	    @Param          vendorid   path int  true  "Vendor ID"
//		@Router			/vendors/{vendorid}/steckbrief/ [put]
func UpdateVendorSteckbrief(w http.ResponseWriter, r *http.Request) {
	vendorID, ok := steckbriefVendorID(w, r)
	if !ok {
		return
	}
	var s database.VendorSteckbrief
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&s); err != nil {
		utils.ErrorJSON(w, errors.New("invalid Steckbrief"), http.StatusBadRequest)
		return
	}
	for name, value := range map[string]*string{
		"Description": &s.Description, "Family": &s.Family, "Residence": &s.Residence,
		"Education": &s.Education, "CareerGoal": &s.CareerGoal, "Support": &s.Support,
	} {
		*value = strings.TrimSpace(*value)
		if utf8.RuneCountInString(*value) > maxSteckbriefTextLen {
			utils.ErrorJSON(w, fmt.Errorf("%s is longer than %d characters", name, maxSteckbriefTextLen), http.StatusBadRequest)
			return
		}
	}
	s, err := database.Db.UpdateVendorSteckbrief(vendorID, s)
	if err == nil {
		log.Info(r.Header.Get("X-Auth-User-Name") + " updated the Steckbrief of vendor " + strconv.Itoa(vendorID))
	}
	respondSteckbrief(w, s, err)
}

// UploadVendorSteckbriefImage godoc
//
//	 	@Summary 		Upload the photo of the vendor's Steckbrief
//		@Description	Replaces the photo. The form field is "Image"; jpeg, png and webp are accepted. The photo is served publicly under /img/.
//		@Tags			Vendors
//		@Accept			mpfd
//		@Produce		json
//		@Success		200	{object}	database.VendorSteckbrief
//		@Failure		400
//		@Failure		403
//		@Failure		404
//		@Security		KeycloakAuth
//	    @Param          vendorid   path int  true  "Vendor ID"
//		@Router			/vendors/{vendorid}/steckbrief/image/ [post]
func UploadVendorSteckbriefImage(w http.ResponseWriter, r *http.Request) {
	vendorID, ok := steckbriefVendorID(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSteckbriefImageSize+(1<<20))
	file, _, err := r.FormFile("Image")
	if err != nil {
		utils.ErrorJSON(w, errors.New("no image uploaded or image too large"), http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, maxSteckbriefImageSize+1))
	if err != nil || len(data) > maxSteckbriefImageSize {
		utils.ErrorJSON(w, errors.New("image too large"), http.StatusBadRequest)
		return
	}
	ext, ok := steckbriefImageTypes[http.DetectContentType(data)]
	if !ok {
		utils.ErrorJSON(w, errors.New("image must be a jpg, png or webp file"), http.StatusBadRequest)
		return
	}
	if _, err = database.Db.GetVendorSteckbrief(vendorID); err != nil {
		respondSteckbrief(w, database.VendorSteckbrief{}, err)
		return
	}

	// Random name, so the public URL of the photo cannot be guessed from the vendor
	name := "steckbrief_" + strings.ToLower(rand.Text()) + "_0." + ext
	if err = writeImage(name, data); err != nil {
		log.Error("UploadVendorSteckbriefImage: ", err)
		utils.ErrorJSON(w, errors.New("failed to save image"), http.StatusInternalServerError)
		return
	}
	if err = createThumbnail(data, "img/"+name); err != nil {
		log.Warn("UploadVendorSteckbriefImage: no thumbnail: ", err)
	}

	s, previous, err := database.Db.SetVendorSteckbriefImage(vendorID, "img/"+name)
	if err != nil {
		removeImage(name)
	} else {
		removeImage(filepath.Base(previous))
		log.Info(r.Header.Get("X-Auth-User-Name") + " uploaded a Steckbrief photo for vendor " + strconv.Itoa(vendorID))
	}
	respondSteckbrief(w, s, err)
}

// DeleteVendorSteckbriefImage godoc
//
//	 	@Summary 		Remove the photo of the vendor's Steckbrief
//		@Tags			Vendors
//		@Produce		json
//		@Success		200	{object}	database.VendorSteckbrief
//		@Failure		403
//		@Failure		404
//		@Security		KeycloakAuth
//	    @Param          vendorid   path int  true  "Vendor ID"
//		@Router			/vendors/{vendorid}/steckbrief/image/ [delete]
func DeleteVendorSteckbriefImage(w http.ResponseWriter, r *http.Request) {
	vendorID, ok := steckbriefVendorID(w, r)
	if !ok {
		return
	}
	s, previous, err := database.Db.SetVendorSteckbriefImage(vendorID, "")
	if err == nil {
		removeImage(filepath.Base(previous))
	}
	respondSteckbrief(w, s, err)
}

// writeImage saves data as img/<name>
func writeImage(name string, data []byte) error {
	if !validImageName(name) {
		return errors.New("invalid image name")
	}
	root, err := os.OpenRoot("img")
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	if _, err = io.Copy(f, bytes.NewReader(data)); err != nil {
		_ = f.Close()
		_ = root.Remove(name)
		return err
	}
	return f.Close()
}

// removeImage deletes img/<name> and its thumbnail. Missing files are ignored.
func removeImage(name string) {
	if !validImageName(name) {
		return
	}
	root, err := os.OpenRoot("img")
	if err != nil {
		return
	}
	defer func() { _ = root.Close() }()
	thumb := strings.TrimPrefix(thumbnailDir, "img/") + "/" + name
	for _, p := range []string{name, thumb} {
		if err := root.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Warn("removeImage: ", err)
		}
	}
}

// PublicVendorSteckbrief is the Steckbrief as the shop shows it to customers
type PublicVendorSteckbrief struct {
	FirstName   string // empty when the shop shows license IDs instead of names
	LastName    string // empty when the shop shows license IDs instead of names
	LicenseID   string
	Image       string
	Description string
	Family      string
	Residence   string
	Education   string
	CareerGoal  string
	Support     string
}

// GetPublicVendorSteckbrief godoc
//
//	 	@Summary 		Get the Steckbrief of a vendor for the shop
//		@Description	Public. Answers 404 when the module is off, the vendor is disabled or blocked, or the Steckbrief is empty.
//		@Tags			Vendors
//		@Produce		json
//		@Success		200	{object}	PublicVendorSteckbrief
//		@Failure		404
//	    @Param          licenseID   path string  true  "License ID"
//		@Router			/vendors/check/{licenseID}/steckbrief/ [get]
func GetPublicVendorSteckbrief(w http.ResponseWriter, r *http.Request) {
	notFound := func() { utils.ErrorJSON(w, errors.New("no Steckbrief for this vendor"), http.StatusNotFound) }
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if !settings.SteckbriefEnabled {
		notFound()
		return
	}
	vendor, err := database.Db.GetVendorByLicenseIDWithoutDisabled(chi.URLParam(r, "licenseID"))
	if err != nil || vendor.IsBlocked || vendor.IsDeleted {
		notFound()
		return
	}
	s, err := database.Db.GetVendorSteckbrief(vendor.ID)
	if err != nil || s.IsEmpty() {
		notFound()
		return
	}
	public := PublicVendorSteckbrief{
		FirstName:   s.FirstName,
		LastName:    s.LastName,
		LicenseID:   s.LicenseID,
		Image:       s.Image,
		Description: s.Description,
		Family:      s.Family,
		Residence:   s.Residence,
		Education:   s.Education,
		CareerGoal:  s.CareerGoal,
		Support:     s.Support,
	}
	if settings.UseVendorLicenseIdInShop {
		public.FirstName, public.LastName = "", ""
	}
	respond(w, nil, public)
}
