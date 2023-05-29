package handlers

type Setting struct {
	Color string  `json:"color"`
	Logo  string  `json:"logo"`
	Price float64 `json:"price"`
}

type Vendor struct {
	Credit   float64 `json:"credit"`
	QRcode   string  `json:"qrcode"`
	IDnumber string  `json:"id-number"`
}
