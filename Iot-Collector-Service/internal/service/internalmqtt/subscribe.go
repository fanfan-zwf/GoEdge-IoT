package internalmqtt

import (
	"collector/internal/handler"
	"collector/internal/infra/mqtt"
	"collector/internal/model"
	"collector/internal/repo/mysql"
	"collector/internal/repo/redis"
	"collector/internal/service/collector"
	"collector/internal/service/driver"
	"collector/pkg/conv"
	"collector/pkg/security"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

// mqttTokenBytes 生成临时令牌使用的随机字节数（与 handler 登录令牌一致）
const mqttTokenBytes = 32

// 临时令牌默认 TTL（tokenReq 未显式提供时回退）
const (
	defaultAccessTTL  = 15 * time.Minute
	defaultRefreshTTL = 168 * time.Hour
)

// 写值类型名 → reflect.Type（与 handler.point.typeMap 保持一致，用于精确转换下发值）
var writeTypeMap = map[string]reflect.Type{
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

// subscribeCommands 逐个订阅 4 个下行指令主题（不使用 base+"/#" 通配）。
// 只订阅具体指令主题，可避免把本服务自己发布的 /data、/status/*、/result/* 回声收回。
func subscribeCommands() error {
	for _, suffix := range []string{cfg.subWrite, cfg.subRestart, cfg.subToken, cfg.subConfigUpdate} {
		topic := joinTopic(baseTopic, suffix)
		if err := mqtt.Subscribe(cfg.broker, topic, func(_ string, t string, data []byte) {
			dispatch(t, data)
		}); err != nil {
			return err
		}
	}
	return nil
}

// dispatch 去掉主题前缀，按已配置的下行后缀匹配并异步分发（不阻塞 MQTT 回调线程）。
func dispatch(topic string, data []byte) {
	// 主题形如 /IOT/Collector/<label>/write，取 base 之后的相对后缀
	rel := strings.TrimPrefix(topic, baseTopic)
	rel = strings.Trim(rel, "/")

	// 处理器统一收原始主题（不只后缀），日志里能直接看到指令来源
	switch rel {
	case trimSlash(cfg.subWrite):
		go safeHandle("write", handleWrite, topic, data)
	case trimSlash(cfg.subRestart):
		go safeHandle("restart", handleRestart, topic, data)
	case trimSlash(cfg.subToken):
		go safeHandle("token", handleToken, topic, data)
	case trimSlash(cfg.subConfigUpdate):
		go safeHandle("config_update", handleConfigUpdate, topic, data)
	default:
		// 已改为逐个订阅指令主题，正常不会进到这里；保留兼容异常主题
		log.Printf("WARN 内部MQTT收到未处理的主题后缀: %s", rel)
	}
}

// trimSlash 去掉后缀两端的"/"，用于与相对主题比较。
func trimSlash(s string) string { return strings.Trim(s, "/") }

// safeHandle 包裹处理器，recover panic 并记录，避免异常处理函数影响进程。
func safeHandle(name string, fn func(topic string, data []byte), topic string, data []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ERROR 内部MQTT处理[%s] panic: %v", name, r)
		}
	}()
	fn(topic, data)
}

