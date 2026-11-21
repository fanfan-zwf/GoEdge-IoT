/*
* 日期: 2026.02.15 	PM11:06
* 作者: 范范zwf
* 作用: 点位读写 API + WebSocket 实时推送
 */
package handler

import (
	"collector/internal/model"
	"collector/internal/repo/influxdb"
	"collector/internal/repo/mysql"
	"collector/internal/service/collector"
	"collector/internal/service/driver"
	rbacsvc "collector/internal/service/rbac"
	"collector/pkg/conv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ======== HTTP 读写接口 ========

// api_point_read_value 点位实时值读取（HTTP）
// 请求体为裸数组，元素可以是点位 Tag（推荐）或旧的点位 id，两种可混用：
//
//	["//0001//AI01", "//0001//A10"]   按点位标识
//	[1, 2]                            按点位 id（兼容旧前端）
func api_point_read_value(ctx *gin.Context) {
	var raw []any
	if err := ctx.BindJSON(&raw); err != nil {
		ctx.Set("Response", []any{417, `请求格式不对，需要 JSON 数组，如 ["//0001//AI01"] 或 [1,2]`})
		return
	}

	ids, tags, bad := splitIdTagArray(raw)
	if len(bad) > 0 {
		ctx.Set("Response", []any{417, "数组元素只能是点位Tag(字符串)或点位ID(整数): " + strings.Join(bad, ", ")})
		return
	}

	keys, missing, err := mergePointIds(tags, ids)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位Tag失败: " + err.Error()})
		return
	}
	if len(missing) > 0 {
		ctx.Set("Response", []any{404, tagResolveError(missing)})
		return
	}

	// 按点位读权限过滤：仅返回有 /point/read/{id} 授权的点位（通配 /point/read/* = 全部）
	if allowed, all := valueAllowed(ctx, "/point/read", keys); !all {
		keys = allowed
	}

	r := collector.Value_Map__Query_list(keys)
	fillValueTags(r) // 缓存里的老数据可能没带 Tag，按 id 补全后统一返回

	ctx.Set("Response", []any{200, "ok", r})
}

