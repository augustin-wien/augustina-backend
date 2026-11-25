package database

import (
	"context"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// createDevCustomersAndAbonements creates customers in every abonement state
func (db *Database) createDevCustomersAndAbonements() error {
	// Abonement items may start disabled, so look them up including disabled ones
	abonementItemID, err := db.devItemIDByType("abonement")
	if err != nil {
		return err
	}

	customers := []Customer{
		{
			KeycloakID:    "dev-customer-keycloak-1",
			Email:         "anna.mueller@example.com",
			FirstName:     "Anna",
			LastName:      "Müller",
			LicenseGroups: []string{"digital_edition"},
		},
		{
			KeycloakID: "dev-customer-keycloak-2",
			Email:      "max.mustermann@example.com",
			FirstName:  "Max",
			LastName:   "Mustermann",
		},
		{
			KeycloakID: "dev-customer-keycloak-3",
			Email:      "eva.schmidt@example.com",
			FirstName:  "Eva",
			LastName:   "Schmidt",
		},
		{
			KeycloakID:    "dev-customer-keycloak-4",
			Email:         "jonas.berger@example.com",
			FirstName:     "Jonas",
			LastName:      "Berger",
			LicenseGroups: []string{"digital_edition", "testedition"},
		},
		{
			KeycloakID: "dev-customer-keycloak-5",
			Email:      "lena.gruber@example.com",
			FirstName:  "Lena",
			LastName:   "Gruber",
		},
		{
			KeycloakID: "dev-customer-keycloak-6",
			Email:      "paul.wagner@example.com",
			FirstName:  "Paul",
			LastName:   "Wagner",
		},
		// Registered without ever buying anything, name unknown
		{
			KeycloakID: "dev-customer-keycloak-7",
			Email:      "nur.email@example.com",
		},
	}

	createdIDs := make([]int, 0, len(customers))
	for _, c := range customers {
		created, err := db.CreateCustomer(&c)
		if err != nil {
			log.Error("createDevCustomersAndAbonements: customer creation failed ", zap.Error(err))
			return err
		}
		createdIDs = append(createdIDs, created.ID)
	}

	if abonementItemID == 0 {
		return nil
	}

	now := time.Now()
	abonements := []Abonement{
		// Running
		{CustomerID: createdIDs[0], FromDate: now.AddDate(-1, 0, 0), ToDate: now.AddDate(0, 6, 0), Status: "active"},
		{CustomerID: createdIDs[1], FromDate: now.AddDate(0, -3, 0), ToDate: now.AddDate(0, 9, 0), Status: "active"},
		// Still marked active, but ran out
		{CustomerID: createdIDs[2], FromDate: now.AddDate(-2, 0, 0), ToDate: now.AddDate(-1, 0, 0), Status: "active"},
		// Renewed: the old one ran out, the new one runs
		{CustomerID: createdIDs[3], FromDate: now.AddDate(-2, 0, 0), ToDate: now.AddDate(-1, 0, 0), Status: "inactive"},
		{CustomerID: createdIDs[3], FromDate: now.AddDate(-1, 0, 0), ToDate: now.AddDate(0, 0, 10), Status: "active"},
		// Starts in the future
		{CustomerID: createdIDs[4], FromDate: now.AddDate(0, 0, 14), ToDate: now.AddDate(1, 0, 14), Status: "active"},
		// Cancelled before running out
		{CustomerID: createdIDs[5], FromDate: now.AddDate(0, -6, 0), ToDate: now.AddDate(0, 6, 0), Status: "cancelled"},
	}

	for _, a := range abonements {
		a.ItemID = abonementItemID
		_, err := db.CreateAbonement(&a)
		if err != nil {
			log.Error("createDevCustomersAndAbonements: abonement creation failed ", zap.Error(err))
			return err
		}
	}

	return nil
}

// createDevCampaigns creates a campaign for every schedule state
func (db *Database) createDevCampaigns(items devItems) error {
	now := time.Now()
	at := func(days int) *time.Time {
		t := now.AddDate(0, 0, days)
		return &t
	}
	campaigns := []struct {
		campaign      ent.Campaign
		views, clicks int
	}{
		{ent.Campaign{Name: "Kalender-Aktion", ItemID: items.calendar, Title: "Der neue Kalender ist da!", Text: "Jetzt den Augustin-Kalender sichern und Verkäufer:innen unterstützen.", StartsAt: at(-10), EndsAt: at(20), Enabled: true}, 340, 27},
		{ent.Campaign{Name: "Dauerwerbung Digitale Zeitung", ItemID: items.digitalNewspaper, Title: "Lies uns auch digital", Text: "Die Zeitung als digitale Ausgabe, direkt nach dem Kauf.", Enabled: true}, 1250, 88},
		{ent.Campaign{Name: "Weihnachtsaktion", ItemID: items.bag, Title: "Das Geschenk mit Sinn", Text: "Die Stofftasche als Weihnachtsgeschenk.", StartsAt: at(30), EndsAt: at(60), Enabled: true}, 0, 0},
		{ent.Campaign{Name: "Sommeraktion (vorbei)", ItemID: items.newspaper, Title: "Sommerausgabe", Text: "Die Doppelnummer für den Sommer.", StartsAt: at(-90), EndsAt: at(-30), Enabled: true}, 2100, 64},
		{ent.Campaign{Name: "Entwurf Spendenaufruf", ItemID: items.donation, Title: "Spenden hilft", Text: "Noch nicht freigegeben.", Enabled: false}, 0, 0},
	}

	for _, c := range campaigns {
		if c.campaign.ItemID == 0 {
			continue
		}
		created, err := db.CreateCampaign(&c.campaign)
		if err != nil {
			return err
		}
		err = db.EntClient.Campaign.UpdateOneID(created.ID).
			SetViews(c.views).
			SetClicks(c.clicks).
			Exec(context.Background())
		if err != nil {
			return err
		}
	}
	return nil
}

// createDevPDFDownloads adds a used download link next to the unused one the PDF
// sale created
func (db *Database) createDevPDFDownloads(items devItems) error {
	item, err := db.GetItem(items.pdfNewspaper)
	if err != nil {
		return err
	}
	download, err := db.CreatePDFDownloadForItem(int(item.PDF.Int64), item.ID)
	if err != nil {
		return err
	}
	download.DownloadCount = 3
	download.LastDownload = time.Now().Add(-5 * time.Hour)
	download.ItemID = null.IntFrom(int64(item.ID))
	return db.UpdatePdfDownload(download)
}

// createDevBlockedIPs creates an active block, an expired block and an IP that
// only collected strikes
func (db *Database) createDevBlockedIPs() error {
	ctx := context.Background()
	now := time.Now()
	err := db.EntClient.BlockedIP.Create().
		SetIP("203.0.113.10").
		SetStrikes(0).
		SetBlockExpiresAt(now.Add(12 * time.Hour)).
		SetReason("too many invalid vendor IDs").
		Exec(ctx)
	if err != nil {
		return err
	}
	err = db.EntClient.BlockedIP.Create().
		SetIP("203.0.113.20").
		SetStrikes(0).
		SetBlockExpiresAt(now.AddDate(0, 0, -2)).
		SetReason("too many invalid vendor IDs").
		Exec(ctx)
	if err != nil {
		return err
	}
	return db.EntClient.BlockedIP.Create().
		SetIP("198.51.100.7").
		SetStrikes(2).
		Exec(ctx)
}
