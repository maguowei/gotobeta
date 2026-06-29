package sensitive

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"strings"
)

var sensitiveKeys = map[string]bool{
	"password":      true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"api_key":       true,
	"apikey":        true,
	"authorization": true,
	"secret":        true,
	"secret_key":    true,
	"secretkey":     true,
	"dsn":           true,
}

// MaskByContentType 按内容类型脱敏。
func MaskByContentType(contentType string, data []byte) string {
	if len(data) == 0 {
		return ""
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(contentType))
	}

	switch mediaType {
	case "application/x-www-form-urlencoded":
		return maskForm(data)
	default:
		return maskJSON(data)
	}
}

func maskJSON(data []byte) string {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Sprintf("[invalid json body, size=%d]", len(data))
	}

	masked, err := json.Marshal(maskValue(value))
	if err != nil {
		return fmt.Sprintf("[unserializable json body, size=%d]", len(data))
	}

	return string(masked)
}

func maskForm(data []byte) string {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return fmt.Sprintf("[invalid form body, size=%d]", len(data))
	}

	for key := range values {
		if sensitiveKeys[strings.ToLower(key)] {
			values.Set(key, "***")
		}
	}

	return values.Encode()
}

func maskValue(value any) any {
	switch current := value.(type) {
	case map[string]any:
		masked := make(map[string]any, len(current))
		for key, child := range current {
			if sensitiveKeys[strings.ToLower(key)] {
				masked[key] = "***"
			} else {
				masked[key] = maskValue(child)
			}
		}

		return masked
	case []any:
		masked := make([]any, len(current))
		for index, child := range current {
			masked[index] = maskValue(child)
		}

		return masked
	default:
		return value
	}
}
