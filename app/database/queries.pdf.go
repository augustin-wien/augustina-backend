package database

import (
	"context"
	"errors"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gopkg.in/guregu/null.v4"
)

// DeletePDF removes pdfs if their creation date is older than 6 weeks
func (db *Database) DeletePDF() (err error) {
	deleteInterval := config.Config.IntervalToDeletePDFsInWeeks
	log.Info("DeletePDF entered: ", deleteInterval)
	_, err = db.Dbpool.Exec(context.Background(), "DELETE FROM PDF WHERE timestamp < NOW() - $1 * INTERVAL '1 week'", deleteInterval)
	if err != nil {
		log.Error("DeletePDF: ", err)
		return err
	}
	return err
}

// CreatePDF creates an instance of the PDF with given path and timestamp into the database
func (db *Database) CreatePDF(pdf PDF) (pdfId int64, err error) {

	// CreatePDF creates an instance of the PDF with given path and timestamp into the database
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO PDF (Path, Timestamp) values ($1, $2) RETURNING ID", pdf.Path, pdf.Timestamp).Scan(&pdf.ID)
	if err != nil {
		log.Error("CreatePDF: failed to add to database ", err)
	}
	if err != nil {
		log.Error("CreatePDF: ", err)
	}
	log.Info("Created new PDF with ID: ", pdf.ID, " and path: ", pdf.Path)
	return int64(pdf.ID), err
}

// GetPDF returns the latest PDF from the database
func (db *Database) GetPDF() (pdf PDF, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PDF ORDER BY ID DESC LIMIT 1").Scan(&pdf.ID, &pdf.Path, &pdf.Timestamp)
	if err != nil {
		log.Error("GetPDF: ", err)
	}
	return pdf, err
}

// GetPDFByID returns the PDF with the given ID
func (db *Database) GetPDFByID(id int64) (pdf PDF, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PDF WHERE ID = $1", id).Scan(&pdf.ID, &pdf.Path, &pdf.Timestamp)
	if err != nil {
		log.Error("GetPDFByID: failed for id:", id, err)
	}
	return pdf, err
}

// CreatePDFDownload creates an instance of the PDFDownload with given linkID and timestamp into the database
func (db *Database) CreatePDFDownload(tx pgx.Tx, pdf PDF, orderId, itemId int) (pdfDownload PDFDownload, err error) {
	// generate download id
	linkID := uuid.New()
	pdfDownload = PDFDownload{
		LinkID:        linkID.String(),
		PDF:           pdf.ID,
		Timestamp:     time.Now(),
		EmailSent:     false,
		LastDownload:  time.Time{},
		DownloadCount: 0,
		OrderID:       null.IntFrom(int64(orderId)),
		ItemID:        null.IntFrom(int64(itemId)),
	}

	// CreatePDF creates an instance of the PDF with given path and timestamp into the database
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO PDFDownload (LinkID, PDF, Timestamp, OrderId, ItemId) values ($1, $2, $3, $4, $5) RETURNING ID", pdfDownload.LinkID, pdfDownload.PDF, pdfDownload.Timestamp, pdfDownload.OrderID, pdfDownload.ItemID).Scan(&pdfDownload.ID)
	if err != nil {
		log.Error("CreatePDFDownload: ", err)
	}
	return
}

// GetPDFDownload returns the latest PDFDownload from the database
func (db *Database) GetPDFDownload(linkID string) (pdfDownload PDFDownload, err error) {
	if len(linkID) == 0 {
		return pdfDownload, errors.New("linkID is empty")
	}
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PDFDownload WHERE LinkID = $1", linkID).Scan(&pdfDownload.ID, &pdfDownload.PDF, &pdfDownload.LinkID, &pdfDownload.Timestamp, &pdfDownload.EmailSent, &pdfDownload.OrderID, &pdfDownload.LastDownload, &pdfDownload.DownloadCount, &pdfDownload.ItemID)
	if err != nil {
		log.Error("GetPDFDownload: ", linkID, "err: ", err)
	}
	return pdfDownload, err
}

