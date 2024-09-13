package database

import "gopkg.in/guregu/null.v4"

// GetTotal returns the total amount of a payment order in cents
// Only entries that are part of the sale are counted
func (order Order) GetTotal() (amount int) {
	amount = 0
	for _, entry := range order.Entries {
		if entry.IsSale {
			amount += entry.Price * entry.Quantity
		}
	}
	return amount
}

type PDFDownloadLinks struct {
	Link   string
	ItemID null.Int
}

func (order Order) GetPDFDownloadLinks() *[]PDFDownloadLinks {
	log.Info("Getting PDF download links for order ", order.ID)
	links := []PDFDownloadLinks{}
	PDFDownloads, err := Db.GetPDFDownloadByOrderId(order.ID)
	if err != nil {
		log.Error("Error getting PDF download links ", err)
		return nil
	}
	for _, download := range PDFDownloads {
		links = append(links, PDFDownloadLinks{
			Link:   download.LinkID,
			ItemID: download.ItemID,
		})
	}
	if len(links) > 0 {
		return &links
	}
	return nil
}

func (order Order) HasDigitalItem() bool {
	for _, entry := range order.Entries {
		if entry.IsDigital() {
			return true
		}
	}
	return false
}

func (entry OrderEntry) IsDigital() bool {
	item, err := Db.GetItem(entry.Item)
	if err != nil {
		log.Error("Error getting item ", err)
		return false
	}
	if item.IsPDFItem {
		return true
	}
	return false
}
