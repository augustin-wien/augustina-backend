package database

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
