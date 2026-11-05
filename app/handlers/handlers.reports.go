package handlers

import (
	"errors"
	"net/http"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/reports"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

// buildMonthlyReport builds the report of the month in the URL and writes an error response if that fails
func buildMonthlyReport(w http.ResponseWriter, r *http.Request) (reports.MonthlyReport, bool) {
	month := chi.URLParam(r, "month")
	if _, _, err := reports.ParseMonth(month); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return reports.MonthlyReport{}, false
	}
	report, err := reports.Build(month)
	if err != nil {
		log.Error("buildMonthlyReport: ", err)
		utils.ErrorJSON(w, errors.New("building the report failed"), http.StatusInternalServerError)
		return reports.MonthlyReport{}, false
	}
	return report, true
}

// GetMonthlyReport godoc
//
//	@Summary		Monthly accounting report
//	@Description	Income, license income, transaction costs and payouts of one month (Europe/Vienna)
//	@Tags			Reports
//	@Produce		json
//	@Param			month	path		string	true	"Month (YYYY-MM)"
//	@Success		200		{object}	reports.MonthlyReport
//	@Security		KeycloakAuth
//	@Router			/reports/monthly/{month}/ [get]
func GetMonthlyReport(w http.ResponseWriter, r *http.Request) {
	report, ok := buildMonthlyReport(w, r)
	if !ok {
		return
	}
	respond(w, nil, report)
}

// GetMonthlyReportCSV godoc
//
//	@Summary		Monthly accounting report as CSV
//	@Tags			Reports
//	@Produce		text/csv
//	@Param			month	path	string	true	"Month (YYYY-MM)"
//	@Success		200
//	@Security		KeycloakAuth
//	@Router			/reports/monthly/{month}/csv/ [get]
func GetMonthlyReportCSV(w http.ResponseWriter, r *http.Request) {
	report, ok := buildMonthlyReport(w, r)
	if !ok {
		return
	}
	data, err := report.CSV()
	if err != nil {
		log.Error("GetMonthlyReportCSV: ", err)
		utils.ErrorJSON(w, errors.New("exporting the report failed"), http.StatusInternalServerError)
		return
	}
	writeReportFile(w, report.FileName()+".csv", "text/csv; charset=utf-8", data)
}

// GetMonthlyReportPDF godoc
//
//	@Summary		Monthly accounting report as PDF
//	@Tags			Reports
//	@Produce		application/pdf
//	@Param			month	path	string	true	"Month (YYYY-MM)"
//	@Success		200
//	@Security		KeycloakAuth
//	@Router			/reports/monthly/{month}/pdf/ [get]
func GetMonthlyReportPDF(w http.ResponseWriter, r *http.Request) {
	report, ok := buildMonthlyReport(w, r)
	if !ok {
		return
	}
	newspaperName := ""
	if settings, err := database.Db.GetSettings(); err == nil {
		newspaperName = settings.NewspaperName
	}
	data, err := report.PDF(newspaperName)
	if err != nil {
		log.Error("GetMonthlyReportPDF: ", err)
		utils.ErrorJSON(w, errors.New("exporting the report failed"), http.StatusInternalServerError)
		return
	}
	writeReportFile(w, report.FileName()+".pdf", "application/pdf", data)
}

// SendMonthlyReport godoc
//
//	@Summary		Mail the monthly accounting report
//	@Description	Sends the report of the month to the accounting report email from the settings
//	@Tags			Reports
//	@Param			month	path	string	true	"Month (YYYY-MM)"
//	@Success		200
//	@Security		KeycloakAuth
//	@Router			/reports/monthly/{month}/send/ [post]
func SendMonthlyReport(w http.ResponseWriter, r *http.Request) {
	month := chi.URLParam(r, "month")
	if _, _, err := reports.ParseMonth(month); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err := reports.SendMonthly(month, mailer.Send)
	if errors.Is(err, reports.ErrNoRecipients) {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Error("SendMonthlyReport: ", err)
		utils.ErrorJSON(w, errors.New("sending the report failed"), http.StatusInternalServerError)
		return
	}
	respond(w, nil, map[string]bool{"success": true})
}

func writeReportFile(w http.ResponseWriter, filename, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Error("writeReportFile: ", err)
	}
}
