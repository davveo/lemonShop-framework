package entity

import "github.com/davveo/lemonShop-framework/security"

type Clerk struct {
	Seller
	ClerkId   int64  // 店员id
	ClerkName string // 店员名称
	Founder   int    // 是否是超级店员
	Role      string // 权限
}

func NewClerk() Clerk {
	clerk := Clerk{}
	clerk.Add(security.Clerk)
	return clerk
}
