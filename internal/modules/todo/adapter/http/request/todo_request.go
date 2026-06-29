package request

import todocmd "github.com/maguowei/gotobeta/internal/modules/todo/application/command"

// CreateTodoRequest 是创建待办的 HTTP 请求体。
type CreateTodoRequest struct {
	Title string `json:"title" binding:"required"`
}

// ToCommand 转换为应用层命令。
func (r CreateTodoRequest) ToCommand() todocmd.CreateTodoCommand {
	return todocmd.CreateTodoCommand{Title: r.Title}
}
