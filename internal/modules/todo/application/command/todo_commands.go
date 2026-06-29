package command

// CreateTodoCommand 是创建待办的命令。
type CreateTodoCommand struct {
	Title string
}

// CompleteTodoCommand 是完成待办的命令。
type CompleteTodoCommand struct {
	ID int64
}

// DeleteTodoCommand 是删除待办的命令。
type DeleteTodoCommand struct {
	ID int64
}
