// Package voucherpdf renders the printable PDF of an abonement gift voucher.
package voucherpdf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boombuler/barcode/qr"
	"github.com/go-pdf/fpdf"
	"github.com/go-pdf/fpdf/contrib/barcode"
)

// Voucher holds everything printed on a voucher PDF
type Voucher struct {
	Code           string
	ItemName       string
	DurationMonths int
	RecipientName  string
	Message        string
	// RedeemURL is the redemption page without the code; it is printed as text
	RedeemURL string
	// QRCodeURL is the redemption page with the code filled in; it is encoded in the QR code
	QRCodeURL     string
	NewspaperName string
	// LogoPath is a local PNG or JPEG file; it is left out when empty or unreadable
	LogoPath string
}

const (
	pageWidth   = 210.0
	marginX     = 20.0
	contentW    = pageWidth - 2*marginX
	maxMsgLines = 8
)

// Generate renders one A4 page per voucher into a single PDF
func Generate(vouchers []Voucher) ([]byte, error) {
	if len(vouchers) == 0 {
		return nil, fmt.Errorf("no vouchers to render")
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginX, 20, marginX)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle("Gutschein", true)
	tr := pdf.UnicodeTranslatorFromDescriptor("cp1252")

	for _, v := range vouchers {
		renderPage(pdf, tr, v)
	}
	if err := pdf.Error(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderPage(pdf *fpdf.Fpdf, tr func(string) string, v Voucher) {
	pdf.AddPage()

	y := 20.0
	if logoType := imageType(v.LogoPath); logoType != "" {
		opts := fpdf.ImageOptions{ImageType: logoType, ReadDpi: true}
		info := pdf.RegisterImageOptions(v.LogoPath, opts)
		if pdf.Ok() && info != nil && info.Height() > 0 {
			h := 22.0
			w := h * info.Width() / info.Height()
			if w > contentW {
				w = contentW
				h = w * info.Height() / info.Width()
			}
			pdf.ImageOptions(v.LogoPath, (pageWidth-w)/2, y, w, h, false, opts, 0, "")
			y += h + 8
		} else {
			// A broken logo must not cost the customer their voucher
			pdf.ClearError()
		}
	} else if v.NewspaperName != "" {
		pdf.SetFont("Helvetica", "B", 20)
		pdf.SetXY(marginX, y)
		pdf.CellFormat(contentW, 10, tr(v.NewspaperName), "", 0, "C", false, 0, "")
		y += 16
	}

	pdf.SetFont("Helvetica", "B", 36)
	pdf.SetXY(marginX, y)
	pdf.CellFormat(contentW, 16, "Gutschein", "", 0, "C", false, 0, "")
	y += 18

	pdf.SetFont("Helvetica", "", 15)
	pdf.SetXY(marginX, y)
	pdf.CellFormat(contentW, 8, tr(fmt.Sprintf("%s · %s", v.ItemName, durationText(v.DurationMonths))), "", 0, "C", false, 0, "")
	y += 16

	if name := strings.TrimSpace(v.RecipientName); name != "" {
		pdf.SetFont("Helvetica", "", 13)
		pdf.SetXY(marginX, y)
		pdf.CellFormat(contentW, 7, tr("Für"), "", 0, "C", false, 0, "")
		y += 8
		pdf.SetFont("Helvetica", "B", 22)
		pdf.SetXY(marginX, y)
		pdf.CellFormat(contentW, 11, tr(name), "", 0, "C", false, 0, "")
		y += 16
	}

	if msg := strings.TrimSpace(v.Message); msg != "" {
		pdf.SetFont("Helvetica", "I", 13)
		lines := pdf.SplitLines([]byte(tr(msg)), contentW-20)
		if len(lines) > maxMsgLines {
			lines = lines[:maxMsgLines]
		}
		for _, line := range lines {
			pdf.SetXY(marginX+10, y)
			pdf.CellFormat(contentW-20, 6.5, string(line), "", 0, "C", false, 0, "")
			y += 6.5
		}
		y += 8
	}

	// QR code with the code below it
	qrSize := 55.0
	if y > 150 {
		qrSize = 45
	}
	key := barcode.RegisterQR(pdf, v.QRCodeURL, qr.M, qr.Unicode)
	barcode.Barcode(pdf, key, (pageWidth-qrSize)/2, y, qrSize, qrSize, false)
	y += qrSize + 6

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(marginX, y)
	pdf.CellFormat(contentW, 6, "Gutscheincode", "", 0, "C", false, 0, "")
	y += 7
	pdf.SetFont("Courier", "B", 24)
	pdf.SetXY(marginX, y)
	pdf.CellFormat(contentW, 12, v.Code, "", 0, "C", false, 0, "")

	// Instructions at the bottom of the page
	y = 240
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(marginX, y, pageWidth-marginX, y)
	y += 6
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(marginX, y)
	pdf.CellFormat(contentW, 6, tr("So löst du den Gutschein ein"), "", 0, "L", false, 0, "")
	y += 8
	pdf.SetFont("Helvetica", "", 10)
	steps := []string{
		"1. QR-Code scannen oder " + v.RedeemURL + " öffnen.",
		"2. Gutscheincode eingeben, falls er nicht schon ausgefüllt ist.",
		"3. E-Mail-Adresse angeben – damit meldest du dich später zum Lesen an.",
	}
	for _, s := range steps {
		pdf.SetXY(marginX, y)
		pdf.MultiCell(contentW, 5, tr(s), "", "L", false)
		y = pdf.GetY() + 1
	}
	y += 3
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(90, 90, 90)
	pdf.SetXY(marginX, y)
	pdf.MultiCell(contentW, 4.5, tr(fmt.Sprintf("Das Abo läuft %s ab dem Tag der Einlösung. Der Gutschein ist unbegrenzt gültig und kann einmal eingelöst werden.", durationText(v.DurationMonths))), "", "L", false)
	pdf.SetTextColor(0, 0, 0)
}

func durationText(months int) string {
	switch {
	case months == 12:
		return "1 Jahr"
	case months > 12 && months%12 == 0:
		return fmt.Sprintf("%d Jahre", months/12)
	case months == 1:
		return "1 Monat"
	default:
		return fmt.Sprintf("%d Monate", months)
	}
}

func imageType(path string) string {
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "PNG"
	case ".jpg", ".jpeg":
		return "JPG"
	}
	return ""
}
