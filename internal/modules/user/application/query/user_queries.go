// Package query 定义用户认证只读用例入参。
package query

// GetCurrentUserQuery 是查询当前用户入参。
type GetCurrentUserQuery struct {
	UserID int64
}

// ListIdentitiesQuery 是查询第三方身份列表入参。
type ListIdentitiesQuery struct {
	UserID int64
}
