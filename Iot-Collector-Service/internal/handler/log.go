/*
* 日期: 2026.11.11
* 作者: 范范zwf
* 作用: 操作日志埋点与查询（单表 log_operate）。
*       操作者身份由 Auth_Middleware 解析凭证时写入上下文，Operator 列固定形态：
*         user:<id>       网页登录用户（user_account.Id）
*         aksk:<id>       AK/SK 接口账号（api_key.Id）
*         internalmqtt    内部 MQTT 下行指令（无账号凭证，细节见 Content）
*       Content 一律存 JSON：通路 / 请求方法 / 路径 / UA / IP / 业务详情，越全越好。
*       所有写入异步执行且失败只打印，绝不阻塞或改变业务响应。
 */
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"collector/internal/repo/mysql"

	"github.com/gin-gonic/gin"
)

// ======== 上下文键（由鉴权中间件写入） ========
const (
	CtxKeyLogSource  = "Log_Source"     // mysql.Log_Source：aksk / user
	CtxKeyLogAccount = "Log_Account_Id" // uint：api_key.Id 或 user_account.Id
)

// AkSk_Login_Log_Window 同一 AK/SK 记一条"登陆"日志的节流窗口（AK/SK 无显式登录动作，按首次使用计）
var AkSk_Login_Log_Window = 10 * time.Minute

// 入库字段长度上限，与表结构保持一致（Content=text，Operator/Ip=varchar）
const (
	Log_Content_Max  = 8000 // 详情 JSON 截断长度，避免单条日志过大
	Log_Ip_Max       = 45   // 对齐 log_operate.Ip varchar(45)
	Log_Operator_Max = 64   // 对齐 log_operate.Operator varchar(64)
	Log_Keyword_Max  = 128  // 查询关键字长度上限
)

// 通路标识（写入 Content.channel）
const (
	Log_Channel_Http = "http"
	Log_Channel_Mqtt = "internalmqtt"
)

// logActor 一次操作的操作人身份
type logActor struct {
	src       mysql.Log_Source
	accountId uint
	accessKey string
	operator  string // Operator 列：user:1 / aksk:1 / internalmqtt
	ip        string
}

// newActor 组装操作人身份（HTTP 与 MQTT 通路共用）
func newActor(src mysql.Log_Source, accountId uint, accessKey, rawIp string) logActor {
	return logActor{
		src:       src,
		accountId: accountId,
		accessKey: accessKey,
		operator:  clip(src.Operator(accountId), Log_Operator_Max),
		ip:        clip(normalizeIP(rawIp), Log_Ip_Max),
	}
}

// setLogActor 鉴权通过后写入操作者身份，供日志埋点与 RBAC 共用
func setLogActor(ctx *gin.Context, src mysql.Log_Source, accountId uint) {
	ctx.Set(CtxKeyLogSource, src)
	ctx.Set(CtxKeyLogAccount, accountId)
}

// currentActor 读取当前请求的操作者身份；ok=false 表示该路由未鉴权或拿不到账号 id（无法归属日志）
func currentActor(ctx *gin.Context) (logActor, bool) {
	srcVal, hasSrc := ctx.Get(CtxKeyLogSource)
	idVal, hasId := ctx.Get(CtxKeyLogAccount)
	if !hasSrc || !hasId {
		return logActor{}, false
	}
	src, ok1 := srcVal.(mysql.Log_Source)
	id, ok2 := idVal.(uint)
	if !ok1 || !ok2 || id == 0 {
		return logActor{}, false
	}
	return newActor(src, id, ctx.GetString(CtxKeyAccessKey), ctx.ClientIP()), true
}

