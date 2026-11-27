package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/reports"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

// reportPeriod reads the period of a report from the request
type reportPeriod func(r *http.Request) (from, to time.Time, err error)

func monthFromURL(r *http.Request) (from, to time.Time, err error) {
	return reports.ParseMonth(chi.URLParam(r, "month"))
}

func periodFromQuery(r *http.Request) (from, to time.Time, err error) {
	return reports.ParsePeriod(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
}

// buildReport builds the report of the period in the request and writes an error response if that fails
func buildReport(w http.ResponseWriter, r *http.Request, period reportPeriod) (reports.MonthlyReport, bool) {
	from, to, err := period(r)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return reports.MonthlyReport{}, false
	}
	report, err := reports.BuildPeriod(from, to)
	if err != nil {
		log.Error("buildReport: ", err)
		utils.ErrorJSON(w, errors.New("building the report failed"), http.StatusInternalServerError)
		return reports.MonthlyReport{}, false
	}
	return report, true
}

// rememberReport stores the last day of an exported or sent report, so the next period can start the day after
func rememberReport(to time.Time) {
	last := to.In(reports.Location).AddDate(0, 0, -1)
	now := time.Now().In(reports.Location)
	if today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, reports.Location); last.After(today) {
		last = today
	}
	if err := database.Db.SetAccountingReportLastTo(last.Format(time.DateOnly)); err != nil {
		log.Error("rememberReport: ", err)
	}
}

func getReport(w http.ResponseWriter, r *http.Request, period reportPeriod) {
	report, ok := buildReport(w, r, period)
	if !ok {
		return
	}
	respond(w, nil, report)
}

func getReportCSV(w http.ResponseWriter, r *http.Request, period reportPeriod) {
	report, ok := buildReport(w, r, period)
	if !ok {
		return
	}
	data, err := report.CSV()
	if err != nil {
		log.Error("getReportCSV: ", err)
		utils.ErrorJSON(w, errors.New("exporting the report failed"), http.StatusInternalServerError)
		return
	}
	rememberReport(report.To)
	writeReportFile(w, report.FileName()+".csv", "text/csv; charset=utf-8", data)
}

func getReportPDF(w http.ResponseWriter, r *http.Request, period reportPeriod) {
	report, ok := buildReport(w, r, period)
	if !ok {
		return
	}
	newspaperName := ""
	if settings, err := database.Db.GetSettings(); err == nil {
		newspaperName = settings.NewspaperName
	}
	data, err := report.PDF(newspaperName)
	if err != nil {
		log.Error("getReportPDF: ", err)
		utils.ErrorJSON(w, errors.New("exporting the report failed"), http.StatusInternalServerError)
		return
	}
	rememberReport(report.To)
	writeReportFile(w, report.FileName()+".pdf", "application/pdf", data)
}

func sendReport(w http.ResponseWriter, r *http.Request, period reportPeriod) {
	from, to, err := period(r)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = reports.SendPeriod(from, to, database.Db.QueueMailFunc(database.MailMeta{Kind: database.MailKindAccountingReport}))
	if errors.Is(err, reports.ErrNoRecipients) {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Error("sendReport: ", err)
		utils.ErrorJSON(w, errors.New("sending the report failed"), http.StatusInternalServerError)
		return
	}
	rememberReport(to)
	respond(w, nil, map[string]bool{"success": true})
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
	getReport(w, r, monthFromURL)
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
	getReportCSV(w, r, monthFromURL)
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
	getReportPDF(w, r, monthFromURL)
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
	sendReport(w, r, monthFromURL)
}

// GetPeriodReport godoc
//
//	@Summary		Accounting report of a period
//	@Description	Income, license income, transaction costs and payouts from the first to the last day, both included (Europe/Vienna)
//	@Tags			Reports
//	@Produce		json
//	@Param			from	query		string	true	"First day (YYYY-MM-DD)"
//	@Param			to		query		string	true	"Last day (YYYY-MM-DD)"
//	@Success		200		{object}	reports.MonthlyReport
//	@Security		KeycloakAuth
//	@Router			/reports/period/ [get]
func GetPeriodReport(w http.ResponseWriter, r *http.Request) {
	getReport(w, r, periodFromQuery)
}

// GetPeriodReportCSV godoc
//
//	@Summary		Accounting report of a period as CSV
//	@Tags			Reports
//	@Produce		text/csv
//	@Param			from	query	string	true	"First day (YYYY-MM-DD)"
//	@Param			to		query	string	true	"Last day (YYYY-MM-DD)"
//	@Success		200
//	@Security		KeycloakAuth
//	@Router			/reports/period/csv/ [get]
func GetPeriodReportCSV(w http.ResponseWriter, r *http.Request) {
	getReportCSV(w, r, periodFromQuery)
}

// GetPeriodReportPDF godoc
//
//	@Summary		Accounting report of a period as PDF
//	@Tags			Reports
//	@Produce		application/pdf
//	@Param			from	query	string	true	"First day (YYYY-MM-DD)"
//	@Param			to		query	string	true	"Last day (YYYY-MM-DD)"
//	@Success		200
//	@Security		KeycloakAuth
//	@Router			/reports/period/pdf/ [get]
func GetPeriodReportPDF(w http.ResponseWriter, r *http.Request) {
	getReportPDF(w, r, periodFromQuery)
}

// SendPeriodReport godoc
//
//	@Summary		Mail the accounting report of a period
//	@Description	Sends the report of the period to the accounting report email from the settings
//	@Tags			Reports
//	@Param			from	query	string	true	"First day (YYYY-MM-DD)"
//	@Param			to		query	string	true	"Last day (YYYY-MM-DD)"
//	@Success		200
//	@Security		KeycloakAuth
//	@Router			/reports/period/send/ [post]
func SendPeriodReport(w http.ResponseWriter, r *http.Request) {
	sendReport(w, r, periodFromQuery)
}

// LastReport is the last day of the last report exported or sent in the backoffice
type LastReport struct {
	To string // YYYY-MM-DD, empty if there is none yet
}

// GetLastReport godoc
//
//	@Summary		Last accounting report
//	@Description	Last day of the last report exported or sent in the backoffice
//	@Tags			Reports
//	@Produce		json
//	@Success		200	{object}	LastReport
//	@Security		KeycloakAuth
//	@Router			/reports/last/ [get]
func GetLastReport(w http.ResponseWriter, r *http.Request) {
	last, err := database.Db.GetAccountingReportLastTo()
	if err != nil {
		utils.ErrorJSON(w, errors.New("loading the last report failed"), http.StatusInternalServerError)
		return
	}
	respond(w, nil, LastReport{To: last})
}

func writeReportFile(w http.ResponseWriter, filename, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Error("writeReportFile: ", err)
	}
}
