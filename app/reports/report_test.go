package reports

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func testReport(t *testing.T) MonthlyReport {
	t.Helper()
	items := []database.Item{
		{ID: 10, Name: "Zeitung", Type: "issue"},
		{ID: 11, Name: "Digitale Ausgabe", Type: "online_issue"},
		{ID: 93, Name: "donation", Type: "donation"},
		{ID: 94, Name: "transactionCosts", Type: "transaction_costs"},
		{ID: 95, Name: "Digitale Lizenz", Type: "license_item", IsLicenseItem: true},
	}
	accounts := []database.Account{
		{ID: 1, Type: "UserAnon"},
		{ID: 2, Type: "Vendor", Vendor: null.IntFrom(7)},
		{ID: 3, Type: "UserAuth"},
		{ID: 4, Type: "VivaWallet"},
		{ID: 5, Type: "Orga"},
		{ID: 6, Type: "Cash"},
		{ID: 8, Type: "Backoffice"},
	}
	from, to, err := ParseMonth("2026-10")
	require.NoError(t, err)
	day := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	payments := []database.Payment{
		// 23:30 on Sep 30 in Vienna, before the month
		{Item: null.IntFrom(10), Sender: 1, Receiver: 2, IsSale: true, Quantity: 1, Amount: 300, Timestamp: time.Date(2026, 9, 30, 21, 30, 0, 0, time.UTC)},
		// Webshop sales
		{Item: null.IntFrom(10), Sender: 1, Receiver: 2, IsSale: true, Quantity: 2, Amount: 600, Order: null.IntFrom(1), Timestamp: day},
		{Item: null.IntFrom(11), Sender: 3, Receiver: 2, IsSale: true, Quantity: 1, Amount: 400, Order: null.IntFrom(2), Timestamp: day},
		{Item: null.IntFrom(93), Sender: 1, Receiver: 2, IsSale: true, Quantity: 150, Amount: 150, Order: null.IntFrom(2), Timestamp: day},
		// The license of the digital issue
		{Item: null.IntFrom(95), Sender: 2, Receiver: 5, Quantity: 1, Amount: 50, Order: null.IntFrom(2), Timestamp: day},
		// Transaction costs taken over by the organisation are booked twice
		{Item: null.IntFrom(94), Sender: 2, Receiver: 4, Quantity: 27, Amount: 27, Order: null.IntFrom(1), Timestamp: day},
		{Item: null.IntFrom(94), Sender: 5, Receiver: 2, Quantity: 27, Amount: 27, Order: null.IntFrom(1), Timestamp: day},
		// Payouts
		{Sender: 2, Receiver: 6, Amount: 1200, Timestamp: day},
		{Sender: 2, Receiver: 6, Amount: 300, Timestamp: day},
		// POS: part cash, part credit, plus the bookkeeping-only item record
		{Sender: 6, Receiver: 8, Amount: 200, IsPOS: true, Timestamp: day},
		{Sender: 2, Receiver: 5, Amount: 100, IsPOS: true, Timestamp: day},
		{Sender: 5, Receiver: 8, Amount: 100, IsPOS: true, Timestamp: day},
		{Item: null.IntFrom(10), Sender: 2, Receiver: 8, IsSale: true, IsPOS: true, Quantity: 1, Amount: 300, Timestamp: day},
		// First minute of November in Vienna
		{Sender: 2, Receiver: 6, Amount: 999, Timestamp: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)},
	}
	external := []*ent.ExternalPayment{{Amount: 1000, Fee: 15}}

	return build(from, to, items, accounts, payments, external)
}

func TestBuild(t *testing.T) {
	r := testReport(t)

	require.Equal(t, "2026-10", r.Month)
	require.Equal(t, 1150, r.OnlineSales)
	require.Equal(t, 2, r.OnlineOrders)
	require.Equal(t, 200, r.POSCash)
	require.Equal(t, 100, r.POSCredit)
	require.Equal(t, 1000, r.ExternalPayments)
	require.Equal(t, 2350, r.Income)
	require.Equal(t, 50, r.LicenseIncome)
	require.Equal(t, 1, r.LicenseQuantity)
	require.Equal(t, 42, r.TransactionCosts)
	require.Equal(t, 27, r.TransactionCostsCoveredByOrga)
	require.Equal(t, 1500, r.Payouts)
	require.Equal(t, 2, r.PayoutCount)

	require.Equal(t, []ItemLine{
		{ItemID: 10, Name: "Zeitung", Type: "issue", Quantity: 2, Amount: 600},
		{ItemID: 11, Name: "Digitale Ausgabe", Type: "online_issue", Quantity: 1, Amount: 400},
		{ItemID: 93, Name: "donation", Type: "donation", Quantity: 1, Amount: 150},
	}, r.Items)
}

