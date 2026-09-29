package database

import (
	"context"
	"errors"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/wordpress"
)

var (
	// ErrOrderNotVerified is returned when mails are requested for an order that was never paid
	ErrOrderNotVerified = errors.New("order is not verified")
	// ErrOrderHasNoCustomer is returned when an order has no customer email to send mails to
	ErrOrderHasNoCustomer = errors.New("order has no customer email")
	// ErrOrderHasNoDigitalItems is returned when an order contains nothing that is delivered by mail
	ErrOrderHasNoDigitalItems = errors.New("order has no digital items")
)

// buildDigitalLicenceMail builds the mail that links the customer to the online paper
func (db *Database) buildDigitalLicenceMail(email, inviteURL string) (*mailer.EmailRequest, error) {
	templateData := struct {
		URL       string
		EMAIL     string
		InviteURL string
	}{
		URL:       config.Config.OnlinePaperUrl,
		EMAIL:     email,
		InviteURL: inviteURL,
	}
	return db.BuildEmailRequestFromTemplate("digitalLicenceItemTemplate.html", []string{email}, templateData)
}

// buildPDFLicenceMail builds the mail with the download link of a purchased PDF
func (db *Database) buildPDFLicenceMail(email, linkID string) (*mailer.EmailRequest, error) {
	templateData := struct {
		URL   string
		EMAIL string
	}{
		URL:   config.Config.FrontendURL + "/pdf/" + linkID,
		EMAIL: email,
	}
	return db.BuildEmailRequestFromTemplate("PDFLicenceItemTemplate.html", []string{email}, templateData)
}

// isMailDeliveredItem reports whether buying the item sends the customer a mail
func isMailDeliveredItem(item Item) bool {
	return item.LicenseItem.Valid || item.Type == "abonement"
}

// sendMail sends a mail and turns an unsuccessful send into an error
func sendMail(m *mailer.EmailRequest) error {
	success, err := mailer.Send(m)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("mail was not sent")
	}
	return nil
}

// ResendOrderMails sends the digital licence and PDF download mails of a
// verified order again, e.g. when the customer lost them or sending failed.
// Keycloak users and license groups are left alone, they were set up when the
// order was verified. PDF download links are reused and valid for another six
// weeks from now; a missing link is created. Returns the number of mails sent.
func (db *Database) ResendOrderMails(orderID int) (sent int, err error) {
	o, err := db.GetOrderByID(orderID)
	if err != nil {
		return 0, err
	}
	if !o.Verified {
		return 0, ErrOrderNotVerified
	}
	if !o.CustomerEmail.Valid || o.CustomerEmail.String == "" {
		return 0, ErrOrderHasNoCustomer
	}
	email := o.CustomerEmail.String

	var mails []*mailer.EmailRequest
	digitalMailAdded := false
	pdfItemsDone := make(map[int]bool)
	for _, entry := range o.Entries {
		item, err := db.GetItem(entry.Item)
		if err != nil {
			return 0, err
		}
		if !isMailDeliveredItem(item) {
			continue
		}

		if !item.IsPDFItem {
			// Like on verification, one online paper mail per order
			if digitalMailAdded {
				continue
			}
			mail, err := db.buildDigitalLicenceMail(email, db.createWordPressInvite(email))
			if err != nil {
				return 0, err
			}
			mails = append(mails, mail)
			digitalMailAdded = true
			continue
		}

		if pdfItemsDone[item.ID] {
			continue
		}
		pdfDownload, err := db.renewOrderPDFDownload(orderID, item)
		if err != nil {
			return 0, err
		}
		mail, err := db.buildPDFLicenceMail(email, pdfDownload.LinkID)
		if err != nil {
			return 0, err
		}
		mails = append(mails, mail)
		pdfItemsDone[item.ID] = true
	}

	if len(mails) == 0 {
		return 0, ErrOrderHasNoDigitalItems
	}

	for _, mail := range mails {
		if err := sendMail(mail); err != nil {
			log.Error("ResendOrderMails: failed to send mail: ", orderID, err)
			return sent, err
		}
		sent++
	}
	log.Info("ResendOrderMails: resent ", sent, " mails for order ", orderID)
	return sent, nil
}

// createWordPressInvite returns a one-time WordPress login link for the
// customer, or an empty string if invites are not configured or fail.
func (db *Database) createWordPressInvite(email string) string {
	settings, err := db.GetSettings()
	if err != nil || settings == nil || settings.WordPressInviteURL == "" {
		return ""
	}
	inviteURL, err := wordpress.CreateInvite(
		settings.WordPressInviteURL,
		settings.WordPressInviteAPIKey,
		email,
		settings.WordPressInviteTTL,
	)
	if err != nil {
		log.Error("createWordPressInvite: ", err)
		return ""
	}
	return inviteURL
}

// renewOrderPDFDownload returns the order's download link for a PDF item with
// its validity restarted, creating the link if the order has none.
func (db *Database) renewOrderPDFDownload(orderID int, item Item) (pdfDownload PDFDownload, err error) {
	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		return pdfDownload, err
	}
	defer tx.Rollback()

	pdfDownload, err = db.GetPDFDownloadByOrderIdAndItemTx(tx, orderID, item.ID)
	if ent.IsNotFound(err) {
		if !item.PDF.Valid {
			return pdfDownload, errors.New("item has no pdf")
		}
		pdf, err := db.GetPDFByID(item.PDF.Int64)
		if err != nil {
			return pdfDownload, err
		}
		pdfDownload, err = db.CreatePDFDownload(tx, pdf, orderID, item.ID)
		if err != nil {
			return pdfDownload, err
		}
	} else if err != nil {
		return pdfDownload, err
	}

	pdfDownload.Timestamp = time.Now()
	pdfDownload.EmailSent = true
	if err = db.UpdatePdfDownloadTx(tx, pdfDownload); err != nil {
		return pdfDownload, err
	}
	return pdfDownload, tx.Commit()
}
