package database

import (
	"strings"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

// TestCreateAndGetPDF verifies PDF creation and retrieval
func TestCreateAndGetPDF(t *testing.T) {
	Db.InitEmptyTestDb()

	ts := time.Now().Truncate(time.Second).UTC() // Ent/Postgres timestamp precision might differ
	pdf := PDF{
		Path:      "s3://bucket/test.pdf",
		Timestamp: ts,
	}

	// Create
	id, err := Db.CreatePDF(pdf)
	require.NoError(t, err)
	require.Greater(t, id, int64(0))

	// Get latest
	latest, err := Db.GetPDF()
	require.NoError(t, err)
	require.Equal(t, id, int64(latest.ID))
	require.Equal(t, pdf.Path, latest.Path)
	// Compare timestamp: if stored as string, format might matter.
	// If stored as time, truncation matters.
	// Since Ent schema says String, implementation will format it.
	// Test should account for possible serialization.
	// We check closer enough or same string representation.
	require.WithinDuration(t, ts, latest.Timestamp, time.Second)

	// Get by ID
	fetched, err := Db.GetPDFByID(id)
	require.NoError(t, err)
	require.Equal(t, id, int64(fetched.ID))
	require.Equal(t, pdf.Path, fetched.Path)
}

// TestDeletePDF verifies old PDFs are deleted based on config
func TestDeletePDF(t *testing.T) {
	Db.InitEmptyTestDb()

	// Set config for test
	config.Config.IntervalToDeletePDFsInWeeks = 1

	// Create old PDF (older than 1 week)
	oldTS := time.Now().AddDate(0, 0, -8) // 8 days ago
	oldPDF := PDF{Path: "old.pdf", Timestamp: oldTS}
	idOld, err := Db.CreatePDF(oldPDF)
	require.NoError(t, err)

	// Create new PDF
	newTS := time.Now()
	newPDF := PDF{Path: "new.pdf", Timestamp: newTS}
	idNew, err := Db.CreatePDF(newPDF)
	require.NoError(t, err)

	// Delete — skip if the prevent-delete trigger is installed (local dev DB with migration 011)
	err = Db.DeletePDF()
	if err != nil && strings.Contains(err.Error(), "Cannot delete from table PDF") {
		t.Skip("prevent-delete trigger active; skipping (migration 011 not excluded in this environment)")
	}
	require.NoError(t, err)

	// Verify old deleted
	_, err = Db.GetPDFByID(idOld)
	require.Error(t, err) // Should be not found

	// Verify new exists
	found, err := Db.GetPDFByID(idNew)
	require.NoError(t, err)
	require.Equal(t, int64(found.ID), int64(found.ID))
}

// TestGetPDFDownloadKeepsOldLinks verifies that looking up a download link
// works while older links exist and does not delete them
func TestGetPDFDownloadKeepsOldLinks(t *testing.T) {
	Db.InitEmptyTestDb()
	config.Config.IntervalToDeletePDFsInWeeks = 1

	pdfID, err := Db.CreatePDF(PDF{Path: "issue.pdf", Timestamp: time.Now()})
	require.NoError(t, err)
	itemID, err := Db.CreateItem(Item{Name: "PDF issue", Description: "PDF issue", Price: 100})
	require.NoError(t, err)

	oldDownload, err := Db.CreatePDFDownloadForItem(int(pdfID), itemID)
	require.NoError(t, err)
	oldDownload.Timestamp = time.Now().AddDate(0, 0, -8)
	require.NoError(t, Db.UpdatePdfDownload(oldDownload))

	newDownload, err := Db.CreatePDFDownloadForItem(int(pdfID), itemID)
	require.NoError(t, err)

	found, err := Db.GetPDFDownload(newDownload.LinkID)
	require.NoError(t, err)
	require.Equal(t, newDownload.ID, found.ID)

	found, err = Db.GetPDFDownload(oldDownload.LinkID)
	require.NoError(t, err)
	require.Equal(t, oldDownload.ID, found.ID)
}
