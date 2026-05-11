package auth

const (
	RoleMarketAdmin     = "market_admin"
	RoleMarketPublisher = "market_publisher"
)

type Rule struct {
	Public        bool
	RequiredRoles []string
}
