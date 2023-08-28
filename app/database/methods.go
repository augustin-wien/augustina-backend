package database

// GetTotal returns the total amount of a payment order in cents
func (order Order) GetTotal() (amount int) {
	amount = 0
	for _, item := range order.Entries {
		amount += item.Price * item.Quantity
	}
	return amount
}