// api_point_write_value 点位写入
// 定位点位优先用 Tag，未传 Tag 时兼容旧的 pointId 写法；
// 解析转换走 collector.PrepareWrite，下发走 collector.WriteValues（API 与 MQTT 共用的标准入口），本函数只做协议适配与审计。
func api_point_write_value(ctx *gin.Context) {
	var req []struct {
		Tag     string // 点位标识（优先使用）
		PointId uint   // 点位ID（未传 Tag 时按 id 定位）
		Value   json.RawMessage
		Type    string
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	inputs := make([]collector.WriteInput, 0, len(req))
	for _, item := range req {
		inputs = append(inputs, collector.WriteInput{
			Tag:     item.Tag,
			PointId: item.PointId,
			Value:   item.Value,
			Type:    item.Type,
		})
	}

	// 1. 标准入口（解析）：定位点位、精确转换为 []model.Value_type（任一非法即整批拒绝，不产生部分写入）
	values, err := collector.PrepareWrite(inputs)
	if err != nil {
		ctx.Set("Response", []any{writeErrCode(err), err.Error()})
		return
	}

	// 2. 按点位写权限过滤：仅下发有 /point/write/{id} 授权的点位（通配 /point/write/* = 全部）。
	//    values 与 req 下标严格对齐，过滤时同步保留原始报文供审计。
	idsOf := make([]uint, 0, len(values))
	for i := range values {
		idsOf = append(idsOf, values[i].PointId)
	}
	allowed, all := valueAllowed(ctx, "/point/write", idsOf)
	var allowedSet map[uint]bool
	if !all {
		allowedSet = make(map[uint]bool, len(allowed))
		for _, id := range allowed {
			allowedSet[id] = true
		}
	}
	type writePair struct {
		val model.Value_type
		raw json.RawMessage
	}
	pairs := make([]writePair, 0, len(values))
	for i := range values {
		if allowedSet == nil || allowedSet[values[i].PointId] {
			pairs = append(pairs, writePair{values[i], req[i].Value})
		}
	}
	if len(pairs) == 0 {
		ctx.Set("Response", []any{403, "无权限操作所选资源"})
		return
	}
	writeVals := make([]model.Value_type, 0, len(pairs))
	for _, p := range pairs {
		writeVals = append(writeVals, p.val)
	}

	// 3. 标准入口（下发）：按驱动分组逐驱动写入，只读属性在 DriveWrite 入口统一校验
	if err := collector.WriteValues(writeVals); err != nil {
		ctx.Set("Response", []any{writeErrCode(err), err.Error()})
		return
	}

	// 4. 写值审计：以过滤后的 pairs 为准，一个点位一行；值保留请求原始 JSON 报文（字符串会带引号），
	//    同时带上点位标识 Tag 与类型名，事后不必反查点位表就能看懂写了什么
	items := make([]Log_Write_item, 0, len(pairs))
	for _, p := range pairs {
		items = append(items, Log_Write_item{
			PointId: p.val.PointId,
			Tag:     p.val.Tag,
			Type:    conv.ValueTypeIntToString(p.val.Type),
			Value:   string(p.raw),
			Time:    p.val.Time.Format(time.RFC3339),
		})
	}
	logWriteValues(ctx, items)

	ctx.Set("Response", []any{200, "ok"})
}

// writeErrCode 把写值标准入口返回的哨兵错误// 解析转换走 collector.PrepareWrite，下发走 collector.WriteValues（API 与 MQTT 共用的标准入口），本函数只做协议适配与审计。映射为业务码：
// 定位失败→404，参数非法/点位不可写→417，其余（DB/设备写入失败）→500
func writeErrCode(err error) int {
	switch {
	case errors.Is(err, collector.ErrWriteNotFound):
		return 404
	case errors.Is(err, collector.ErrWriteInvalid), errors.Is(err, driver.ErrPointNotWritable):
		return 417
	default:
		return 500
	}
}

// ======== WebSocket 实时推送（只读） ========

// PathPointWS 实时推送端点。浏览器 new WebSocket() 无法自定义请求头，
// 因此令牌只能借 Sec-WebSocket-Protocol 子协议位传递（鉴权中间件仅对该路径接受此方式）。
const PathPointWS = "/api/v1.0/point/ws"

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsClient WebSocket 客户端
type wsClient struct {
	conn   *websocket.Conn
	send   chan []model.Value_type // 传原始结构体，writePump 中批量序列化
	points map[uint]bool           // 订阅的点位 ID 集合

	// done 用于通知 writePump 退出：writePump 平时阻塞在 select 上，仅靠 conn.Close()
	// 叫不醒它（要等到下一条数据或 30s 心跳才报错），服务关闭时就会留下挂死的协程。
	// 不能改成 close(send)：订阅回调对已关闭 channel 发数据会 panic。
	done      chan struct{}
	closeOnce sync.Once

	// writeMu 串行化对同一连接的所有写操作（gorilla 不允许多协程并发写），
	// writePump 与退出时的 WSCloseAll 都会写，必须互斥。
	writeMu sync.Mutex
}

// stop 幂等地通知 writePump 退出。
func (c *wsClient) stop() {
	c.closeOnce.Do(func() { close(c.done) })
}

// wsManager WebSocket 客户端管理
var wsManager = struct {
	clients map[*wsClient]bool
	mu      sync.RWMutex
}{
	clients: make(map[*wsClient]bool),
}

func init() {
	// 订阅数据采集流，推送给所有 WebSocket 客户端
	collector.Update_Subscriber(func(values []model.Value_type) error {
		wsManager.mu.RLock()
		defer wsManager.mu.RUnlock()

		// 按客户端分组数据，减少序列化次数
		clientData := make(map[*wsClient][]model.Value_type)
		for _, v := range values {
			for client := range wsManager.clients {
				if client.points[v.PointId] {
					clientData[client] = append(clientData[client], v)
				}
			}
		}

		// 将分组后的数据发送到各个客户端
		for client, data := range clientData {
			if len(data) > 0 {
				select {
				case client.send <- data:
				default:
					// 客户端消费太慢，跳过
				}
			}
		}
		return nil
	})
}

// api_point_ws WebSocket 实时点位推送（只读）
// GET /api/v1.0/point/ws?points=["//0001//AI01","//0001//A10"]
// URL 参数 points: 点位标识 Tag 的 JSON 数组；仍兼容旧的点位 id 数组 [1,2,3]，两种可混用
func api_point_ws(ctx *gin.Context) {
	// 解析 URL 中的点位 JSON 数组（元素可为 Tag 字符串或点位 id 数字）
	pointsStr := ctx.Query("points")
	if pointsStr == "" {
		ctx.Set("Response", []any{417, "缺少 points 参数"})
		return
	}

	var raw []any
	if err := json.Unmarshal([]byte(pointsStr), &raw); err != nil {
		ctx.Set("Response", []any{417, `points 格式错误，需要 JSON 数组如 ["//0001//AI01"] 或 [1,2,3]`})
		return
	}

	ids, tags, bad := splitIdTagArray(raw)
	if len(bad) > 0 {
		ctx.Set("Response", []any{417, "points 元素只能是点位Tag(字符串)或点位ID(整数): " + strings.Join(bad, ", ")})
		return
	}

	if len(ids)+len(tags) == 0 {
		ctx.Set("Response", []any{417, "points 不能为空"})
		return
	}

	// Tag 先反查成 id 再订阅：推送过滤与实时值缓存内部都以点位 id 为键
	pointIdList, missing, err := mergePointIds(tags, ids)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位Tag失败: " + err.Error()})
		return
	}
	if len(missing) > 0 {
		ctx.Set("Response", []any{404, tagResolveError(missing)})
		return
	}

	// 按点位读权限过滤：仅订阅有 /point/read/{id} 授权的点位（与读值接口共用同一权限）
	if allowed, all := valueAllowed(ctx, "/point/read", pointIdList); !all {
		pointIdList = allowed
	}
	if len(pointIdList) == 0 {
		ctx.Set("Response", []any{403, "无权限订阅这些点位的实时数据"})
		return
	}

	// 转为 map
	pointIds := make(map[uint]bool, len(pointIdList))
	for _, id := range pointIdList {
		pointIds[id] = true
	}

	// 升级 WebSocket。若客户端借子协议位携带了令牌，按 RFC 6455 必须回选一个子协议，否则浏览器会中断握手。
	// 注意：gorilla 会 Hijack 连接并手工拼装 101 响应，改 ctx.Writer.Header() 一律无效，
	// 回选只能经 Upgrade 的 responseHeader 参数（键名为 gorilla 约定的 Sec-Websocket-Protocol）。
	var respHeader http.Header
	if proto := wsSubprotocolToken(ctx); proto != "" {
		respHeader = http.Header{}
		respHeader.Set("Sec-Websocket-Protocol", proto)
	}
	conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, respHeader)
	if err != nil {
		log.Printf("ERROR WebSocket 升级失败: %v", err)
		return
	}

	client := &wsClient{
		conn:   conn,
		send:   make(chan []model.Value_type, 256),
		points: pointIds,
		done:   make(chan struct{}),
	}

	// 注册客户端
	wsManager.mu.Lock()
	wsManager.clients[client] = true
	wsManager.mu.Unlock()

	log.Printf("INFO WebSocket 连接建立，订阅点位: %v", pointIds)

	// 启动写入协程（服务端→客户端）
	go client.writePump()
}

