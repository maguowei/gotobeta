package sensitive

import (
	"strings"
	"unicode/utf8"
)

// RedactPhone 中间 4 位脱敏（11 位手机号）。
func RedactPhone(s string) string {
	if utf8.RuneCountInString(s) < 7 {
		return strings.Repeat("*", utf8.RuneCountInString(s))
	}
	r := []rune(s)
	return string(r[:3]) + "****" + string(r[len(r)-4:])
}

// RedactEmail 保留首字符与 @ 后域名。
func RedactEmail(s string) string {
	at := strings.Index(s, "@")
	if at <= 0 {
		return strings.Repeat("*", utf8.RuneCountInString(s))
	}
	return s[:1] + "***" + s[at:]
}

// RedactIDCard 中间 10 位脱敏（18 位身份证）。
func RedactIDCard(s string) string {
	if utf8.RuneCountInString(s) < 8 {
		return strings.Repeat("*", utf8.RuneCountInString(s))
	}
	r := []rune(s)
	return string(r[:4]) + "**********" + string(r[len(r)-4:])
}

// RedactToken 前 4 后 4 保留，中间全部 *。
func RedactToken(s string) string {
	n := utf8.RuneCountInString(s)
	if n <= 8 {
		return strings.Repeat("*", n)
	}
	r := []rune(s)
	return string(r[:4]) + "****" + string(r[len(r)-4:])
}

// RedactBearer 处理 "Bearer xxx" 形式 Authorization 头。
func RedactBearer(h string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return RedactToken(h)
	}
	return prefix + RedactToken(strings.TrimPrefix(h, prefix))
}
