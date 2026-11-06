package reports

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
)

var monthNames = [...]string{"Jänner", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}

// MonthName returns the month of the report in German, e.g. "Oktober 2026"
func (r MonthlyReport) MonthName() string {
	return fmt.Sprintf("%s %d", monthNames[r.From.In(Location).Month()-1], r.From.In(Location).Year())
}

// FileName returns the name of the exported file without extension
func (r MonthlyReport) FileName() string {
	return "monatsbericht_" + r.Month
}

// FormatEuro formats cents in the Austrian notation, e.g. 1.234,56 €
func FormatEuro(cents int) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	euros := strconv.Itoa(cents / 100)
	var grouped strings.Builder
	for i, c := range euros {
		if i > 0 && (len(euros)-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(c)
	}
	return fmt.Sprintf("%s%s,%02d €", sign, grouped.String(), cents%100)
}

// csvAmount formats cents for spreadsheets with a German locale, e.g. 1234,56
func csvAmount(cents int) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d,%02d", sign, cents/100, cents%100)
}

type summaryRow struct {
	label  string
	amount int
	count  string // optional number of payments, orders or licenses
	detail bool   // part of the row above it
}

func (r MonthlyReport) summaryRows() []summaryRow {
	return []summaryRow{
		{label: "Einnahmen gesamt", amount: r.Income},
		{label: "davon Webshop (Bestellungen)", amount: r.OnlineSales, count: strconv.Itoa(r.OnlineOrders), detail: true},
		{label: "davon Kassa bar", amount: r.POSCash, detail: true},
		{label: "davon externe Zahlungen (z.B. Kartenterminal)", amount: r.ExternalPayments, detail: true},
		{label: "Lizenzeinnahmen (Lizenzen)", amount: r.LicenseIncome, count: strconv.Itoa(r.LicenseQuantity)},
		{label: "Transaktionsgebühren", amount: r.TransactionCosts},
		{label: "davon von der Organisation übernommen", amount: r.TransactionCostsCoveredByOrga, detail: true},
		{label: "Auszahlungen an Verkäufer*innen (Auszahlungen)", amount: r.Payouts, count: strconv.Itoa(r.PayoutCount)},
		{label: "Kassa mit Guthaben verrechnet (kein Geldfluss)", amount: r.POSCredit},
	}
}

// CSV exports the report separated by semicolons, as spreadsheets with a German locale expect
func (r MonthlyReport) CSV() ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("\xEF\xBB\xBF") // BOM, so Excel reads UTF-8
	w := csv.NewWriter(&b)
	w.Comma = ';'

	rows := [][]string{
		{"Monatsbericht", r.MonthName()},
		{"Zeitraum", r.From.In(Location).Format("02.01.2006"), r.To.AddDate(0, 0, -1).In(Location).Format("02.01.2006")},
		{},
		{"Position", "Betrag (EUR)", "Anzahl"},
	}
	for _, row := range r.summaryRows() {
		rows = append(rows, []string{row.label, csvAmount(row.amount), row.count})
	}
	rows = append(rows, []string{}, []string{"Webshop-Verkäufe nach Produkt"}, []string{"Produkt", "Betrag (EUR)", "Anzahl"})
	for _, item := range r.Items {
		rows = append(rows, []string{item.Name, csvAmount(item.Amount), strconv.Itoa(item.Quantity)})
	}
	if err := w.WriteAll(rows); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// PDF exports the report as an A4 page
func (r MonthlyReport) PDF(newspaperName string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(true, 20)
	tr := pdf.UnicodeTranslatorFromDescriptor("cp1252")
	pdf.SetTitle(tr("Monatsbericht "+r.MonthName()), false)
	pdf.AddPage()

	pageWidth, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	width := pageWidth - left - right
	amountWidth := 40.0
	countWidth := 40.0
	labelWidth := width - amountWidth - countWidth

	pdf.SetFont("Helvetica", "B", 18)
	title := "Monatsbericht " + r.MonthName()
	if newspaperName != "" {
		title = newspaperName + " – " + title
	}
	pdf.MultiCell(width, 9, tr(title), "", "L", false)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(90, 90, 90)
	period := fmt.Sprintf("Zeitraum %s – %s", r.From.In(Location).Format("02.01.2006"), r.To.AddDate(0, 0, -1).In(Location).Format("02.01.2006"))
	pdf.CellFormat(width, 6, tr(period), "", 1, "L", false, 0, "")
	if !r.GeneratedAt.IsZero() {
		pdf.CellFormat(width, 6, tr("Erstellt am "+r.GeneratedAt.In(Location).Format("02.01.2006 15:04")), "", 1, "L", false, 0, "")
	}
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(6)

	header := func(cols ...string) {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetFillColor(235, 235, 235)
		pdf.CellFormat(labelWidth, 8, tr(cols[0]), "B", 0, "L", true, 0, "")
		pdf.CellFormat(amountWidth, 8, tr(cols[1]), "B", 0, "R", true, 0, "")
		pdf.CellFormat(countWidth, 8, tr(cols[2]), "B", 1, "R", true, 0, "")
	}

	header("Position", "Betrag", "Anzahl")
	for _, row := range r.summaryRows() {
		label := row.label
		if row.detail {
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(90, 90, 90)
			label = "    " + label
		} else {
			pdf.SetFont("Helvetica", "B", 10)
			pdf.SetTextColor(0, 0, 0)
		}
		pdf.CellFormat(labelWidth, 7, tr(label), "B", 0, "L", false, 0, "")
		pdf.CellFormat(amountWidth, 7, tr(FormatEuro(row.amount)), "B", 0, "R", false, 0, "")
		pdf.CellFormat(countWidth, 7, tr(row.count), "B", 1, "R", false, 0, "")
	}
	pdf.SetTextColor(0, 0, 0)

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(width, 8, tr("Webshop-Verkäufe nach Produkt"), "", 1, "L", false, 0, "")
	header("Produkt", "Betrag", "Anzahl")
	pdf.SetFont("Helvetica", "", 10)
	if len(r.Items) == 0 {
		pdf.CellFormat(width, 7, tr("Keine Verkäufe"), "B", 1, "L", false, 0, "")
	}
	for _, item := range r.Items {
		pdf.CellFormat(labelWidth, 7, tr(item.Name), "B", 0, "L", false, 0, "")
		pdf.CellFormat(amountWidth, 7, tr(FormatEuro(item.Amount)), "B", 0, "R", false, 0, "")
		pdf.CellFormat(countWidth, 7, strconv.Itoa(item.Quantity), "B", 1, "R", false, 0, "")
	}

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(90, 90, 90)
	notes := "Einnahmen gesamt = Webshop + Kassa bar + externe Zahlungen. Lizenzeinnahmen sind der Anteil, " +
		"den Verkäufer*innen für verkaufte digitale Ausgaben an die Organisation zahlen, und bereits in den " +
		"Webshop-Einnahmen enthalten. Transaktionsgebühren sind die Gebühren der Zahlungsanbieter."
	pdf.MultiCell(width, 4, tr(notes), "", "L", false)

	var b bytes.Buffer
	if err := pdf.Output(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
