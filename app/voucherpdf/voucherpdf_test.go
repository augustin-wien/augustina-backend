package voucherpdf

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	out, err := Generate([]Voucher{{
		Code:           "ABCD-EFGH-JKLM",
		ItemName:       "Online-Abo",
		DurationMonths: 12,
		RecipientName:  "Grete Müller",
		Message:        "Alles Gute zum Geburtstag! Viel Freude beim Lesen – und liebe Grüße aus Wien.",
		RedeemURL:      "https://shop.example.com/einloesen",
		QRCodeURL:      "https://shop.example.com/einloesen?code=ABCD-EFGH-JKLM",
		NewspaperName:  "Augustin",
	}, {
		Code:           "MNPQ-RSTU-VWXY",
		ItemName:       "Online-Abo",
		DurationMonths: 6,
		RedeemURL:      "https://shop.example.com/einloesen",
		QRCodeURL:      "https://shop.example.com/einloesen?code=MNPQ-RSTU-VWXY",
	}})
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(out, []byte("%PDF")))
	if path := os.Getenv("VOUCHER_PDF_OUT"); path != "" {
		require.NoError(t, os.WriteFile(path, out, 0o600))
	}
}

func TestGenerateWithoutVouchers(t *testing.T) {
	_, err := Generate(nil)
	require.Error(t, err)
}

func TestDurationText(t *testing.T) {
	require.Equal(t, "1 Jahr", durationText(12))
	require.Equal(t, "2 Jahre", durationText(24))
	require.Equal(t, "6 Monate", durationText(6))
	require.Equal(t, "1 Monat", durationText(1))
}
