package internalmqtt

import (
	"encoding/json"
	"time"
)

// ======== 下行（订阅）入参 DTO ========

// writeItem 写数据单条：数组 json，以全局唯一的 tag 为键，带类型、值，可选时间戳。
// 服务内部根据 tag 反查点位 id 与所属驱动；value 保留原始 JSON，交由 reflect 按 type 精确转换。
type writeItem struct {
	Tag   string          `json:"tag"`
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
	Time  string          `json:"time,omitempty"` // 可选，RFC3339；缺省用当前时间
}

// restartReq 重启请求：target 区分 driver / service；driver 可选指定 drive_ids，为空则重启全部驱动。
type restartReq struct {
	Target   string `json:"target"`    // driver | service
	DriveIds []uint `json:"drive_ids"` // 仅 target=driver 时生效；空=全部
}

// tokenReq 增加临时 handler token：为给定 access_key 生成短(access)/长(refresh)令牌，直写 Redis。
// TTL 使用 conv.Duration 语义（"15m0s"/"168h"），留空则回退默认值。
// WithAccessToken / WithRefreshToken 控制各签发哪种令牌：两者均为 nil（缺省）时都签发（向后兼容）；
// 只想签发长令牌则置 with_access_token=false。
type tokenReq struct {
	AccessKey       string `json:"access_key"`
	AccessTokenTTL  string `json:"access_token_ttl"`  // 短令牌 TTL
	RefreshTokenTTL string `json:"refresh_token_ttl"` // 长令牌 TTL

	WithAccessToken  *bool `json:"with_access_token"`  // 是否签发短令牌（nil=签发）
	WithRefreshToken *bool `json:"with_refresh_token"` // 是否签发长令牌（nil=签发）
}

// configUpdateReq 更新所有配置：json 仅携带 http 下载地址，服务下载表格(xlsx)后全量导入。
// url / download_url 二者取其一（优先 url）。
type configUpdateReq struct {
	Url         string `json:"url"`
	DownloadUrl string `json:"download_url"`
}

// ======== 上行（发布）数据 DTO ========

// dataItem 点位实时值对外格式：以全局唯一的 tag 为键，附带可读的类型名与 RFC3339 时间。
type dataItem struct {
	Tag   string    `json:"tag"`
	Value any       `json:"value"`
	Type  string    `json:"type"`
	Msg   string    `json:"msg,omitempty"`
	Time  time.Time `json:"time"`
}

// alarmItem 报警事件对外格式：包含报警配置全部字段（id/tag/config/group/status）+ 报警值和报警时间。
type alarmItem struct {
	Tag     string    `json:"tag"`      // 报警点位标（全局唯一）
	AlarmId uint      `json:"alarm_id"` // 报警配置 ID
	Config  string    `json:"config"`   // 报警表达式（如 "x > 100"）
	Group   int       `json:"group"`    // 报警组号
	Status  string    `json:"status"`   // 报警/恢复/错误信息
	Value   any       `json:"value"`    // 触发时的点位值
	Type    string    `json:"type"`     // 点位值类型名
	Time    time.Time `json:"time"`     // 报警触发时间
}

// resultPayload 操作结果回执（重启 / 令牌 / 配置更新等异步动作的反馈）。
type resultPayload struct {
	Action string `json:"action"`
	Ok     bool   `json:"ok"`
	Msg    string `json:"msg"`
	Data   any    `json:"data,omitempty"`
}

// publishResult 向 resultSuffix(action) 发布一条结果回执，序列化失败仅记日志。
func publishResult(action string, ok bool, msg string, data any) {
	payload := resultPayload{Action: action, Ok: ok, Msg: msg, Data: data}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = publish(resultSuffix(action), b)
}
