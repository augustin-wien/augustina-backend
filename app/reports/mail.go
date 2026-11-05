package reports

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
)

// BuildMail creates the mail to the accounting with the report attached as PDF and CSV
func BuildMail(r MonthlyReport, newspaperName string, recipients []string) (*mailer.EmailRequest, error) {
	if len(recipients) == 0 {
		return nil, errors.New("no recipients for the accounting report")
	}

	subject := "Monatsbericht " + r.MonthName()
	if newspaperName != "" {
		subject = newspaperName + ": " + subject
	}

	var body strings.Builder
	body.WriteString("<p>Hallo,</p>")
	fmt.Fprintf(&body, "<p>anbei der Monatsbericht für %s als PDF und CSV.</p>", html.EscapeString(r.MonthName()))
	body.WriteString(`<table cellpadding="4" style="border-collapse:collapse">`)
	for _, row := range r.summaryRows() {
		label := html.EscapeString(row.label)
		if row.detail {
			label = "&nbsp;&nbsp;&nbsp;&nbsp;" + label
		} else {
			label = "<strong>" + label + "</strong>"
		}
		fmt.Fprintf(&body, `<tr><td>%s</td><td style="text-align:right">%s</td><td>%s</td></tr>`,
			label, html.EscapeString(FormatEuro(row.amount)), html.EscapeString(row.count))
	}
	body.WriteString("</table>")
	body.WriteString("<p>Diese Mail wurde automatisch erstellt.</p>")

	m, err := mailer.NewRequest(recipients, subject, body.String())
	if err != nil {
		return nil, err
	}

	pdf, err := r.PDF(newspaperName)
	if err != nil {
		return nil, err
	}
	csv, err := r.CSV()
	if err != nil {
		return nil, err
	}
	if err = m.AddAttachment(r.FileName()+".pdf", "application/pdf", pdf); err != nil {
		return nil, err
	}
	if err = m.AddAttachment(r.FileName()+".csv", "text/csv; charset=utf-8", csv); err != nil {
		return nil, err
	}
	return m, nil
}

// ErrNoRecipients is returned when no accounting report email is configured
var ErrNoRecipients = errors.New("no accounting report email configured")

// SendMonthly builds the report of the month (YYYY-MM) and mails it to the configured recipients with send,
// mailer.Send or mailer.SendWithRetry
func SendMonthly(month string, send func(*mailer.EmailRequest) (bool, error)) error {
	settings, err := database.Db.GetSettings()
	if err != nil {
		return err
	}
	recipients, err := ParseRecipients(settings.AccountingReportEmail)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return ErrNoRecipients
	}
	r, err := Build(month)
	if err != nil {
		return err
	}
	m, err := BuildMail(r, settings.NewspaperName, recipients)
	if err != nil {
		return err
	}
	success, err := send(m)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("sending the accounting report failed")
	}
	return nil
}
