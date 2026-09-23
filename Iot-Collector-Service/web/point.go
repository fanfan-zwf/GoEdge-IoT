/*
* 日期: 2026.02.15 	PM11:06
* 作者: 范范zwf
* 作用: 点位读写 API + WebSocket 实时推送
 */
package web

import (
	"encoding/json"
	"fmt"
	"log"
	"main/IO/manager"
	"main/IO/manager/fullConfig"
	"main/db/db_point"
	"main/db/influxdb"
	"main/db/mysql"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ======== 全局类型映射（写入值类型转换） ========
var typeMap = map[string]reflect.Type{
	"bool":    reflect.TypeOf(bool(false)),
	"int8":    reflect.TypeOf(int8(0)),
	"uint8":   reflect.TypeOf(uint8(0)),
	"int16":   reflect.TypeOf(int16(0)),
	"uint16":  reflect.TypeOf(uint16(0)),
	"int32":   reflect.TypeOf(int32(0)),
	"uint32":  reflect.TypeOf(uint32(0)),
	"int64":   reflect.TypeOf(int64(0)),
	"uint64":  reflect.TypeOf(uint64(0)),
	"int":     reflect.TypeOf(int(0)),
	"uint":    reflect.TypeOf(uint(0)),
	"float32": reflect.TypeOf(float32(0)),
	"float64": reflect.TypeOf(float64(0)),
	"float":   reflect.TypeOf(float64(0)),
	"string":  reflect.TypeOf(""),
}

// ======== HTTP 读写接口 ========

// api_point_read_value 点位实时值读取（HTTP）
func api_point_read_value(ctx *gin.Context) {
	var keys []uint
	err := ctx.BindJSON(&keys)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	r := db_point.Value_Map__Query_list(keys)

	ctx.Set("Response", []any{200, "ok", r})
}

// api_point_write_value 点位写入
func api_point_write_value(ctx *gin.Context) {
	var tempData []struct {
		PointId uint
		Value   json.RawMessage
		Type    string
	}
	err := ctx.BindJSON(&tempData)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 1. 收集所有 PointId，查询对应的 Drive_Id
	pointIds := make([]uint, 0, len(tempData))
	for _, item := range tempData {
		pointIds = append(pointIds, item.PointId)
	}
	driveIdList, err := mysql.Point_Config__Query__DriveIds(pointIds...)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位驱动关系失败: " + err.Error()})
		return
	}
	pointDriveMap := make(map[uint]uint, len(driveIdList))
	for _, d := range driveIdList {
		pointDriveMap[d.Id] = d.Drive_Id
	}

	// 2. 转换值并按 Drive_Id 分组
	driveGroups := make(map[uint][]fullConfig.Value_type)
	for _, item := range tempData {
		var realValue any
		err = json.Unmarshal(item.Value, &realValue)
		if err != nil {
			ctx.Set("Response", []any{500, err.Error()})
			return
		}

		t, ok := typeMap[item.Type]
		if !ok {
			ctx.Set("Response", []any{500, "不存在的类型"})
			return
		}

		realValue = reflect.ValueOf(realValue).Convert(t).Interface()
		driveId, exists := pointDriveMap[item.PointId]
		if !exists {
			ctx.Set("Response", []any{404, fmt.Sprintf("点位 id=%d 不存在", item.PointId)})
			return
		}

		driveGroups[driveId] = append(driveGroups[driveId], fullConfig.Value_type{
			PointId: item.PointId,
			Value:   realValue,
			Type:    item.Type,
			Time:    time.Now(),
		})
	}

	// 3. 按驱动调用写入
	for driveId, values := range driveGroups {
		if err := manager.DriveWrite(driveId, values); err != nil {
			ctx.Set("Response", []any{500, fmt.Sprintf("驱动 id=%d 写入失败: %v", driveId, err)})
			return
		}
	}

	ctx.Set("Response", []any{200, "ok"})
}