// handleWrite 写数据：解析数组 json（以 tag 为键），反查点位后按类型精确转换，分组调用对应驱动写入。
// 成功与失败都落审计日志（失败带原始报文），与 HTTP 写值接口同构：一个点位一行。
func handleWrite(topic string, data []byte) {
	// fail 统一处理整批拒绝：先落日志再发回执，避免只看到回执看不到现场
	fail := func(tag, reason string) {
		logMqttFail(topic, mysql.Log_Type_Point, mysql.Log_Operate_Write, reason, data,
			map[string]any{"tag": tag})
		publishResult("write", false, reason, nil)
	}

	var items []writeItem
	if err := json.Unmarshal(data, &items); err != nil {
		fail("", "写数据解析失败: "+err.Error())
		return
	}
	if len(items) == 0 {
		fail("", "写数据为空")
		return
	}

	// 逐条按 tag 定位点位、按类型精确转换，产出标准入参数组 []model.Value_type
	values := make([]model.Value_type, 0, len(items))
	for _, it := range items {
		if it.Tag == "" {
			fail(it.Tag, "写数据存在空 tag")
			return
		}

		// 按 tag 反查点位；缓存未命中时刷新一次再重试（应对刚新增的点位）
		ref, ok := lookupPoint(it.Tag)
		if !ok {
			refreshTagMap()
			ref, ok = lookupPoint(it.Tag)
		}
		if !ok {
			fail(it.Tag, fmt.Sprintf("点位 tag=%s 不存在", it.Tag))
			return
		}

		t, ok := writeTypeMap[it.Type]
		if !ok {
			fail(it.Tag, fmt.Sprintf("点位 tag=%s 不存在的类型: %s", it.Tag, it.Type))
			return
		}
		var raw any
		if err := json.Unmarshal(it.Value, &raw); err != nil {
			fail(it.Tag, fmt.Sprintf("点位 tag=%s 值解析失败: %s", it.Tag, err.Error()))
			return
		}
		// json 数字统一为 float64，按目标类型转换
		val := reflect.ValueOf(raw).Convert(t).Interface()

		ts := time.Now()
		if it.Time != "" {
			if parsed, perr := time.Parse(time.RFC3339, it.Time); perr == nil {
				ts = parsed
			}
		}

		values = append(values, model.Value_type{
			PointId: ref.id,
			Tag:     it.Tag,
			Value:   val,
			Type:    conv.ValueTypeCode[it.Type],
			Time:    ts,
		})
	}

	// 走写值标准入口下发（按驱动分组 + 只读校验在 DriveWrite 入口统一执行），与 HTTP 通路完全复用
	if err := collector.WriteValues(values); err != nil {
		logMqttFail(topic, mysql.Log_Type_Point, mysql.Log_Operate_Write, "写入失败: "+err.Error(), data,
			map[string]any{"count": len(values)})
		publishResult("write", false, err.Error(), nil)
		return
	}

	// 写值审计：一个点位一行，值取请求原始 JSON 报文（与 HTTP 通路一致，字符串带引号）
	writes := make([]handler.Log_Write_item, 0, len(values))
	for i := range values {
		writes = append(writes, handler.Log_Write_item{
			PointId: values[i].PointId,
			Tag:     values[i].Tag,
			Type:    items[i].Type,
			Value:   rawValue(items[i].Value),
			Time:    values[i].Time.Format(time.RFC3339),
		})
	}
	logMqttWrites(topic, writes)
	publishResult("write", true, "ok", map[string]any{"count": len(items)})
}

// handleRestart 重启：target=driver 重启指定/全部驱动；target=service 让本进程优雅退出（由守护拉起）。
// 受理、完成、拒绝三类结果都落审计日志，Type 按重启对象区分（驱动 / 服务进程）。
func handleRestart(topic string, data []byte) {
	var req restartReq
	if err := json.Unmarshal(data, &req); err != nil {
		reason := "重启请求解析失败: " + err.Error()
		logMqttFail(topic, mysql.Log_Type_All, mysql.Log_Operate_Restart, reason, data, nil)
		publishResult("restart", false, reason, nil)
		return
	}

	switch strings.ToLower(strings.TrimSpace(req.Target)) {
	case "driver", "drivers":
		// drive_ids 为空表示全部驱动，日志里如实记成 "all"，免得事后误判成没带参数
		var ids any = req.DriveIds
		if len(req.DriveIds) == 0 {
			ids = "all"
		}
		logMqtt(topic, mysql.Log_Type_Drive, mysql.Log_Operate_Restart, map[string]any{
			"action": "restart", "target": "driver", "drive_ids": ids, "stage": "accepted",
		})
		publishResult("restart", true, "驱动重启已启动", nil)
		go func() {
			defer recoverGoroutine("内部MQTT 驱动重启")
			restartDrivers(topic, req.DriveIds)
		}()
	case "service":
		logMqtt(topic, mysql.Log_Type_Service, mysql.Log_Operate_Restart, map[string]any{
			"action": "restart", "target": "service", "stage": "accepted",
			"note": "本进程发送 SIGTERM 优雅退出，等待守护进程拉起",
		})
		publishResult("restart", true, "服务即将退出（等待守护进程重新拉起）", nil)
		// 稍作延时确保回执先发出，再向自身发送 SIGTERM 触发 main 优雅退出
		go func() {
			time.Sleep(1 * time.Second)
			_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		}()
	default:
		reason := "未知的重启目标 target=" + req.Target
		logMqttFail(topic, mysql.Log_Type_All, mysql.Log_Operate_Restart, reason, data, nil)
		publishResult("restart", false, reason, nil)
	}
}