// clip 按字符数截断（按 rune 切，避免切断多字节字符）
func clip(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// marshalLogContent 把详情压成紧凑 JSON 入库；序列化失败时退化为文本形式
func marshalLogContent(content any) string {
	if content == nil {
		return ""
	}
	switch v := content.(type) {
	case string:
		return clip(v, Log_Content_Max)
	}
	b, err := json.Marshal(content)
	if err != nil {
		return clip(fmt.Sprintf("%v", content), Log_Content_Max)
	}
	return clip(string(b), Log_Content_Max)
}

// logEvent 一次待落库的操作日志（HTTP 与 MQTT 通路共用同一套组装逻辑）
type logEvent struct {
	actor   logActor
	channel string            // http / internalmqtt
	method  string            // HTTP 方法
	path    string            // HTTP 路径
	query   string            // HTTP 查询串
	agent   string            // User-Agent
	topic   string            // MQTT 主题
	typ     mysql.Log_Type    // 操作对象
	operate mysql.Log_Operate // 操作方式
	detail  any               // 业务详情（已剔除敏感值）
}

// buildContent 汇总通路信息 + 业务详情为 Content
func (e logEvent) buildContent() string {
	m := map[string]any{"channel": e.channel, "operator": e.actor.operator}
	if e.actor.accessKey != "" {
		m["access_key"] = e.actor.accessKey
	}
	if e.actor.ip != "" {
		m["ip"] = e.actor.ip
	}
	if e.method != "" {
		m["method"] = e.method
		m["path"] = e.path
	}
	if e.query != "" {
		m["query"] = e.query
	}
	if e.agent != "" {
		m["user_agent"] = e.agent
	}
	if e.topic != "" {
		m["topic"] = e.topic
	}
	if e.detail != nil {
		m["detail"] = e.detail
	}
	return marshalLogContent(m)
}

// emit 异步落库；拿不到操作人身份直接跳过（只留 WARN，不影响业务）
func (e logEvent) emit() {
	if e.actor.operator == "" {
		log.Printf("WARN 跳过操作日志：无操作人身份 type=%s op=%s path=%s", e.typ, e.operate, e.path)
		return
	}
	row := mysql.Log_Row{
		Type:        e.typ,
		Operate:     e.operate,
		Operator:    e.actor.operator,
		Content:     e.buildContent(),
		Ip:          e.actor.ip,
		Create_Time: time.Now(),
	}
	go func() {
		if err := mysql.Log__Add(row); err != nil {
			log.Printf("ERROR 写操作日志失败 operator=%s type=%s op=%s: %v", row.Operator, row.Type, row.Operate, err)
		}
	}()
}

// httpEvent 从 gin 上下文取齐通路字段，业务代码只需给类型与详情
func httpEvent(ctx *gin.Context, typ mysql.Log_Type, operate mysql.Log_Operate, detail any) (logEvent, bool) {
	actor, ok := currentActor(ctx)
	if !ok {
		return logEvent{}, false
	}
	return logEvent{
		actor:   actor,
		channel: Log_Channel_Http,
		method:  ctx.Request.Method,
		path:    ctx.Request.URL.Path,
		query:   ctx.Request.URL.RawQuery,
		agent:   ctx.Request.UserAgent(),
		typ:     typ,
		operate: operate,
		detail:  detail,
	}, true
}

// logOperate 记录一条 HTTP 通路的操作日志。
// typ 取 mysql.Log_Type_*，operate 取 mysql.Log_Operate_*；
// detail 由调用方组装，必须已剔除密码 / secret_key 等敏感值。
func logOperate(ctx *gin.Context, typ mysql.Log_Type, operate mysql.Log_Operate, detail any) {
	if e, ok := httpEvent(ctx, typ, operate, detail); ok {
		e.emit()
	}
}

// LogExternalEvent 非 HTTP 通路（内部 MQTT 等）记录操作日志的统一入口。
// src 决定 Operator 形态（mysql.Log_Source_Mqtt → internalmqtt），topic 为触发本次动作的主题。
func LogExternalEvent(src mysql.Log_Source, topic string, typ mysql.Log_Type, operate mysql.Log_Operate, detail any) {
	logEvent{
		actor:   newActor(src, 0, "", ""),
		channel: string(src),
		topic:   topic,
		typ:     typ,
		operate: operate,
		detail:  detail,
	}.emit()
}

// Log_Write_item 一次写值动作里的单个点位（写值按点位逐条落库，便于按点位追溯）
type Log_Write_item struct {
	PointId uint   `json:"point_id"`
	Tag     string `json:"tag,omitempty"`
	Type    string `json:"type,omitempty"`
	Value   string `json:"value"`
	Time    string `json:"time,omitempty"`
}

// logWriteValues 记录一次下发涉及的写值日志（每个点位一条）
func logWriteValues(ctx *gin.Context, items []Log_Write_item) {
	if len(items) == 0 {
		return
	}
	e, ok := httpEvent(ctx, mysql.Log_Type_Point, mysql.Log_Operate_Write, nil)
	if !ok {
		log.Printf("WARN 跳过写值日志：请求无账号身份 path=%s", ctx.Request.URL.Path)
		return
	}
	e.emitWrites(items)
}

// emitWrites 写值日志专用：一个点位一行，共用同一份通路字段
func (e logEvent) emitWrites(items []Log_Write_item) {
	rows := make([]mysql.Log_Row, 0, len(items))
	for _, it := range items {
		item := it
		item.Value = clip(item.Value, Log_Content_Max)
		e.detail = item
		rows = append(rows, mysql.Log_Row{
			Type:        e.typ,
			Operate:     e.operate,
			Operator:    e.actor.operator,
			Content:     e.buildContent(),
			Ip:          e.actor.ip,
			Create_Time: time.Now(),
		})
	}
	go func() {
		if err := mysql.Log__Add(rows...); err != nil {
			log.Printf("ERROR 写操作日志失败 operator=%s type=%s op=%s 条数=%d: %v",
				e.actor.operator, e.typ, e.operate, len(rows), err)
		}
	}()
}

// LogMqttWriteValues 内部 MQTT 通路的写值日志（一个点位一条）
func LogMqttWriteValues(topic string, items []Log_Write_item) {
	if len(items) == 0 {
		return
	}
	logEvent{
		actor:   newActor(mysql.Log_Source_Mqtt, 0, "", ""),
		channel: Log_Channel_Mqtt,
		topic:   topic,
		typ:     mysql.Log_Type_Point,
		operate: mysql.Log_Operate_Write,
	}.emitWrites(items)
}

// logLogin 记录一条登陆日志（密码登录、AK/SK 首次使用、MQTT 远程签发令牌共用）
func logLogin(ctx *gin.Context, src mysql.Log_Source, accountId uint, accessKey string, detail any) {
	e := logEvent{
		actor:   newActor(src, accountId, accessKey, ctx.ClientIP()),
		channel: Log_Channel_Http,
		method:  ctx.Request.Method,
		path:    ctx.Request.URL.Path,
		agent:   ctx.Request.UserAgent(),
		typ:     mysql.Log_Type_Account,
		operate: mysql.Log_Operate_Login,
		detail:  detail,
	}
	e.emit()
}

// logLogout 记录一条登出日志
func logLogout(ctx *gin.Context, src mysql.Log_Source, accountId uint, accessKey string, detail any) {
	e := logEvent{
		actor:   newActor(src, accountId, accessKey, ctx.ClientIP()),
		channel: Log_Channel_Http,
		method:  ctx.Request.Method,
		path:    ctx.Request.URL.Path,
		agent:   ctx.Request.UserAgent(),
		typ:     mysql.Log_Type_Account,
		operate: mysql.Log_Operate_Logout,
		detail:  detail,
	}
	e.emit()
}

// ======== 操作方式的中文展示名（供日志查询接口回显；库里存的是英文标识） ========

// Log_Type_Name 操作对象标识 → 可读名称，同时充当合法值白名单
var Log_Type_Name = map[mysql.Log_Type]string{
	mysql.Log_Type_All:     "全部配置",
	mysql.Log_Type_Drive:   "驱动",
	mysql.Log_Type_Point:   "点位",
	mysql.Log_Type_Alarm:   "报警",
	mysql.Log_Type_Log:     "日志",
	mysql.Log_Type_History: "历史",
	mysql.Log_Type_Account: "账号权限",
	mysql.Log_Type_System:  "系统配置",
	mysql.Log_Type_Service: "服务进程",
}

// Log_Operate_Name 操作方式标识 → 可读名称，同时充当合法值白名单
var Log_Operate_Name = map[mysql.Log_Operate]string{
	mysql.Log_Operate_Add:     "增",
	mysql.Log_Operate_Del:     "删",
	mysql.Log_Operate_Update:  "改",
	mysql.Log_Operate_Query:   "查",
	mysql.Log_Operate_Write:   "写值",
	mysql.Log_Operate_Login:   "登陆",
	mysql.Log_Operate_Logout:  "登出",
	mysql.Log_Operate_Restart: "重启",
	mysql.Log_Operate_Import:  "导入",
	mysql.Log_Operate_Export:  "导出",
}

// ============================================================
// 操作日志查询/清理 API（log_operate 单表入口）
// ============================================================

// logQueryRequest 日志查询/计数入参；各列表为空表示该维度不限
// types / operates 直接传库里的文本值（如 ["point"], ["write","login"]），JSON 数组解码到命名 string 切片
type logQueryRequest struct {
	Types     []mysql.Log_Type    `json:"types"`     // 操作对象
	Operates  []mysql.Log_Operate `json:"operates"`  // 操作方式
	Operators []string            `json:"operators"` // 操作人，如 user:1 / aksk:1 / internalmqtt
	Keyword   string              `json:"keyword"`   // 操作人 / 详情模糊匹配
	Start     string              `json:"start"`     // 起始时间，RFC3339 或 2006-01-02 15:04:05
	End       string              `json:"end"`       // 结束时间，同上
	Page      uint                `json:"page"`
	PageSize  uint                `json:"pageSize"`
}

// logDelRequest 日志删除入参
type logDelRequest struct {
	Ids []uint `json:"ids"`
}

// logRowView 日志对外视图：类型与方式同时给出库里的标识和中文名，前端不必再维护映射表
type logRowView struct {
	Id          uint              `json:"id"`
	Type        mysql.Log_Type    `json:"type"`
	TypeName    string            `json:"type_name"`
	Operate     mysql.Log_Operate `json:"operate"`
	OperateName string            `json:"operate_name"`
	Operator    string            `json:"operator"`
	Channel     string            `json:"channel,omitempty"` // 从 Content 里提出来，列表页可直接区分 HTTP/MQTT
	Content     string            `json:"content"`
	Ip          string            `json:"ip"`
	Create_Time time.Time         `json:"create_time"`
}

// parseLogTime 兼容 RFC3339 与 "2006-01-02 15:04:05" 两种时间写法
func parseLogTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("时间格式不支持: %s", s)
}