func TestParseMonth(t *testing.T) {
	from, to, err := ParseMonth("2026-12")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 12, 1, 0, 0, 0, 0, Location), from)
	require.Equal(t, time.Date(2027, 1, 1, 0, 0, 0, 0, Location), to)

	_, _, err = ParseMonth("2026-13")
	require.Error(t, err)
	_, _, err = ParseMonth("")
	require.Error(t, err)
}

func TestPreviousMonth(t *testing.T) {
	require.Equal(t, "2026-12", PreviousMonth(time.Date(2027, 1, 1, 0, 30, 0, 0, Location)))
	// 23:30 UTC on Oct 31 is already November in Vienna
	require.Equal(t, "2026-10", PreviousMonth(time.Date(2026, 10, 31, 23, 30, 0, 0, time.UTC)))
	require.Equal(t, "2026-02", PreviousMonth(time.Date(2026, 3, 31, 12, 0, 0, 0, Location)))
}

func TestParseRecipients(t *testing.T) {
	r, err := ParseRecipients(" buchhaltung@example.com, Chefin <chefin@example.com>,, ")
	require.NoError(t, err)
	require.Equal(t, []string{"buchhaltung@example.com", "chefin@example.com"}, r)

	r, err = ParseRecipients("")
	require.NoError(t, err)
	require.Empty(t, r)

	_, err = ParseRecipients("not an address")
	require.Error(t, err)
}

func TestFormatEuro(t *testing.T) {
	require.Equal(t, "0,00 €", FormatEuro(0))
	require.Equal(t, "0,05 €", FormatEuro(5))
	require.Equal(t, "999,99 €", FormatEuro(99999))
	require.Equal(t, "1.234,56 €", FormatEuro(123456))
	require.Equal(t, "1.234.567,00 €", FormatEuro(123456700))
	require.Equal(t, "-12,30 €", FormatEuro(-1230))
}

func TestCSV(t *testing.T) {
	data, err := testReport(t).CSV()
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("\xEF\xBB\xBF")))

	csv := string(data)
	require.Contains(t, csv, "Monatsbericht;Oktober 2026\n")
	require.Contains(t, csv, "Zeitraum;01.10.2026;31.10.2026\n")
	require.Contains(t, csv, "Einnahmen gesamt;23,50;\n")
	require.Contains(t, csv, "Lizenzeinnahmen (Lizenzen);0,50;1\n")
	require.Contains(t, csv, "Transaktionsgebühren;0,42;\n")
	require.Contains(t, csv, "Auszahlungen an Verkäufer*innen (Auszahlungen);15,00;2\n")
	require.Contains(t, csv, "Zeitung;6,00;2\n")
}

func TestPDF(t *testing.T) {
	data, err := testReport(t).PDF("Augustin")
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func TestBuildMail(t *testing.T) {
	m, err := BuildMail(testReport(t), "Augustin", []string{"buchhaltung@example.com"})
	require.NoError(t, err)
	require.Equal(t, "Augustin: Monatsbericht Oktober 2026", m.Subject())
	require.Contains(t, m.Body(), "23,50 €")

	attachments := m.Attachments()
	require.Len(t, attachments, 2)
	require.Equal(t, "monatsbericht_2026-10.pdf", attachments[0].Filename)
	require.Equal(t, "monatsbericht_2026-10.csv", attachments[1].Filename)

	msg, _, err := m.BuildMessage()
	require.NoError(t, err)
	require.True(t, strings.Contains(string(msg), "Content-Type: multipart/mixed;"))
	require.Contains(t, string(msg), `Content-Disposition: attachment; filename="monatsbericht_2026-10.pdf"`)

	_, err = BuildMail(testReport(t), "Augustin", nil)
	require.Error(t, err)
}
