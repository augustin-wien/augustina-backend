package handlers

type setting struct {
	Color string  `json:"color"`
	Logo  string  `json:"logo"`
	Price float64 `json:"price"`
}

type vendor struct {
	Credit   float64 `json:"credit"`
	QRcode   string  `json:"qrcode"`
	IDnumber string  `json:"id-number"`
}
