package entity

import "github.com/davveo/lemonShop-framework/security"

type Buyer struct {
	User
}

func NewBuyer() Buyer {
	buyer := Buyer{}
	buyer.Add(security.Buyer)
	return buyer
}
