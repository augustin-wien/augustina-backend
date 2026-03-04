package database

import (
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

	// Delete
	err = Db.DeletePDF()
	require.NoError(t, err)

	// Verify old deleted
	_, err = Db.GetPDFByID(idOld)
	require.Error(t, err) // Should be not found

	// Verify new exists
	found, err := Db.GetPDFByID(idNew)
	require.NoError(t, err)
	require.Equal(t, int64(found.ID), int64(found.ID))
}
