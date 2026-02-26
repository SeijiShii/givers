package model

// CostItem represents one line in a project's cost estimate.
type CostItem struct {
	Label     string `json:"label"`
	UnitPrice int    `json:"unit_price"`
	Quantity  int    `json:"quantity"`
}

// Monthly returns the monthly cost for this line item.
func (c *CostItem) Monthly() int {
	return c.UnitPrice * c.Quantity
}

// TotalMonthly sums the monthly costs of all items.
func TotalMonthly(items []CostItem) int {
	total := 0
	for i := range items {
		total += items[i].Monthly()
	}
	return total
}
