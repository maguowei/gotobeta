package rbac

import "testing"

func TestPolicyAllowsRoleDirectAndWildcardPermissions(t *testing.T) {
	policy := Policy{
		"admin":  []Permission{Permission("*")},
		"editor": []Permission{Permission("articles:write")},
	}

	testCases := []struct {
		name        string
		roles       []string
		permissions []string
		required    Permission
		want        bool
	}{
		{name: "role permission", roles: []string{"editor"}, required: "articles:write", want: true},
		{name: "direct permission", permissions: []string{"users:read"}, required: "users:read", want: true},
		{name: "wildcard role", roles: []string{"admin"}, required: "billing:refund", want: true},
		{name: "missing", roles: []string{"viewer"}, required: "articles:write", want: false},
		{name: "empty required", roles: []string{"admin"}, required: "", want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := policy.Allows(tc.roles, tc.permissions, tc.required)
			if got != tc.want {
				t.Fatalf("Allows() = %v, want %v", got, tc.want)
			}
		})
	}
}
