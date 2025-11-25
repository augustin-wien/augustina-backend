package handlers

import (
	"errors"
	"net/http"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

var mtlog = utils.GetLogger()

// ListMailTemplates godoc
// @Summary List mail templates
// @Tags MailTemplates
// @Produce json
// @Success 200 {array} database.MailTemplate
// @Router /mail-templates/ [get]
func ListMailTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := database.Db.ListMailTemplates()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err := utils.WriteJSON(w, http.StatusOK, templates); err != nil {
		mtlog.Error("ListMailTemplates: WriteJSON failed", err)
	}
}

// GetMailTemplate godoc
// @Summary Get mail template by name
// @Tags MailTemplates
// @Produce json
// @Param name path string true "Template name"
// @Success 200 {object} database.MailTemplate
// @Router /mail-templates/{name}/ [get]
func GetMailTemplate(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	mt, err := database.Db.GetMailTemplateByName(name)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err := utils.WriteJSON(w, http.StatusOK, mt); err != nil {
		mtlog.Error("GetMailTemplate: WriteJSON failed", err)
	}
}

// CreateOrUpdateMailTemplate godoc
// @Summary Create or update a mail template
// @Tags MailTemplates
// @Accept json
// @Produce json
// @Param data body database.MailTemplate true "Template"
// @Success 200 {string} string "ok"
// @Security KeycloakAuth
// @Router /mail-templates/ [post]
func CreateOrUpdateMailTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		utils.ErrorJSON(w, errors.New("invalid request: name required"), http.StatusBadRequest)
		return
	}
	if err := database.Db.CreateOrUpdateMailTemplate(req.Name, req.Subject, req.Body); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err := utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		mtlog.Error("CreateOrUpdateMailTemplate: WriteJSON failed", err)
	}
}

// DeleteMailTemplate godoc
// @Summary Delete a mail template
// @Tags MailTemplates
// @Param name path string true "Template name"
// @Success 200 {string} string "ok"
// @Security KeycloakAuth
// @Router /mail-templates/{name}/ [delete]
func DeleteMailTemplate(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		utils.ErrorJSON(w, errors.New("invalid request: name required"), http.StatusBadRequest)
		return
	}
	if err := database.Db.DeleteMailTemplate(name); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err := utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		mtlog.Error("DeleteMailTemplate: WriteJSON failed", err)
	}
}

// SendMailTemplateTest godoc
// @Summary Send a test email for a stored template
// @Tags MailTemplates
// @Accept json
// @Produce json
// @Param name path string true "Template name"
// @Param data body object true "Payload with 'to' (array of recipients) and 'data' for template rendering"
// @Success 200 {object} map[string]string
// @Security KeycloakAuth
// @Router /mail-templates/{name}/send/ [post]
func SendMailTemplateTest(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		utils.ErrorJSON(w, errors.New("invalid request: name required"), http.StatusBadRequest)
		return
	}

	var req struct {
		To   []string               `json:"to"`
		Data map[string]interface{} `json:"data"`
	}
	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if len(req.To) == 0 {
		utils.ErrorJSON(w, errors.New("invalid request: 'to' must contain at least one recipient"), http.StatusBadRequest)
		return
	}
	if req.Data == nil {
		req.Data = map[string]interface{}{
			"URL":   "http://example.com",
			"EMAIL": req.To[0],
		}
	}

	mailReq, err := database.BuildEmailRequestFromTemplate(name, req.To, req.Data)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	ok, err := mailer.Send(mailReq)
	if err != nil || !ok {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err := utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "sent"}); err != nil {
		mtlog.Error("SendMailTemplateTest: WriteJSON failed", err)
	}
}