// GetPDFDownload returns the latest PDFDownload from the database
func (db *Database) GetPDFDownloadTx(tx pgx.Tx, linkID string) (pdfDownload PDFDownload, err error) {
	if len(linkID) == 0 {
		return pdfDownload, errors.New("linkID is empty")
	}
	err = tx.QueryRow(context.Background(), "SELECT * FROM PDFDownload WHERE LinkID = $1", linkID).Scan(&pdfDownload.ID, &pdfDownload.PDF, &pdfDownload.LinkID, &pdfDownload.Timestamp, &pdfDownload.EmailSent, &pdfDownload.OrderID, &pdfDownload.LastDownload, &pdfDownload.DownloadCount, &pdfDownload.ItemID)
	if err != nil {
		log.Error("GetPDFDownload: ", linkID, "err: ", err)
	}
	return pdfDownload, err
}

// DeletePDFDownload removes pdfs if their creation date is older than 6 weeks
func (db *Database) DeletePDFDownload() (err error) {
	// Get interval from config
	deleteInterval := config.Config.IntervalToDeletePDFsInWeeks
	_, err = db.Dbpool.Exec(context.Background(), "DELETE FROM PDFDownload WHERE timestamp < NOW() - $1 * INTERVAL '1 week'", deleteInterval)
	if err != nil {
		log.Error("DeletePDFDownload: ", err)
		return err
	}
	return err
}

func (db *Database) UpdatePdfDownloadTx(tx pgx.Tx, pdfDownload PDFDownload) (err error) {
	_, err = tx.Exec(context.Background(), `
	UPDATE PDFDownload SET PDF = $1, LinkID = $2, Timestamp = $3, EmailSent = $4, OrderID = $5, LastDownload = $6, DownloadCount = $7, ItemId = $8 WHERE ID = $9`,
		pdfDownload.PDF, pdfDownload.LinkID, pdfDownload.Timestamp, pdfDownload.EmailSent, pdfDownload.OrderID, pdfDownload.LastDownload, pdfDownload.DownloadCount, pdfDownload.ItemID, pdfDownload.ID)
	if err != nil {
		log.Error("UpdatePdfDownload: ", err)
	}
	return err
}

func (db *Database) UpdatePdfDownload(pdfDownload PDFDownload) (err error) {
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("UpdatePdfDownload: failed to start transaction ", err)
		return
	}
	defer func() {
		err = DeferTx(tx, err)
		if err != nil {

			log.Error("UpdatePdfDownload: reached defer error ", err)
		}
	}()
	log.Info("UpdatePdfDownload: pdfDownload: ", pdfDownload)
	return db.UpdatePdfDownloadTx(tx, pdfDownload)
}

func (db *Database) GetPDFDownloadByOrderIdTx(tx pgx.Tx, order int) (pdfDownload []PDFDownload, err error) {
	rows, err := tx.Query(context.Background(), "SELECT * FROM PDFDownload WHERE OrderId = $1", order)
	if err != nil {
		log.Error("GetPDFDownloadByOrderIdTx: ", err)
		return pdfDownload, err
	}
	defer rows.Close()
	for rows.Next() {
		var nextPdfDownload PDFDownload
		err = rows.Scan(&nextPdfDownload.ID, &nextPdfDownload.PDF, &nextPdfDownload.LinkID, &nextPdfDownload.Timestamp, &nextPdfDownload.EmailSent, &nextPdfDownload.OrderID, &nextPdfDownload.LastDownload, &nextPdfDownload.DownloadCount, &nextPdfDownload.ItemID)
		if err != nil {
			log.Error("GetPDFDownloadByOrderIdTx: ", err)
			return pdfDownload, err
		}
		pdfDownload = append(pdfDownload, nextPdfDownload)
	}
	return pdfDownload, nil
}

func (db *Database) GetPDFDownloadByOrderId(order int) (pdfDownload []PDFDownload, err error) {
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return
	}
	defer func() {
		err = DeferTx(tx, err)
		if err != nil {

			log.Error("GetPDFDownloadByOrderId: reached defer error ", err)
		}
	}()
	return db.GetPDFDownloadByOrderIdTx(tx, order)
}

func (db *Database) GetPDFDownloadByOrderIdAndItemTx(tx pgx.Tx, order int, item int) (pdfDownload PDFDownload, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PDFDownload WHERE OrderId = $1 AND ItemId = $2", order, item).Scan(&pdfDownload.ID, &pdfDownload.PDF, &pdfDownload.LinkID, &pdfDownload.Timestamp, &pdfDownload.EmailSent, &pdfDownload.OrderID, &pdfDownload.LastDownload, &pdfDownload.DownloadCount, &pdfDownload.ItemID)
	return pdfDownload, err
}
