package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/go-chi/chi/v5"
)

// ListVendorComments godoc
//
// @Summary List vendor comments
// @Description List vendor comments
// @ID listVendorComments
// @Produce json
// @Success 200 {array} Comment
// @Router /api/vendors/comments/ [get]
// @Security KeycloakAuth

func ListVendorComments(w http.ResponseWriter, r *http.Request) {
	vendorID := chi.URLParam(r, "vendorID")
	if vendorID == "" {
		utils.ErrorJSON(w, fmt.Errorf("vendorId is required"), http.StatusBadRequest)
		return
	}
	vendorIDInt, err := strconv.Atoi(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	comments, err := database.Db.GetVendorComments(vendorIDInt)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, comments)
}

func CreateVendorComment(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("CreateVendorComment: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var comment ent.Comment
	err = utils.ReadJSON(w, r, &comment)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.CreateVendorComment(vendorID, comment)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}

func UpdateVendorComment(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("CreateVendorComment: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var comment ent.Comment
	err = utils.ReadJSON(w, r, &comment)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.UpdateVendorComment(vendorID, comment)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}

// DeleteVendorComment godoc
//
// @Summary Delete vendor comment
// @Description Delete vendor comment
// @ID deleteVendorComment
// @Produce json
// @Router /api/vendors/comments/{id}/ [delete]
// @Security KeycloakAuth
func DeleteVendorComment(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("DeleteVendorComment: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	commentID, err := strconv.Atoi(chi.URLParam(r, "commentID"))
	if err != nil {
		log.Error("DeleteVendorComment: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.DeleteVendorComment(vendorID, commentID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}
