// handlerx 包提供网关层参数解析与校验辅助能力。
package handlerx

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSONInt64 支持从 JSON number 或 string 解码为 int64。
// 主要用于网关层兼容前端以字符串传递大整数 ID 的场景。
type JSONInt64 int64

// Int64 返回底层 int64 值。
func (v JSONInt64) Int64() int64 {
	return int64(v)
}

// UnmarshalJSON 兼容 number/string 两种输入格式。
func (v *JSONInt64) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("JSONInt64: nil receiver")
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		return fmt.Errorf("invalid int64 value")
	}
	if raw[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return fmt.Errorf("invalid int64 string: %w", err)
		}
		parsed, err := parseStrictInt64(strings.TrimSpace(text))
		if err != nil {
			return err
		}
		*v = JSONInt64(parsed)
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("invalid int64 number: %w", err)
	}
	parsed, err := parseStrictInt64(num.String())
	if err != nil {
		return err
	}
	*v = JSONInt64(parsed)
	return nil
}

func parseStrictInt64(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("invalid int64 value")
	}
	if strings.ContainsAny(raw, ".eE") {
		return 0, fmt.Errorf("int64 value must be an integer")
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("int64 parse failed: %w", err)
	}
	return parsed, nil
}