// WSCloseAll 优雅退出时主动通知全部 WebSocket 客户端：
// 先按 RFC 6455 发 1001（Going Away）关闭帧再断连，前端能立即拿到正常的关闭事件，
// 而不是等 TCP 被强制关掉后的异常断开。
func WSCloseAll(reason string) {
	wsManager.mu.Lock()
	clients := make([]*wsClient, 0, len(wsManager.clients))
	for c := range wsManager.clients {
		clients = append(clients, c)
	}
	wsManager.mu.Unlock() // 持锁期间不做网络写，避免慢客户端拖住退出流程

	for _, c := range clients {
		_ = c.writeMsg(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, reason))
		c.stop()           // 叫醒 writePump，让它自己退出并从管理器注销
		_ = c.conn.Close() // 兜底：客户端不回 close 帧时也不会挂在 IO 上
	}
	if len(clients) > 0 {
		log.Printf("INFO 已通知 %d 个 WebSocket 客户端服务即将关闭", len(clients))
	}
}

// alarmStatusRow 报警状态响应：直接嵌服务缓存结构，另补一个点位标识 Point_Tag
// 报警语境里 Tag 已经被「报警规则名」占用，点位标识只能另起字段名，避免两个含义撞车
type alarmStatusRow struct {
	collector.Alarm_Config_type
	Point_Tag string     // 点位标识（按 PointId 反查补全）
	Time      *time.Time `json:"Time"` // 覆盖内嵌 Time：未报警（零值）时输出 null，避免前端显示 1/1/1
}

