package database

// GetTotal returns the total amount of a payment order
func (order PaymentOrder)GetTotal() (amount float32) {
	amount = 0.0
	for _, item := range order.OrderItems {
		amount += item.Price * item.Quantity
	}
	return amount
}