// buildLogFilter 请求体 → 查询条件；枚举维度先按白名单校验，避免拼错类型静默返回空页
func buildLogFilter(req logQueryRequest) (mysql.Log_Filter, error) {
	for _, t := range req.Types {
		if _, ok := Log_Type_Name[t]; !ok {
			return mysql.Log_Filter{}, fmt.Errorf("未知的日志操作对象: %s", string(t))
		}
	}
	for _, o := range req.Operates {
		if _, ok := Log_Operate_Name[o]; !ok {
			return mysql.Log_Filter{}, fmt.Errorf("未知的日志操作方式: %s", string(o))
		}
	}
	f := mysql.Log_Filter{
		Types:     req.Types,
		Operates:  req.Operates,
		Operators: req.Operators,
		Keyword:   clip(strings.TrimSpace(req.Keyword), Log_Keyword_Max),
		Page:      req.Page,
		PageSize:  req.PageSize,
	}
	var err error
	if f.Start, err = parseLogTime(req.Start); err != nil {
		return f, err
	}
	if f.End, err = parseLogTime(req.End); err != nil {
		return f, err
	}
	return f, nil
}

// logChannel 从 Content 里取出 channel 字段，便于列表页直接显示通路
func logChannel(content string) string {
	if content == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return ""
	}
	if s, ok := m["channel"].(string); ok {
		return s
	}
	return ""
}