// newAlarmStatusRow 组装报警状态响应行，把零值 Time 归一为 null
func newAlarmStatusRow(cfg collector.Alarm_Config_type, pointTag string) alarmStatusRow {
	row := alarmStatusRow{
		Alarm_Config_type: cfg,
		Point_Tag:         pointTag,
	}
	if !cfg.Time.IsZero() {
		t := cfg.Time
		row.Time = &t
	}
	return row
}

// api_alarm_status 报警状态查询
// 请求体为裸数组，元素可以是点位 Tag（推荐）或旧的点位 id：["//0001//AI01"] / [1,2]
func api_alarm_status(ctx *gin.Context) {
	var raw []any
	if err := ctx.BindJSON(&raw); err != nil {
		ctx.Set("Response", []any{417, `请求格式不对，需要 JSON 数组，如 ["//0001//AI01"] 或 [1,2]`})
		return
	}

	ids, tags, bad := splitIdTagArray(raw)
	if len(bad) > 0 {
		ctx.Set("Response", []any{417, "数组元素只能是点位Tag(字符串)或点位ID(整数): " + strings.Join(bad, ", ")})
		return
	}

	// 报警配置按点位 id 缓存，Tag 先反查成 id 再查报警状态
	pointIds, missing, err := mergePointIds(tags, ids)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位Tag失败: " + err.Error()})
		return
	}
	if len(missing) > 0 {
		ctx.Set("Response", []any{404, tagResolveError(missing)})
		return
	}

	configs := collector.Alarm_Config__Query_list(pointIds)

	// 缓存结构里只有 PointId，按 id 补出点位标识；补全失败只记 WARN，不影响报警状态查询
	tagIds := make([]uint, 0, len(configs))
	for _, cfg := range configs {
		tagIds = append(tagIds, cfg.PointId)
	}
	tagMap := tagsByIdsSoft(tagIds)

	rows := make([]alarmStatusRow, 0, len(configs))
	for _, cfg := range configs {
		rows = append(rows, newAlarmStatusRow(cfg, tagMap[cfg.PointId]))
	}

	ctx.Set("Response", []any{200, "ok", rows})
}

// writeMsg 统一写入口：写完一帧（含设置写超时）。
func (c *wsClient) writeMsg(msgType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.conn.WriteMessage(msgType, data)
}

