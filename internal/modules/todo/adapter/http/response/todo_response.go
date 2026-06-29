package response

import (
	"time"

	todoresult "github.com/maguowei/gotobeta/internal/modules/todo/application/result"
)

// TodoResponse 是待办的 HTTP 响应体。
type TodoResponse struct {
	TodoID    int64  `json:"todoId,string"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ToTodoResponse 将应用层结果转换为 HTTP 响应。
func ToTodoResponse(out *todoresult.TodoResult) TodoResponse {
	return TodoResponse{
		TodoID:    out.ID,
		Title:     out.Title,
		Status:    out.Status,
		CreatedAt: out.CreatedAt.Format(time.DateTime),
		UpdatedAt: out.UpdatedAt.Format(time.DateTime),
	}
}

// ToTodoListResponse 批量转换。
func ToTodoListResponse(items []*todoresult.TodoResult) []TodoResponse {
	result := make([]TodoResponse, 0, len(items))
	for _, item := range items {
		result = append(result, ToTodoResponse(item))
	}
	return result
}
