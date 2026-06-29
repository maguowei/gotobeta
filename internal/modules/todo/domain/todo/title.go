package todo

import (
	"strings"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// Title 是待办标题值对象：不可变（私有字段、无 setter）、自带构造校验与相等性。
//
// 把"什么是合法标题"的规则收敛进值对象，聚合根与调用方只能通过 NewTitle 构造，
// 无法绕过校验得到非法标题。
type Title struct {
	value string
}

// NewTitle 构造并校验标题。
func NewTitle(raw string) (Title, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return Title{}, apperr.InvalidParam("title 不能为空")
	}
	if len([]rune(value)) > 200 {
		return Title{}, apperr.InvalidParam("title 长度不能超过 200")
	}
	return Title{value: value}, nil
}

// String 返回标题的字符串表示。
func (t Title) String() string {
	return t.value
}

// Equals 比较两个标题是否相等。
func (t Title) Equals(other Title) bool {
	return t.value == other.value
}