// ======== WebSocket 实时推送（只读） ========

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsClient WebSocket 客户端
type wsClient struct {
	conn   *websocket.Conn
	send   chan []fullConfig.Value_type // 传原始结构体，writePump 中批量序列化
	points map[uint]bool                // 订阅的点位 ID 集合
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
	db_point.Update_Subscriber(func(values []fullConfig.Value_type) error {
		wsManager.mu.RLock()
		defer wsManager.mu.RUnlock()

		// 按客户端分组数据，减少序列化次数
		clientData := make(map[*wsClient][]fullConfig.Value_type)
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
// GET /api/v1.0/point/ws?points=[1,2,3]
// URL 参数 points: 点位 ID JSON 数组
func api_point_ws(ctx *gin.Context) {
	// 解析 URL 中的点位 ID JSON 数组
	pointsStr := ctx.Query("points")
	if pointsStr == "" {
		ctx.Set("Response", []any{417, "缺少 points 参数"})
		return
	}

	// 解析 JSON 数组
	var pointIdList []uint
	if err := json.Unmarshal([]byte(pointsStr), &pointIdList); err != nil {
		ctx.Set("Response", []any{417, "points 格式错误，需要 JSON 数组如 [1,2,3]"})
		return
	}

	if len(pointIdList) == 0 {
		ctx.Set("Response", []any{417, "points 不能为空"})
		return
	}

	// 转为 map
	pointIds := make(map[uint]bool)
	for _, id := range pointIdList {
		pointIds[id] = true
	}

	// 升级 WebSocket
	conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("ERROR WebSocket 升级失败: %v", err)
		return
	}

	client := &wsClient{
		conn:   conn,
		send:   make(chan []fullConfig.Value_type, 256),
		points: pointIds,
	}

	// 注册客户端
	wsManager.mu.Lock()
	wsManager.clients[client] = true
	wsManager.mu.Unlock()

	log.Printf("INFO WebSocket 连接建立，订阅点位: %v", pointIds)

	// 启动写入协程（服务端→客户端）
	go client.writePump()
}

// api_alarm_status 报警状态查询
func api_alarm_status(ctx *gin.Context) {
	var tags []uint
	err := ctx.BindJSON(&tags)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	r := db_point.Alarm_Config__Query_list(tags)
	ctx.Set("Response", []any{200, "ok", r})
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
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// 排空 channel，合并同一批次的所有数据，减少 WebSocket 帧数
			for {
				select {
				case more, ok := <-c.send:
					if !ok {
						c.conn.WriteMessage(websocket.CloseMessage, []byte{})
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
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			// 定时 Ping 保活
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// api_history_data_query 查询历史数据（从 InfluxDB）
func api_history_data_query(ctx *gin.Context) {
	var req struct {
		PointId   uint      `json:"pointId"`   // 点位ID
		StartTime time.Time `json:"startTime"` // 开始时间 (RFC3339)
		EndTime   time.Time `json:"endTime"`   // 结束时间 (RFC3339)
		Page      uint      `json:"page"`      // 页码
		PageSize  uint      `json:"pageSize"`  // 每页数量
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if req.PointId == 0 {
		ctx.Set("Response", []any{417, "点位ID不能为空"})
		return
	}

	if req.StartTime.After(req.EndTime) {
		ctx.Set("Response", []any{417, "开始时间不能晚于结束时间"})
		return
	}

	// 调用 InfluxDB 查询接口
	data, total, err := influxdb.QueryByPointIdAndTime(req.PointId, req.StartTime, req.EndTime, req.Page, req.PageSize)
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
	r.GET("/api/v1.0/point/ws", api_point_ws)

	// 报警状态
	r.POST("/api/v1.0/alarm/status", api_alarm_status)

	// 历史数据查询（InfluxDB）
	r.POST("/api/v1.0/history/data/query", api_history_data_query)
}