// writePump 向客户端推送数据
func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		// 注销客户端
		wsManager.mu.Lock()
		delete(wsManager.clients, c)
		wsManager.mu.Unlock()
		c.conn.Close()
		log.Printf("INFO WebSocket 连接关闭")
	}()

	for {
		select {
		case batch, ok := <-c.send:
			if !ok {
				_ = c.writeMsg(websocket.CloseMessage, []byte{})
				return
			}
			// 排空 channel，合并同一批次的所有数据，减少 WebSocket 帧数
			for {
				select {
				case more, ok := <-c.send:
					if !ok {
						_ = c.writeMsg(websocket.CloseMessage, []byte{})
						return
					}
					batch = append(batch, more...)
				default:
					goto done
				}
			}
		done:
			// 一次性序列化整个批次
			data, err := json.Marshal(batch)
			if err != nil {
				continue
			}
			if err := c.writeMsg(websocket.TextMessage, data); err != nil {
				return
			}

		case <-c.done:
			return

		case <-ticker.C:
			// 定时 Ping 保活
			if err := c.writeMsg(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// api_history_data_query 查询历史数据（从 InfluxDB）
func api_history_data_query(ctx *gin.Context) {
	var req struct {
		Tag       string    `json:"tag"`       // 点位标识（优先使用，InfluxDB 直接按此检索）
		PointId   uint      `json:"pointId"`   // 点位ID（未传 tag 时按 id 反查 tag）
		StartTime time.Time `json:"startTime"` // 开始时间 (RFC3339)
		EndTime   time.Time `json:"endTime"`   // 结束时间 (RFC3339)
		Page      uint      `json:"page"`      // 页码
		PageSize  uint      `json:"pageSize"`  // 每页数量
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if req.StartTime.After(req.EndTime) {
		ctx.Set("Response", []any{417, "开始时间不能晚于结束时间"})
		return
	}

	// 历史数据在 InfluxDB 中以点位标识 Tag 作为序列维度，只传 pointId 时先反查 Tag
	tag, err := resolveTag(req.Tag, req.PointId)
	if err != nil {
		ctx.Set("Response", []any{417, err.Error()})
		return
	}

	// 调用 InfluxDB 查询接口
	data, total, err := influxdb.QueryByTagAndTime(tag, req.StartTime, req.EndTime, req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", map[string]interface{}{"data": data, "total": total}})
}

// point_api_register 注册点位相关路由
func point_api_register(r *gin.Engine) {
	r.POST("/api/v1.0/point/read/value", api_point_read_value)
	r.POST("/api/v1.0/point/write/value", api_point_write_value)
	r.GET(PathPointWS, api_point_ws)

	// 报警状态
	r.POST("/api/v1.0/alarm/status", api_alarm_status)

	// 历史数据查询（InfluxDB）
	r.POST("/api/v1.0/history/data/query", api_history_data_query)
}

// ============================================================
// Tag ↔ 点位id 适配工具（原 point_tag.go）
//
// 对外接口统一以点位标识 Tag 定位点位（与 InfluxDB 历史数据的序列维度一致），
// 而内存实时值缓存、报警/历史配置表、驱动写入内部仍以点位 id 为键，
// 因此请求进来先把 Tag 反查成 id，响应里要展示 Tag 的再按 id 批量补回。
// pointId 入参继续兼容（Tag 优先），前端可平滑迁移，不必一次性切换。
// ============================================================

// maxSafePointId 点位 id 的上限：超过它 float64 已无法精确表示整数值，视为非法入参
const maxSafePointId = float64(1 << 53)

// tagResolveError 把反查不到的 Tag 拼成统一错误文案，便于调用方定位是哪几个点位写错了
func tagResolveError(tags []string) string {
	return fmt.Sprintf("点位不存在: %s", strings.Join(tags, ", "))
}

// idsByTags 按 Tag 批量反查点位 id
// 返回：idMap（Tag -> id，只含库里存在的）、missing（库里查不到的 Tag）
func idsByTags(tags []string) (idMap map[string]uint, missing []string, err error) {
	idMap = make(map[string]uint, len(tags))
	if len(tags) == 0 {
		return
	}

	list, err := mysql.Point_Config__Query__IdTag_ByTags(tags...)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range list {
		idMap[item.Tag] = item.Id
	}

	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, ok := idMap[tag]; !ok {
			missing = append(missing, tag)
		}
	}
	return idMap, missing, nil
}

// mergePointIds 合并两种入参：Tag 列表（优先）与 pointId 列表（兼容旧写法），返回去重后的点位 id
// merged 保持入参顺序，避免同一请求每次得到的 id 顺序抖动
func mergePointIds(tags []string, ids []uint) (merged []uint, missing []string, err error) {
	merged = make([]uint, 0, len(tags)+len(ids))
	seen := make(map[uint]bool, len(tags)+len(ids))

	idMap, miss, err := idsByTags(tags)
	if err != nil {
		return nil, nil, err
	}
	for _, tag := range tags {
		if id, ok := idMap[tag]; ok && !seen[id] {
			seen[id] = true
			merged = append(merged, id)
		}
	}

	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		merged = append(merged, id)
	}

	return merged, miss, nil
}

// splitIdTagArray 解析裸数组入参（如 [1,2] 或 ["//0001//AI01"]，两种可混用）
// 数字按点位 id、字符串按 Tag；负数、小数等非法元素归入 bad，由调用方决定如何报错
func splitIdTagArray(raw []any) (ids []uint, tags []string, bad []string) {
	for _, v := range raw {
		switch item := v.(type) {
		case string:
			if item != "" {
				tags = append(tags, item)
			}
		case float64:
			if item < 0 || item > maxSafePointId || item != math.Trunc(item) {
				bad = append(bad, fmt.Sprintf("%v", item))
				continue
			}
			if id := uint(item); id > 0 {
				ids = append(ids, id)
			}
		default:
			bad = append(bad, fmt.Sprintf("%v", item))
		}
	}
	return
}

// tagsByIds 按点位 id 批量查询 Tag（用于响应体补全），库里不存在或 Tag 为空的 id 不会出现在结果中
func tagsByIds(ids []uint) (tagMap map[uint]string, err error) {
	tagMap = make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return
	}

	list, err := mysql.Point_Config__Query__IdTag_ByIds(ids...)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if item.Tag != "" {
			tagMap[item.Id] = item.Tag
		}
	}
	return tagMap, nil
}

// tagsByIdsSoft 同 tagsByIds，但查询失败只记 WARN 并返回空结果
// 用于响应体里 Tag 这类附加字段：不能因为补全失败就把主查询结果一起拒掉
func tagsByIdsSoft(ids []uint) (tagMap map[uint]string) {
	tagMap, err := tagsByIds(ids)
	if err != nil {
		log.Printf("WARN 按点位id补全Tag失败: %v", err)
		return map[uint]string{}
	}
	return tagMap
}

// fillValueTags 就地补全实时值里缺失的 Tag
// 正常链路驱动已把 Tag 填进 Value_type，这里只兜住点位配置刚改完、缓存里还是旧数据的场景
func fillValueTags(values []collector.Update_Value_type) {
	missing := make([]uint, 0, len(values))
	for i := range values {
		if values[i].Tag == "" && values[i].PointId != 0 {
			missing = append(missing, values[i].PointId)
		}
	}
	if len(missing) == 0 {
		return
	}

	tagMap := tagsByIdsSoft(missing)
	for i := range values {
		if values[i].Tag == "" {
			values[i].Tag = tagMap[values[i].PointId]
		}
	}
}

// resolveTag 把「Tag 优先 + pointId 兼容」两个入参合成一个 Tag，供只接受单一标点的查询接口使用
func resolveTag(tag string, id uint) (string, error) {
	if tag != "" {
		return tag, nil
	}
	if id == 0 {
		return "", fmt.Errorf("点位标识(tag)与点位ID(pointId)不能同时为空")
	}

	result, err := mysql.Point_Config__Query__Tag(id)
	if err != nil {
		return "", fmt.Errorf("点位 id=%d 不存在或查询失败", id)
	}
	return result, nil
}

// pointLocator 配置写入请求里的一条点位定位信息：点位标识优先，点位 id 兼容
type pointLocator struct {
	Tag string
	Id  uint
}

// resolvePointIds 批量解析配置写入请求里的点位 id：给了点位标识的按标识反查，没给的沿用原 id
// 返回的 ids 与入参等长；任一条定位失败就返回错误，错误文案里带上是第几条，前端可直接展示
//
// 只做「点位标识 -> 点位id」的换算，不会反过来去写请求里的同名字段：
// Alarm_Config.Tag 装的是「报警规则名」，与点位标识不是一回事，回填会覆掉用户写的规则名
func resolvePointIds(items []pointLocator) (ids []uint, err error) {
	if len(items) == 0 {
		return
	}

	tags := make([]string, 0, len(items))
	for _, item := range items {
		if item.Tag != "" {
			tags = append(tags, item.Tag)
		}
	}

	tagIdMap, missing, err := idsByTags(tags)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("点位不存在: %s", strings.Join(missing, ", "))
	}

	ids = make([]uint, len(items))
	for i, item := range items {
		if item.Tag != "" {
			ids[i] = tagIdMap[item.Tag]
			continue
		}
		ids[i] = item.Id
		if ids[i] == 0 {
			return nil, fmt.Errorf("第 %d 条：点位标识与点位ID不能同时为空", i+1)
		}
	}

	return ids, nil
}