// api_log_query 查询操作日志
func api_log_query(ctx *gin.Context) {
	var req logQueryRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	filter, err := buildLogFilter(req)
	if err != nil {
		ctx.Set("Response", []any{417, err.Error()})
		return
	}
	data, err := mysql.Log__Query(filter)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	rows := make([]logRowView, 0, len(data))
	for _, d := range data {
		rows = append(rows, logRowView{
			Id:   d.Id,
			Type: d.Type, TypeName: Log_Type_Name[d.Type],
			Operate: d.Operate, OperateName: Log_Operate_Name[d.Operate],
			Operator: d.Operator, Channel: logChannel(d.Content),
			Content: d.Content, Ip: d.Ip, Create_Time: d.Create_Time,
		})
	}
	ctx.Set("Response", []any{200, "ok", rows})
}

// api_log_count 统计操作日志条数
func api_log_count(ctx *gin.Context) {
	var req logQueryRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	filter, err := buildLogFilter(req)
	if err != nil {
		ctx.Set("Response", []any{417, err.Error()})
		return
	}
	count, err := mysql.Log__Count(filter)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// api_log_del 批量删除操作日志（不可恢复；本次操作本身会再记一条"日志/删"）
func api_log_del(ctx *gin.Context) {
	var req logDelRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if len(req.Ids) == 0 {
		ctx.Set("Response", []any{417, "ids 不能为空"})
		return
	}
	if err := mysql.Log_Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Log, mysql.Log_Operate_Del, gin.H{
		"action": "delete_log", "ids": req.Ids, "count": len(req.Ids),
	})
	ctx.Set("Response", []any{200, "ok"})
}

// log_api_register 注册操作日志查询/清理路由（受统一鉴权中间件保护）
func log_api_register(r *gin.Engine) {
	r.POST("/api/v1.0/config/log/query", api_log_query)
	r.POST("/api/v1.0/config/log/count", api_log_count)
	r.POST("/api/v1.0/config/log/del", api_log_del)
}
