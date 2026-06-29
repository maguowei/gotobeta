package timeutil

import "time"

// FormatTime 格式化时间为 YYYY-MM-DD HH:mm:ss。
func FormatTime(t time.Time) string {
	return t.Format(time.DateTime)
}
