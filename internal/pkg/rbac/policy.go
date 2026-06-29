package rbac

import "strings"

// Permission 表示一个权限点，建议使用 resource:action 形式。
type Permission string

// Role 表示角色名。
type Role string

// Policy 定义角色到权限的映射。
type Policy map[Role][]Permission

const wildcard Permission = "*"

// Allows 判断一组角色和直接权限是否满足 required。
func (p Policy) Allows(roles []string, permissions []string, required Permission) bool {
	required = normalizePermission(required)
	if required == "" {
		return false
	}
	if permissionListAllows(toPermissions(permissions), required) {
		return true
	}
	for _, roleName := range roles {
		role := Role(strings.TrimSpace(roleName))
		if role == "" {
			continue
		}
		if permissionListAllows(p[role], required) {
			return true
		}
	}
	return false
}

func permissionListAllows(items []Permission, required Permission) bool {
	for _, item := range items {
		item = normalizePermission(item)
		if item == wildcard || item == required {
			return true
		}
	}
	return false
}

func toPermissions(values []string) []Permission {
	items := make([]Permission, 0, len(values))
	for _, value := range values {
		items = append(items, Permission(value))
	}
	return items
}

func normalizePermission(value Permission) Permission {
	return Permission(strings.TrimSpace(string(value)))
}