// restartDrivers 重启驱动：drive_ids 为空则重启全部；逐个查询配置后调用 DriveResetConfig（异步、耗时约 9 秒/个）。
// 每个驱动的失败单独记一行，最后再记一行完成汇总（含成功数与总数）。
func restartDrivers(topic string, driveIds []uint) {
	driveConfigs, err := mysql.Drive_Config__Query(0, 0)
	if err != nil {
		reason := "查询驱动配置失败: " + err.Error()
		logMqttFail(topic, mysql.Log_Type_Drive, mysql.Log_Operate_Restart, reason, nil,
			map[string]any{"drive_ids": driveIds})
		publishResult("restart", false, reason, nil)
		return
	}

	want := func(id uint) bool {
		if len(driveIds) == 0 {
			return true
		}
		for _, d := range driveIds {
			if d == id {
				return true
			}
		}
		return false
	}

	var restarted, target int
	for _, dc := range driveConfigs {
		if !want(dc.Id) {
			continue
		}
		target++
		points, perr := mysql.Point_Config__Query([]uint{dc.Id}, 0, 0)
		if perr != nil {
			log.Printf("WARN 内部MQTT重启驱动 id=%d 查询点位失败: %v", dc.Id, perr)
			logMqttFail(topic, mysql.Log_Type_Drive, mysql.Log_Operate_Restart,
				fmt.Sprintf("重启驱动 id=%d 查询点位失败: %s", dc.Id, perr.Error()), nil,
				map[string]any{"drive_id": dc.Id})
			continue
		}
		if err := driver.DriveResetConfig(dc.Id, dc, points); err != nil {
			log.Printf("ERROR 内部MQTT重启驱动 id=%d 失败: %v", dc.Id, err)
			logMqttFail(topic, mysql.Log_Type_Drive, mysql.Log_Operate_Restart,
				fmt.Sprintf("重启驱动 id=%d 失败: %s", dc.Id, err.Error()), nil,
				map[string]any{"drive_id": dc.Id, "points": len(points)})
			continue
		}
		restarted++
	}
	logMqtt(topic, mysql.Log_Type_Drive, mysql.Log_Operate_Restart, map[string]any{
		"action": "restart", "target": "driver", "stage": "done", "restarted": restarted, "total": target,
	})
	publishResult("restart", true, "驱动重启完成", map[string]any{"restarted": restarted})
}

// recoverGoroutine 兜住后台 goroutine 的 panic：子 goroutine 里未处理的 panic 会直接带走整个采集进程，
// 兜住后降级为一条带堆栈的 ERROR 日志。
func recoverGoroutine(what string) {
	if r := recover(); r != nil {
		log.Printf("ERROR 异步任务 %s 发生 panic: %v\n%s", what, r, debug.Stack())
	}
}

// handleToken 增加临时 handler token：为 access_key 生成短(access)/长(refresh)令牌并直写 Redis（不落账户表），
// 令牌绑定 IP 置空以跳过会话 IP 校验；生成结果通过回执主题返回。
// 临时令牌拿到后就能直接访问所有授权接口，等同一次登陆，所以记 Log_Operate_Login；
// 只记目标 access_key / 用户 id / TTL，令牌明文绝不入库。
func handleToken(topic string, data []byte) {
	var req tokenReq
	if err := json.Unmarshal(data, &req); err != nil {
		reason := "令牌请求解析失败: " + err.Error()
		logMqttFail(topic, mysql.Log_Type_Account, mysql.Log_Operate_Login, reason, data, nil)
		publishResult("token", false, reason, nil)
		return
	}
	// fail 统一处理拒绝：access_key 是账号标识而非凭证，写进日志便于追查“给谁签了令牌”
	fail := func(reason string) {
		logMqttFail(topic, mysql.Log_Type_Account, mysql.Log_Operate_Login, reason, data,
			map[string]any{"access_key": req.AccessKey})
		publishResult("token", false, reason, nil)
	}
	if req.AccessKey == "" {
		fail("access_key 不能为空")
		return
	}

	accessTTL := parseTTL(req.AccessTokenTTL, defaultAccessTTL)
	refreshTTL := parseTTL(req.RefreshTokenTTL, defaultRefreshTTL)

	// 缺省两者都签发；可单独只要长令牌（with_access_token=false）或只要短令牌（with_refresh_token=false）
	withAccess := req.WithAccessToken == nil || *req.WithAccessToken
	withRefresh := req.WithRefreshToken == nil || *req.WithRefreshToken
	if !withAccess && !withRefresh {
		fail("with_access_token 与 with_refresh_token 不能同时为 false")
		return
	}

	// 令牌需内嵌主体 id（请求期 RBAC/日志直接取用、不再查库），故按 access_key 解析登录用户
	uacc, found, err := mysql.User__Query_By_Access_Key(req.AccessKey)
	if err != nil {
		fail("查询账号失败: " + err.Error())
		return
	}
	if !found {
		fail("access_key 对应的登录用户不存在")
		return
	}

	session := redis.Auth_Token_Info{
		User_Id:    uacc.Id,
		Access_Key: req.AccessKey,
		IP:         "", // 置空跳过 IP 一致性校验，便于 MQTT 远程签发
		User_Agent: "internal-mqtt",
		Issued_At:  time.Now(),
	}

	result := map[string]any{
		"access_key": req.AccessKey,
		"token_type": "Bearer",
	}

	// 短令牌（access）
	if withAccess {
		accessToken, err := security.GenKey(mqttTokenBytes)
		if err != nil {
			fail("生成访问令牌失败: " + err.Error())
			return
		}
		if err := redis.Auth_Access_Token_Add(accessToken, session, accessTTL); err != nil {
			fail("写入访问令牌失败: " + err.Error())
			return
		}
		result["access_token"] = accessToken
		result["expires_in"] = int64(accessTTL.Seconds())
	}

	// 长令牌（refresh）
	if withRefresh {
		refreshToken, err := security.GenKey(mqttTokenBytes)
		if err != nil {
			fail("生成刷新令牌失败: " + err.Error())
			return
		}
		if err := redis.Auth_Refresh_Token_Add(refreshToken, session, refreshTTL); err != nil {
			fail("写入刷新令牌失败: " + err.Error())
			return
		}
		result["refresh_token"] = refreshToken
		result["refresh_expires"] = int64(refreshTTL.Seconds())
	}

	// 签发成功：记下被代登的账号、签了哪几种令牌与有效期，令牌值本身不入库
	logMqtt(topic, mysql.Log_Type_Account, mysql.Log_Operate_Login, map[string]any{
		"action": "issue_token", "target_access_key": req.AccessKey, "user_id": uacc.Id,
		"with_access": withAccess, "with_refresh": withRefresh,
		"access_ttl": accessTTL.String(), "refresh_ttl": refreshTTL.String(),
		"ip_bound": false, "user_agent": session.User_Agent,
	})
	publishResult("token", true, "ok", result)
}

