package database

// GetTotal returns the total amount of a payment order in cents
func (order PaymentOrder) GetTotal() (amount int) {
	amount = 0
	for _, item := range order.OrderItems {
		amount += item.Price * item.Quantity
	}
	return amount
}
