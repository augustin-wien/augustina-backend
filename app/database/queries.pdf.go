package database

import (
	"context"
	"errors"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	entpdf "github.com/augustin-wien/augustina-backend/ent/pdf"
	entpdfdownload "github.com/augustin-wien/augustina-backend/ent/pdfdownload"
	"github.com/google/uuid"
	"gopkg.in/guregu/null.v4"
)

// DeletePDF removes pdfs if their creation date is older than 6 weeks
func (db *Database) DeletePDF() (err error) {
	deleteInterval := config.Config.IntervalToDeletePDFsInWeeks
	log.Info("DeletePDF entered: ", deleteInterval)

	cutoff := time.Now().AddDate(0, 0, -deleteInterval*7)
	// We assume timestamp is stored in a format compatible with string comparison or implicit cast
	// Since schema says String but DB likely Timestamp, providing RFC3339 string should work for Postgres implicit cast
	_, err = db.EntClient.PDF.Delete().Where(entpdf.TimestampLT(cutoff.Format(time.RFC3339))).Exec(context.Background())
	if err != nil {
		log.Error("DeletePDF: ", err)
		return err
	}
	return err
}

// CreatePDF creates an instance of the PDF with given path and timestamp into the database
func (db *Database) CreatePDF(pdf PDF) (pdfId int64, err error) {
	// CreatePDF creates an instance of the PDF with given path and timestamp into the database
	created, err := db.EntClient.PDF.
		Create().
		SetPath(pdf.Path).
		SetTimestamp(pdf.Timestamp.Format(time.RFC3339)).
		Save(context.Background())

	if err != nil {
		log.Error("CreatePDF: failed to add to database ", err)
		return 0, err
	}

	log.Info("Created new PDF with ID: ", created.ID, " and path: ", created.Path)
	return int64(created.ID), err
}

// GetPDF returns the latest PDF from the database
func (db *Database) GetPDF() (pdf PDF, err error) {
	result, err := db.EntClient.PDF.Query().Order(ent.Desc(entpdf.FieldID)).First(context.Background())
	if err != nil {
		log.Error("GetPDF: ", err)
		return pdf, err
	}

	ts, _ := time.Parse(time.RFC3339, result.Timestamp)
	return PDF{
		ID:        result.ID,
		Path:      result.Path,
		Timestamp: ts,
	}, nil
}

// GetPDFByID returns the PDF with the given ID
func (db *Database) GetPDFByID(id int64) (pdf PDF, err error) {
	result, err := db.EntClient.PDF.Get(context.Background(), int(id))
	if err != nil {
		log.Error("GetPDFByID: failed for id:", id, err)
		return pdf, err
	}

	ts, _ := time.Parse(time.RFC3339, result.Timestamp)
	return PDF{
		ID:        result.ID,
		Path:      result.Path,
		Timestamp: ts,
	}, nil
}

// CreatePDFDownload creates an instance of the PDFDownload with given linkID and timestamp into the database
func (db *Database) CreatePDFDownload(tx *ent.Tx, pdf PDF, orderId, itemId int) (pdfDownload PDFDownload, err error) {
	// generate download id
	linkID := uuid.New().String()

	// Use Ent to create
	created, err := tx.PDFDownload.Create().
		SetLinkID(linkID).
		SetPdfID(pdf.ID).
		SetTimestamp(time.Now()).
		SetEmailSent(false).
		SetDownloadCount(0).
		SetOrderID(orderId).
		SetItemID(itemId).
		Save(context.Background())

	if err != nil {
		log.Error("CreatePDFDownload: ", err)
		return pdfDownload, err
	}

	return db.PDFDownloadEntIntoPDFDownload(created), nil
}

// PDFDownloadEntIntoPDFDownload converts ent.PDFDownload to PDFDownload
func (db *Database) PDFDownloadEntIntoPDFDownload(p *ent.PDFDownload) PDFDownload {
	pd := PDFDownload{
		ID:            p.ID,
		LinkID:        p.LinkID,
		PDF:           p.PdfID,
		Timestamp:     p.Timestamp,
		EmailSent:     p.EmailSent,
		DownloadCount: p.DownloadCount,
	}
	if p.LastDownload != nil {
		pd.LastDownload = *p.LastDownload
	}
	if p.OrderID != nil {
		pd.OrderID = null.IntFrom(int64(*p.OrderID))
	}
	if p.ItemID != nil {
		pd.ItemID = null.IntFrom(int64(*p.ItemID))
	}
	return pd
}

