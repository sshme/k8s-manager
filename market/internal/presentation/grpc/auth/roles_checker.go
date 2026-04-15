package auth

type Rule struct {
	Public bool
	Roles  []string
}

func hasRequiredRole(userRoles []string, required []string) bool {
	if len(required) == 0 {
		return true
	}

	set := make(map[string]struct{}, len(userRoles))
	for _, r := range userRoles {
		set[r] = struct{}{}
	}

	for _, r := range required {
		if _, ok := set[r]; ok {
			return true
		}
	}
	return false
}
