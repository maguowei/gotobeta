package query

// GetTodoQuery 是查询单个待办的查询。
type GetTodoQuery struct {
	ID int64
}

// ListTodosQuery 是列出待办的查询，预留分页、过滤等字段位。
type ListTodosQuery struct{}
