/*
* 日期: 2026.11.21
* 作者: 范范zwf
* 作用: 内部 MQTT 下行指令的操作日志埋点。
*       MQTT 通路没有 HTTP 那套凭证（Broker 账号只是连接凭证，不代表操作人），
*       因此 Operator 固定记通道名 internalmqtt，把「哪条主题、动了什么、结果如何」
*       全部写进 Content.detail，回执发布成功与否都在日志里可对账。
*       写日志一律异步、失败只打印，绝不影响指令处理与回执。
 */
package internalmqtt

import (
	"encoding/json"
	"fmt"

	"collector/internal/handler"
	"collector/internal/repo/mysql"
)

// Mqtt_Raw_Max 失败兜底日志里原始报文的最大长度，避免整包配置等超大 payload 撑爆日志
const Mqtt_Raw_Max = 1024

// logMqtt 记一条 MQTT 通路的操作日志，detail 为业务详情（禁止放令牌、密码等敏感值）
func logMqtt(topic string, typ mysql.Log_Type, operate mysql.Log_Operate, detail map[string]any) {
	handler.LogExternalEvent(mysql.Log_Source_Mqtt, topic, typ, operate, detail)
}

// logMqttFail 记一条失败日志：带上失败原因与原始报文（截断），便于事后复盘是谁发了什么
func logMqttFail(topic string, typ mysql.Log_Type, operate mysql.Log_Operate, reason string, raw []byte, extra map[string]any) {
	detail := map[string]any{
		"ok":     false,
		"reason": reason,
		"raw":    clipRaw(raw),
	}
	for k, v := range extra {
		detail[k] = v
	}
	logMqtt(topic, typ, operate, detail)
}

// clipRaw 按字符截断原始报文，非法 UTF-8 也照常入库（只是可读性下降）
func clipRaw(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	r := []rune(string(raw))
	if len(r) <= Mqtt_Raw_Max {
		return string(raw)
	}
	return string(r[:Mqtt_Raw_Max]) + "…"
}

// configTypeBySheet 配置表 Sheet 名 → 日志 Type；未知返回 Log_Type_All（整包动作）
func configTypeBySheet(sheet string) mysql.Log_Type {
	switch sheet {
	case "drive_config":
		return mysql.Log_Type_Drive
	case "point_config":
		return mysql.Log_Type_Point
	case "alarm_config":
		return mysql.Log_Type_Alarm
	case "history_config":
		return mysql.Log_Type_History
	default:
		return mysql.Log_Type_All
	}
}

// logMqttWrites 写值日志：一个点位一行，值保留请求里的原始 JSON 报文
func logMqttWrites(topic string, items []handler.Log_Write_item) {
	handler.LogMqttWriteValues(topic, items)
}

// rawValue 把点位值转成入库文本（写入时用的是请求原始 JSON，这里仅做兜底序列化）
func rawValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
