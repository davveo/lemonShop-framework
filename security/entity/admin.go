package entity

type Admin struct {
	Uid      int64  `json:"uid"`
	Uuid     string `json:"uuid"`
	Founder  int    `json:"founder"`
	UserName string `json:"username"`
	Role     string `json:"role"`
}
