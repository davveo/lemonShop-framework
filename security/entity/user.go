package entity

import "github.com/davveo/lemonShop-framework/security"

type User struct {
	Uid      int64           `json:"uid"`
	Uuid     string          `json:"uuid"`
	UserName string          `json:"username"`
	Roles    []security.Role `json:"roles"`
}

func (u *User) Add(roles ...security.Role) {
	for _, role := range roles {
		u.Roles = append(u.Roles, role)
	}
}