// parseTTL 解析 duration 字符串，空或非法时回退默认值。
func parseTTL(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

// handleConfigUpdate 更新所有配置：从 json 里的 http 地址下载表格(xlsx)，全量导入后刷新点位缓存与驱动。
// 导入会整表覆写配置，每个 Sheet 各记一条“导入”日志（只记影响条数，不复制全量数据）。
func handleConfigUpdate(topic string, data []byte) {
	var req configUpdateReq
	if err := json.Unmarshal(data, &req); err != nil {
		reason := "配置更新请求解析失败: " + err.Error()
		logMqttFail(topic, mysql.Log_Type_All, mysql.Log_Operate_Import, reason, data, nil)
		publishResult("config_update", false, reason, nil)
		return
	}
	url := req.Url
	if url == "" {
		url = req.DownloadUrl
	}
	// fail 统一处理拒绝，带上下载地址便于事后对账“当时到底导了哪份表”
	fail := func(reason string) {
		logMqttFail(topic, mysql.Log_Type_All, mysql.Log_Operate_Import, reason, data,
			map[string]any{"url": url})
		publishResult("config_update", false, reason, nil)
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fail("缺少合法的 http 下载地址")
		return
	}

	body, err := downloadConfig(url)
	if err != nil {
		fail("下载配置失败: " + err.Error())
		return
	}

	results, err := handler.ImportConfigBytes(body)
	if err != nil {
		fail("导入配置失败: " + err.Error())
		return
	}

	// 导入后立即刷新 tag 缓存，并让各驱动按新点位配置重组（不断连接）
	refreshTagMap()
	refreshAllDrivePoints()

	for _, res := range results {
		logMqtt(topic, configTypeBySheet(res.Sheet), mysql.Log_Operate_Import, map[string]any{
			"action": "config_update", "url": url, "sheet": res.Sheet,
			"added": res.Added, "updated": res.Updated, "deleted": res.Deleted,
		})
	}
	publishResult("config_update", true, "配置更新完成", results)
}

// downloadConfig 下载配置表格，限制最大 50MB，超时 30 秒。
func downloadConfig(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}
	const maxSize = 50 << 20 // 50MB
	return io.ReadAll(io.LimitReader(resp.Body, maxSize))
}

// refreshAllDrivePoints 对所有驱动按最新点位配置动态重组（不重连），复用 driver.DriveUpdatePoints。
func refreshAllDrivePoints() {
	driveConfigs, err := mysql.Drive_Config__Query(0, 0)
	if err != nil {
		log.Printf("WARN 内部MQTT刷新驱动点位失败(查询驱动): %v", err)
		return
	}
	for _, dc := range driveConfigs {
		points, perr := mysql.Point_Config__Query([]uint{dc.Id}, 0, 0)
		if perr != nil {
			log.Printf("WARN 内部MQTT刷新驱动 id=%d 点位失败(查询点位): %v", dc.Id, perr)
			continue
		}
		if err := driver.DriveUpdatePoints(dc.Id, points); err != nil {
			log.Printf("WARN 内部MQTT刷新驱动 id=%d 点位失败: %v", dc.Id, err)
		}
	}
}
