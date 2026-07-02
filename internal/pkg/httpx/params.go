// Package httpx 提供 HTTP 协议适配边界的共享工具。
package httpx

import "strconv"

// ParsePositiveID 解析路径/查询参数中的正整数 ID；非数字或非正数返回错误。
func ParsePositiveID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
