package entity

import "github.com/davveo/lemonShop-framework/security"

type Seller struct {
	User

	SellerId     int    `json:"sellerId"`
	SellerName   string `json:"sellerName"`
	SelfOperated int    `json:"selfOperated"`
}

func NewSeller() Seller {
	seller := Seller{}
	seller.Add(security.Seller)
	return seller
}
