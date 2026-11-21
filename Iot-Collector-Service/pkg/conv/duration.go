package conv

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration 自定义包装类型，全局通用，支持 json "3s" / "500ms"
type Duration time.Duration

// String 可选，方便打印
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON 序列化 -> "3s"
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON 反序列化 "3s" -> Duration；null / 空串视为 0（由调用方决定是否回退默认）
func (d *Duration) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*d = 0
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("duration json解析失败: %w", err)
	}
	if raw == "" {
		*d = 0
		return nil
	}
	du, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("time.ParseDuration 失败: %w", err)
	}
	*d = Duration(du)
	return nil
}
