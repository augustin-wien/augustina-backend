package types

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

type Account struct {
	ID int `json:"id"`
	Name string `json:"name"`
}

type PaymentType struct {
	ID   int `json:"id"`
	Name string `json:"name"`
}

type Payment struct {
	ID	 int     	 `json:"id"`
	Timestamp string `json:"timestamp"`
	Sender int 		 `json:"sender"`
	Receiver int 	 `json:"receiver"`
	Type string 	 `json:"type"`
	Amount float64   `json:"amount"`
}