// ============================================================
// 实时点位读写值 / WS 订阅的按点位 id 授权辅助（原 point_rbac.go）
//
// 三个接口（/point/read/value、/point/write/value、/point/ws）在 rbac_instance 中登记为
// kindValueDefer——接口级不整体拦截，由 handler 把入参解析成点位数字 id 后调用本函数，
// 按 /point/read/{id}（读值与 WS 共用）或 /point/write/{id}（写值）逐个判权并过滤。
// 主题口径与 config 实例级一致：通配 /point/read/* = 全部放行，否则仅显式授权的 id 子集。
// ============================================================

// pointSubject 从鉴权中间件写入的上下文取 RBAC 主体类型与 id（与 Rbac_Middleware 同口径）。
// 缺主体（异常）返回 ok=false，调用方按从严处理。
func pointSubject(ctx *gin.Context) (subjectType uint64, subjectId uint, ok bool) {
	srcVal, hasSrc := ctx.Get(CtxKeyLogSource)
	idVal, hasId := ctx.Get(CtxKeyLogAccount)
	if !hasSrc || !hasId {
		return 0, 0, false
	}
	src, o1 := srcVal.(mysql.Log_Source)
	id, o2 := idVal.(uint)
	if !o1 || !o2 {
		return 0, 0, false
	}
	return subjectTypeFromSrc(src), id, true
}

// valueAllowed 依 base 主题（/point/read 或 /point/write）对候选点位 id 求授权子集。
// all=true：主体持有通配，allowed 原样返回（不限制）；all=false：allowed 为有权子集（可能为空）。
// 无鉴权主体时视为全部无权（all=false, allowed=nil），从严拒绝。
// action 维度已退化，此处固定传 "view"，仅作日志。
func valueAllowed(ctx *gin.Context, base string, ids []uint) (allowed []uint, all bool) {
	st, sid, ok := pointSubject(ctx)
	if !ok {
		return nil, false
	}
	if rbacsvc.ScopeAll(st, sid, base, "view") {
		return ids, true
	}
	return rbacsvc.FilterIds(st, sid, base, "view", ids), false
}
