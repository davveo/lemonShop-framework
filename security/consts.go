package security

type Role string
type Permission string

const (
	DefaultDisabled = 0 // 是否被删除0 删除   1  没有删除
	DefaultEnabled  = 1 // 是否被删除0 删除   1  没有删除
	DefaultSellerId = 0 // 默认卖家
)

const (
	Buyer  = Role("Buyer")
	Seller = Role("Seller")
	Clerk  = Role("Clerk")
)

const (
	BuyerPermission  = Permission("Buyer")
	SellerPermission = Permission("Seller")
	AdminPermission  = Permission("Admin")
	ClientPermission = Permission("Client")

	// jwt相关配置
	SECRET        = "ThisIsASecret"
	TOKEN_PREFIX  = "Bearer"
	HEADER_STRING = "Authorization"
	INVALID_TIME  = 60
)