// GetPDFDownload returns the latest PDFDownload from the database
func (db *Database) GetPDFDownload(linkID string) (pdfDownload PDFDownload, err error) {
	if len(linkID) == 0 {
		return pdfDownload, errors.New("linkID is empty")
	}
	deleteInterval := config.Config.IntervalToDeletePDFsInWeeks
	cutoff := time.Now().AddDate(0, 0, -deleteInterval*7)

	_, err = db.EntClient.PDFDownload.Delete().Where(entpdfdownload.TimestampLT(cutoff)).Exec(context.Background())
	if err != nil {
		log.Error("DeletePDFDownload: ", err)
		return pdfDownload, err
	}

	download, err := db.EntClient.PDFDownload.Query().Where(entpdfdownload.LinkID(linkID)).Only(context.Background())
	if err != nil {
		return pdfDownload, err
	}
	return db.PDFDownloadEntIntoPDFDownload(download), nil
}

func (db *Database) UpdatePdfDownloadTx(tx *ent.Tx, pdfDownload PDFDownload) (err error) {
	update := tx.PDFDownload.UpdateOneID(pdfDownload.ID).
		SetPdfID(pdfDownload.PDF).
		SetLinkID(pdfDownload.LinkID).
		SetTimestamp(pdfDownload.Timestamp).
		SetEmailSent(pdfDownload.EmailSent).
		SetDownloadCount(pdfDownload.DownloadCount)

	if pdfDownload.OrderID.Valid {
		update.SetOrderID(int(pdfDownload.OrderID.Int64))
	} else {
		update.ClearOrderID()
	}

	if pdfDownload.ItemID.Valid {
		update.SetItemID(int(pdfDownload.ItemID.Int64))
	} else {
		update.ClearItemID()
	}

	if !pdfDownload.LastDownload.IsZero() {
		update.SetLastDownload(pdfDownload.LastDownload)
	} else {
		update.ClearLastDownload()
	}

	err = update.Exec(context.Background())
	if err != nil {
		log.Error("UpdatePdfDownload: ", err)
	}
	return err
}

func (db *Database) UpdatePdfDownload(pdfDownload PDFDownload) (err error) {
	update := db.EntClient.PDFDownload.UpdateOneID(pdfDownload.ID).
		SetPdfID(pdfDownload.PDF).
		SetLinkID(pdfDownload.LinkID).
		SetTimestamp(pdfDownload.Timestamp).
		SetEmailSent(pdfDownload.EmailSent).
		SetDownloadCount(pdfDownload.DownloadCount)

	if pdfDownload.OrderID.Valid {
		update.SetOrderID(int(pdfDownload.OrderID.Int64))
	} else {
		update.ClearOrderID()
	}

	if pdfDownload.ItemID.Valid {
		update.SetItemID(int(pdfDownload.ItemID.Int64))
	} else {
		update.ClearItemID()
	}

	if !pdfDownload.LastDownload.IsZero() {
		update.SetLastDownload(pdfDownload.LastDownload)
	} else {
		update.ClearLastDownload()
	}

	err = update.Exec(context.Background())
	if err != nil {
		log.Error("UpdatePdfDownload: ", err)
	}
	return err
}

func (db *Database) GetPDFDownloadByOrderIdTx(tx *ent.Tx, order int) (pdfDownload []PDFDownload, err error) {
	res, err := tx.PDFDownload.Query().
		Where(entpdfdownload.OrderID(order)).
		All(context.Background())
	if err != nil {
		log.Error("GetPDFDownloadByOrderIdTx: ", err)
		return nil, err
	}

	for _, e := range res {
		pdfDownload = append(pdfDownload, db.PDFDownloadEntIntoPDFDownload(e))
	}

	return pdfDownload, nil
}

func (db *Database) GetPDFDownloadByOrderId(order int) (pdfDownload []PDFDownload, err error) {
	// Use Ent
	list, err := db.EntClient.PDFDownload.Query().Where(entpdfdownload.OrderID(order)).All(context.Background())
	if err != nil {
		log.Error("GetPDFDownloadByOrderId: ", err)
		return nil, err
	}

	for _, p := range list {
		pdfDownload = append(pdfDownload, db.PDFDownloadEntIntoPDFDownload(p))
	}
	return pdfDownload, nil
}

func (db *Database) GetPDFDownloadByOrderIdAndItemTx(tx *ent.Tx, order int, item int) (pdfDownload PDFDownload, err error) {
	entPDF, err := tx.PDFDownload.Query().
		Where(
			entpdfdownload.OrderID(order),
			entpdfdownload.ItemID(item),
		).
		Only(context.Background())

	if err != nil {
		// Log excluded to match original shortness or add if needed
		return pdfDownload, err
	}

	return db.PDFDownloadEntIntoPDFDownload(entPDF), nil
}

// GetPDFDownloadTx returns the PDFDownload with the given linkID within a transaction
func (db *Database) GetPDFDownloadTx(tx *ent.Tx, linkID string) (pdfDownload PDFDownload, err error) {
	entDL, err := tx.PDFDownload.Query().Where(entpdfdownload.LinkID(linkID)).Only(context.Background())
	if err != nil {
		return pdfDownload, err
	}
	return db.PDFDownloadEntIntoPDFDownload(entDL), nil
}
