package handlers

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const (
	feedbackMaxMessageLength = 2000
	feedbackMaxIDLength      = 200
)

// FeedbackInput is what the shop sends from the feedback form
type FeedbackInput struct {
	Message       string
	TransactionID string
	RequestID     string
	VendorID      string
	Page          string
	Email         string
}

func (f *FeedbackInput) normalize() error {
	f.Message = strings.TrimSpace(f.Message)
	if f.Message == "" {
		return errors.New("message is empty")
	}
	if utf8.RuneCountInString(f.Message) > feedbackMaxMessageLength {
		return fmt.Errorf("message is longer than %d characters", feedbackMaxMessageLength)
	}
	for _, v := range []*string{&f.TransactionID, &f.RequestID, &f.VendorID, &f.Page, &f.Email} {
		*v = strings.TrimSpace(*v)
		if utf8.RuneCountInString(*v) > feedbackMaxIDLength {
			return errors.New("field is too long")
		}
	}
	return nil
}

func feedbackMailBody(f FeedbackInput, requestID string) string {
	row := func(label, value string) string {
		if value == "" {
			value = "-"
		}
		return fmt.Sprintf("<tr><th align=\"left\">%s</th><td>%s</td></tr>", label, html.EscapeString(value))
	}
	return "<p>Neues Feedback aus dem Shop:</p>" +
		"<p>" + strings.ReplaceAll(html.EscapeString(f.Message), "\n", "<br>") + "</p>" +
		"<table>" +
		row("Seite", f.Page) +
		row("Verkäufer*innen-ID", f.VendorID) +
		row("Transaktions-ID", f.TransactionID) +
		row("E-Mail", f.Email) +
		row("Request-ID", f.RequestID) +
		row("Request-ID (Feedback)", requestID) +
		row("Shop", config.Config.FrontendURL) +
		"</table>"
}

// SendFeedback godoc
//
//	@Summary		Send feedback from the shop
//	@Description	Mails the feedback, together with the transaction and request id, to the feedback address from the settings. Only available when feedback is enabled.
//	@Tags			Core
//	@Accept			json
//	@Param			data	body	FeedbackInput	true	"Feedback"
//	@Success		200
//	@Router			/feedback/ [post]
func SendFeedback(w http.ResponseWriter, r *http.Request) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		log.Error("SendFeedback: get settings: ", err)
		utils.ErrorJSON(w, errors.New("internal error"), http.StatusInternalServerError)
		return
	}
	if !settings.FeedbackEnabled || settings.FeedbackEmail == "" {
		utils.ErrorJSON(w, errors.New("feedback is disabled"), http.StatusNotFound)
		return
	}

	var input FeedbackInput
	if err := utils.ReadJSON(w, r, &input); err != nil {
		utils.ErrorJSON(w, errors.New("invalid body"), http.StatusBadRequest)
		return
	}
	if err := input.normalize(); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	subject := "Feedback aus dem Shop"
	if settings.NewspaperName != "" {
		subject += " - " + strings.NewReplacer("\r", "", "\n", "").Replace(settings.NewspaperName)
	}
	mailReq, err := mailer.NewRequest([]string{settings.FeedbackEmail}, subject, feedbackMailBody(input, chimiddleware.GetReqID(r.Context())))
	if err != nil {
		log.Error("SendFeedback: build mail: ", err)
		utils.ErrorJSON(w, errors.New("internal error"), http.StatusInternalServerError)
		return
	}
	if _, err := mailer.Send(mailReq); err != nil {
		log.Error("SendFeedback: send mail: ", err)
		utils.ErrorJSON(w, errors.New("feedback could not be sent"), http.StatusBadGateway)
		return
	}
	respond(w, nil, nil)
}
