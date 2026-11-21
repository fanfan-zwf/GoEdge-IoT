/*
* 日期: 2026.09.22
* 作者: 范范zwf
* 作用: 配置管理 API（驱动/点位/报警/历史 的增删改查）
 */
package handler

import (
	"collector/internal/repo/mysql"
	"collector/internal/service/collector"
	"collector/internal/service/driver"
	"log"
	"sync"

	"github.com/gin-gonic/gin"
)

// ======== 驱动配置 API ========

// api_drive_config_query 查询驱动配置
func api_drive_config_query(ctx *gin.Context) {
	var req struct {
		Page     uint `json:"page"`
		PageSize uint `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	configs, err := mysql.Drive_Config__Query(req.Page, req.PageSize, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// api_drive_config_add 新增驱动配置
func api_drive_config_add(ctx *gin.Context) {
	var configs []mysql.Drive_Config_Add_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if err := mysql.Drive_Config__Add(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Drive, mysql.Log_Operate_Add, configs)
	ctx.Set("Response", []any{200, "ok"})
}

// api_drive_config_update 更新驱动配置
func api_drive_config_update(ctx *gin.Context) {
	var configs []mysql.Drive_Config_Update_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// RBAC 实例级：仅保留有权限的行（无权限 id 自动剔除；全无权限已由中间件 403）
	if subset, present := opIdsFromCtx(ctx); present {
		set := idSet(subset)
		kept := configs[:0]
		for _, c := range configs {
			if set[c.Id] {
				kept = append(kept, c)
			}
		}
		configs = kept
	}

	if err := mysql.Drive_Config__Update(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Drive, mysql.Log_Operate_Update, configs)
	ctx.Set("Response", []any{200, "ok"})
}

// api_drive_config_del 删除驱动配置
func api_drive_config_del(ctx *gin.Context) {
	var req struct {
		Ids []uint `json:"ids"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// RBAC 实例级：只删有权限的 id（全无权限已由中间件 403）
	if subset, present := opIdsFromCtx(ctx); present {
		req.Ids = subset
	}

	if err := mysql.Drive_Config__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Drive, mysql.Log_Operate_Del, gin.H{"ids": req.Ids, "count": len(req.Ids)})
	ctx.Set("Response", []any{200, "ok"})
}

// api_drive_config_count 查询驱动配置数量
func api_drive_config_count(ctx *gin.Context) {
	var req struct {
		Page     uint `json:"page"`
		PageSize uint `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := mysql.Drive_Config__Count(req.Page, req.PageSize, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// api_drive_config_restart 重启驱动配置（异步执行，约9秒完成）
func api_drive_config_restart(ctx *gin.Context) {
	var req struct {
		Id uint `json:"id"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if req.Id == 0 {
		ctx.Set("Response", []any{417, "驱动ID不能为空"})
		return
	}

	// 检查驱动是否忙碌
	if driver.IsDriverBusy(req.Id) {
		ctx.Set("Response", []any{403, "驱动正在重启中，请稍后再试"})
		return
	}

	// 查询驱动配置
	driveConfigs, err := mysql.Drive_Config__Query(0, 0)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	var driveCfg mysql.Drive_Config_Query_type
	found := false
	for _, cfg := range driveConfigs {
		if cfg.Id == req.Id {
			driveCfg = cfg
			found = true
			break
		}
	}
	if !found {
		ctx.Set("Response", []any{404, "驱动不存在"})
		return
	}

	// 查询点位配置
	points, err := mysql.Point_Config__Query([]uint{req.Id}, 0, 0)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	logOperate(ctx, mysql.Log_Type_Drive, mysql.Log_Operate_Restart, gin.H{
		"action": "restart", "drive_id": req.Id, "points": len(points),
	})

	// 异步执行重启
	go func() {
		defer guardGoroutine("驱动重启")
		if err := driver.DriveResetConfig(req.Id, driveCfg, points); err != nil {
			log.Printf("ERROR 驱动 id=%d 重启失败: %v", req.Id, err)
		} else {
			log.Printf("INFO 驱动 id=%d 重启成功", req.Id)
		}
	}()

	ctx.Set("Response", []any{200, "驱动重启已启动，预计9秒完成"})
}

// api_drive_config_search 搜索驱动配置（Name 模糊 + Type 过滤）
func api_drive_config_search(ctx *gin.Context) {
	var req struct {
		Name     string   `json:"name"`
		Types    []string `json:"types"`
		Quantity uint     `json:"quantity"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	configs, err := mysql.Drive_Config__Search(req.Name, req.Types, req.Quantity, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// ======== 点位配置 API ========

// api_point_config_query 查询点位配置
func api_point_config_query(ctx *gin.Context) {
	var req struct {
		DriveId  []uint `json:"driveId"`
		Page     uint   `json:"page"`
		PageSize uint   `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	configs, err := mysql.Point_Config__Query(req.DriveId, req.Page, req.PageSize, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// asyncRefreshDrivePoints 异步刷新受影响的驱动点位配置
func asyncRefreshDrivePoints(driveIds []uint) {
	// 去重
	seen := make(map[uint]bool)
	var unique []uint
	for _, id := range driveIds {
		if id > 0 && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	for _, driveId := range unique {
		go func(did uint) {
			defer guardGoroutine("点位刷新")
			points, err := mysql.Point_Config__Query([]uint{did}, 0, 0)
			if err != nil {
				log.Printf("ERROR 驱动 id=%d 点位查询失败: %v", did, err)
				return
			}
			if err := driver.DriveUpdatePoints(did, points); err != nil {
				log.Printf("WARN 驱动 id=%d 点位动态更新失败: %v", did, err)
			} else {
				log.Printf("INFO 驱动 id=%d 点位动态更新成功，点位数=%d", did, len(points))
			}
		}(driveId)
	}
}

// tagTreeRebuildMu 串行化点位树重建，避免并发点位写触发多重重建相互覆盖
var tagTreeRebuildMu sync.Mutex

// refreshTagTree 在 Point_Config 变更后同步重建 tag_tree；
// 失败仅记日志，不影响点位增删改的主流程（可手动调 /tagtree/rebuild 补偿）
func refreshTagTree() {
	tagTreeRebuildMu.Lock()
	defer tagTreeRebuildMu.Unlock()
	if err := mysql.Tag_Tree__RebuildFromPointConfig(); err != nil {
		log.Printf("WARN 重建点位树(tag_tree)失败: %v", err)
		return
	}
	log.Printf("INFO 点位树(tag_tree)已随点位变更自动重建")
}

// api_point_config_add 新增点位配置
func api_point_config_add(ctx *gin.Context) {
	var configs []mysql.Point_Config_Add_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if err := mysql.Point_Config__Add(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Point, mysql.Log_Operate_Add, configs)

	// 收集受影响的驱动ID，异步刷新点位
	var driveIds []uint
	for _, c := range configs {
		driveIds = append(driveIds, c.Drive_Id)
	}
	asyncRefreshDrivePoints(driveIds)

	// 点位新增后同步重建树，前端随后刷新即可看到最新层级
	refreshTagTree()

	ctx.Set("Response", []any{200, "ok"})
}

// api_point_config_update 更新点位配置
func api_point_config_update(ctx *gin.Context) {
	var configs []mysql.Point_Config_Update_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// RBAC 实例级：仅保留有权限的行
	if subset, present := opIdsFromCtx(ctx); present {
		set := idSet(subset)
		kept := configs[:0]
		for _, c := range configs {
			if set[c.Id] {
				kept = append(kept, c)
			}
		}
		configs = kept
	}

	if err := mysql.Point_Config__Update(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Point, mysql.Log_Operate_Update, configs)

	// 收集受影响的驱动ID，异步刷新点位
	var driveIds []uint
	for _, c := range configs {
		driveIds = append(driveIds, c.Drive_Id)
	}
	asyncRefreshDrivePoints(driveIds)

	// Tag 可能随编辑变化，同步重建树以保证层级与点位一致
	refreshTagTree()

	ctx.Set("Response", []any{200, "ok"})
}

// api_point_config_del 删除点位配置
func api_point_config_del(ctx *gin.Context) {
	var req struct {
		Ids []uint `json:"ids"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// RBAC 实例级：只删有权限的 id
	if subset, present := opIdsFromCtx(ctx); present {
		req.Ids = subset
	}

	if err := mysql.Point_Config__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Point, mysql.Log_Operate_Del, gin.H{"ids": req.Ids, "count": len(req.Ids)})

	// A 方案：删除只作用于数据库与点位树，不热更驱动组包，
	// 避免与 polling 并发读写 Read_Packet_S7DataItem 触发越界崩溃；
	// 被删点位在下次服务重启（InitializeDrivers 从库里读最新点位重建）后彻底停止采集。
	// 点位删除后同步重建树，清理残留的空文件夹与节点
	refreshTagTree()

	ctx.Set("Response", []any{200, "ok"})
}

// api_point_config_count 查询点位配置数量
func api_point_config_count(ctx *gin.Context) {
	var req struct {
		DriveId  []uint `json:"driveId"`
		Page     uint   `json:"page"`
		PageSize uint   `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := mysql.Point_Config__Count(req.DriveId, req.Page, req.PageSize, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// api_point_config_search 搜索点位配置（Tag 模糊 + Drive_Id 过滤）
func api_point_config_search(ctx *gin.Context) {
	var req struct {
		Tag      string `json:"tag"`
		DriveIds []uint `json:"driveIds"`
		Quantity uint   `json:"quantity"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	configs, err := mysql.Point_Config__Search(req.Tag, req.DriveIds, req.Quantity, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// ======== 报警配置 API ========

// alarmConfigWriteReq 报警配置新增/更新入参：在原字段之外可选携带 Point_Tag（点位标识）
// Alarm_Config.Tag 存的是「报警规则名」，跟点位标识不是一回事，所以点位标识只能另起一个字段
// 字段名沿用表里的 PascalCase 写法：{"Id":1, "Point_Tag":"//0001//AI01", "Tag":"高温报警", ...}
type alarmConfigWriteReq struct {
	Id        uint
	Point_Id  uint
	Point_Tag string // 点位标识：非空时按它反查 Point_Id（优先于 Point_Id）
	Tag       string // 报警规则名
	Config    string
	Group     int
}

// resolveAlarmConfigPointIds 把报警配置写入请求里的点位标识换算成 Point_Id
func resolveAlarmConfigPointIds(req []alarmConfigWriteReq) (ids []uint, err error) {
	items := make([]pointLocator, 0, len(req))
	for _, r := range req {
		items = append(items, pointLocator{Tag: r.Point_Tag, Id: r.Point_Id})
	}
	return resolvePointIds(items)
}

// api_alarm_config_query 查询报警配置
// 点位过滤可用 pointTag（点位标识列表，推荐）或 pointId（点位id列表，兼容旧写法），两者可混用
// 报警规则名的模糊搜索走 /config/alarm/search，与这里的点位过滤无关
func api_alarm_config_query(ctx *gin.Context) {
	var req struct {
		PointTag []string `json:"pointTag"`
		PointId  []uint   `json:"pointId"`
		Page     uint     `json:"page"`
		PageSize uint     `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	ids, missing, err := mergePointIds(req.PointTag, req.PointId)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位Tag失败: " + err.Error()})
		return
	}
	if len(missing) > 0 {
		ctx.Set("Response", []any{404, tagResolveError(missing)})
		return
	}

	configs, err := mysql.Alarm_Config__Query(ids, req.Page, req.PageSize, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// api_alarm_config_add 新增报警配置
func api_alarm_config_add(ctx *gin.Context) {
	var req []alarmConfigWriteReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 点位标识优先：按 Point_Tag 反查 Point_Id，没给的沿用 Point_Id
	ids, err := resolveAlarmConfigPointIds(req)
	if err != nil {
		ctx.Set("Response", []any{417, err.Error()})
		return
	}

	configs := make([]mysql.Alarm_Config_Add_type, 0, len(req))
	for i, r := range req {
		configs = append(configs, mysql.Alarm_Config_Add_type{
			Id:       r.Id,
			Point_Id: ids[i],
			Tag:      r.Tag,
			Config:   r.Config,
			Group:    r.Group,
		})
	}

	if err := mysql.Alarm_Config__Add(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Alarm, mysql.Log_Operate_Add, configs)
	collector.Alarm_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_alarm_config_update 更新报警配置
func api_alarm_config_update(ctx *gin.Context) {
	var req []alarmConfigWriteReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	ids, err := resolveAlarmConfigPointIds(req)
	if err != nil {
		ctx.Set("Response", []any{417, err.Error()})
		return
	}

	configs := make([]mysql.Alarm_Config_Update_type, 0, len(req))
	for i, r := range req {
		configs = append(configs, mysql.Alarm_Config_Update_type{
			Id:       r.Id,
			Point_Id: ids[i],
			Tag:      r.Tag,
			Config:   r.Config,
			Group:    r.Group,
		})
	}

	// RBAC 实例级：仅保留有权限的行
	if subset, present := opIdsFromCtx(ctx); present {
		set := idSet(subset)
		kept := configs[:0]
		for _, c := range configs {
			if set[c.Id] {
				kept = append(kept, c)
			}
		}
		configs = kept
	}

	if err := mysql.Alarm_Config__Update(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Alarm, mysql.Log_Operate_Update, configs)
	collector.Alarm_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_alarm_config_del 删除报警配置
func api_alarm_config_del(ctx *gin.Context) {
	var req struct {
		Ids []uint `json:"ids"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// RBAC 实例级：只删有权限的 id
	if subset, present := opIdsFromCtx(ctx); present {
		req.Ids = subset
	}

	if err := mysql.Alarm_Config__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Alarm, mysql.Log_Operate_Del, gin.H{"ids": req.Ids, "count": len(req.Ids)})
	collector.Alarm_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_alarm_config_count 查询报警配置数量（点位过滤同 query：pointTag 优先、pointId 兼容）
func api_alarm_config_count(ctx *gin.Context) {
	var req struct {
		PointTag []string `json:"pointTag"`
		PointId  []uint   `json:"pointId"`
		Page     uint     `json:"page"`
		PageSize uint     `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	ids, missing, err := mergePointIds(req.PointTag, req.PointId)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位Tag失败: " + err.Error()})
		return
	}
	if len(missing) > 0 {
		ctx.Set("Response", []any{404, tagResolveError(missing)})
		return
	}

	count, err := mysql.Alarm_Config__Count(ids, req.Page, req.PageSize, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// api_alarm_config_search 搜索报警配置（报警规则名模糊 + 点位过滤 + Group 过滤）
// tag 对的是「报警规则名」（Alarm_Config.Tag 本意），点位过滤用 pointTag（点位标识，推荐）或 pointIds（兼容旧写法）
func api_alarm_config_search(ctx *gin.Context) {
	var req struct {
		Tag      string   `json:"tag"`
		PointTag []string `json:"pointTag"`
		PointIds []uint   `json:"pointIds"`
		Groups   []int    `json:"groups"`
		Quantity uint     `json:"quantity"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	pointIds, missing, err := mergePointIds(req.PointTag, req.PointIds)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位Tag失败: " + err.Error()})
		return
	}
	if len(missing) > 0 {
		ctx.Set("Response", []any{404, tagResolveError(missing)})
		return
	}

	configs, err := mysql.Alarm_Config__Search(req.Tag, pointIds, req.Groups, req.Quantity, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// ======== 报警数据 API ========

// api_alarm_data_query 查询报警数据
func api_alarm_data_query(ctx *gin.Context) {
	var req struct {
		AlarmId  []uint `json:"alarmId"`
		Page     uint   `json:"page"`
		PageSize uint   `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	data, err := mysql.Alarm_Data__Query(req.AlarmId, req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", data})
}

// api_alarm_data_count 查询报警数据数量
func api_alarm_data_count(ctx *gin.Context) {
	var req struct {
		AlarmId  []uint `json:"alarmId"`
		Page     uint   `json:"page"`
		PageSize uint   `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := mysql.Alarm_Data__Count(req.AlarmId, req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// ======== 历史配置 API ==========

// historyConfigWriteReq 历史配置新增/更新入参：History_Config 表只有 Point_Id，
// 点位标识 Tag 仅用于反查 Point_Id（不落库，不改表结构）；Tag 非空时优先于 Point_Id
type historyConfigWriteReq struct {
	Id       uint
	Point_Id uint
	Tag      string
	Config   string
}

// historyConfigRow 历史配置查询响应：表里没有 Tag 列，按 Point_Id 补出点位标识一并返回，
// 前端展示点位、跳转历史数据都可直接用 Tag，无需为它加列
// （嵌入的 History_Config_Query_type 字段会平铺到同一层，响应体结构不变，只多一个 Tag）
type historyConfigRow struct {
	mysql.History_Config_Query_type
	Tag string // 点位标识（按 Point_Id 反查补全）
}

// api_history_config_query 查询历史配置
// 点位过滤可用 tag（点位标识列表，推荐）或 pointId（点位id列表，兼容旧写法），两者可混用
func api_history_config_query(ctx *gin.Context) {
	var req struct {
		Tag      []string `json:"tag"`
		PointId  []uint   `json:"pointId"`
		Page     uint     `json:"page"`
		PageSize uint     `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	ids, missing, err := mergePointIds(req.Tag, req.PointId)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位Tag失败: " + err.Error()})
		return
	}
	if len(missing) > 0 {
		ctx.Set("Response", []any{404, tagResolveError(missing)})
		return
	}

	configs, err := mysql.History_Config__Query(ids, req.Page, req.PageSize, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 点位标识属于附加字段：反查失败只记 WARN、Tag 留空，不把正常的列表查询一并拒掉
	pointIds := make([]uint, 0, len(configs))
	for _, cfg := range configs {
		pointIds = append(pointIds, cfg.Point_Id)
	}
	tagMap := tagsByIdsSoft(pointIds)

	rows := make([]historyConfigRow, 0, len(configs))
	for _, cfg := range configs {
		rows = append(rows, historyConfigRow{
			History_Config_Query_type: cfg,
			Tag:                       tagMap[cfg.Point_Id],
		})
	}
	ctx.Set("Response", []any{200, "ok", rows})
}

// api_history_config_add 新增历史配置
func api_history_config_add(ctx *gin.Context) {
	var req []historyConfigWriteReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 点位标识优先：按 Tag 反查 Point_Id，没给的沿用 Point_Id
	items := make([]pointLocator, 0, len(req))
	for _, r := range req {
		items = append(items, pointLocator{Tag: r.Tag, Id: r.Point_Id})
	}
	ids, err := resolvePointIds(items)
	if err != nil {
		ctx.Set("Response", []any{417, err.Error()})
		return
	}

	configs := make([]mysql.History_Config_Add_type, 0, len(req))
	for i, r := range req {
		configs = append(configs, mysql.History_Config_Add_type{
			Id:       r.Id,
			Point_Id: ids[i],
			Config:   r.Config,
		})
	}

	if err := mysql.History_Config__Add(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_History, mysql.Log_Operate_Add, configs)
	collector.History_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_history_config_update 更新历史配置
func api_history_config_update(ctx *gin.Context) {
	var req []historyConfigWriteReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	items := make([]pointLocator, 0, len(req))
	for _, r := range req {
		items = append(items, pointLocator{Tag: r.Tag, Id: r.Point_Id})
	}
	ids, err := resolvePointIds(items)
	if err != nil {
		ctx.Set("Response", []any{417, err.Error()})
		return
	}

	configs := make([]mysql.History_Config_Update_type, 0, len(req))
	for i, r := range req {
		configs = append(configs, mysql.History_Config_Update_type{
			Id:       r.Id,
			Point_Id: ids[i],
			Config:   r.Config,
		})
	}

	// RBAC 实例级：仅保留有权限的行
	if subset, present := opIdsFromCtx(ctx); present {
		set := idSet(subset)
		kept := configs[:0]
		for _, c := range configs {
			if set[c.Id] {
				kept = append(kept, c)
			}
		}
		configs = kept
	}

	if err := mysql.History_Config__Update(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_History, mysql.Log_Operate_Update, configs)
	collector.History_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_history_config_del 删除历史配置
func api_history_config_del(ctx *gin.Context) {
	var req struct {
		Ids []uint `json:"ids"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// RBAC 实例级：只删有权限的 id
	if subset, present := opIdsFromCtx(ctx); present {
		req.Ids = subset
	}

	if err := mysql.History_Config__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_History, mysql.Log_Operate_Del, gin.H{"ids": req.Ids, "count": len(req.Ids)})
	collector.History_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_history_config_count 查询历史配置数量（点位过滤同 query：tag 优先、pointId 兼容）
func api_history_config_count(ctx *gin.Context) {
	var req struct {
		Tag      []string `json:"tag"`
		PointId  []uint   `json:"pointId"`
		Page     uint     `json:"page"`
		PageSize uint     `json:"pageSize"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	ids, missing, err := mergePointIds(req.Tag, req.PointId)
	if err != nil {
		ctx.Set("Response", []any{500, "查询点位Tag失败: " + err.Error()})
		return
	}
	if len(missing) > 0 {
		ctx.Set("Response", []any{404, tagResolveError(missing)})
		return
	}

	count, err := mysql.History_Config__Count(ids, req.Page, req.PageSize, scopeIdsFromCtx(ctx))
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// config_api_register 注册配置管理路由
func config_api_register(r *gin.Engine) {
	// 驱动配置
	r.POST("/api/v1.0/config/drive/query", api_drive_config_query)
	r.POST("/api/v1.0/config/drive/count", api_drive_config_count)
	r.POST("/api/v1.0/config/drive/add", api_drive_config_add)
	r.POST("/api/v1.0/config/drive/update", api_drive_config_update)
	r.POST("/api/v1.0/config/drive/del", api_drive_config_del)
	r.POST("/api/v1.0/config/drive/restart", api_drive_config_restart)
	r.POST("/api/v1.0/config/drive/search", api_drive_config_search)

	// 点位配置
	r.POST("/api/v1.0/config/point/query", api_point_config_query)
	r.POST("/api/v1.0/config/point/count", api_point_config_count)
	r.POST("/api/v1.0/config/point/add", api_point_config_add)
	r.POST("/api/v1.0/config/point/update", api_point_config_update)
	r.POST("/api/v1.0/config/point/del", api_point_config_del)
	r.POST("/api/v1.0/config/point/search", api_point_config_search)

	// 报警配置
	r.POST("/api/v1.0/config/alarm/query", api_alarm_config_query)
	r.POST("/api/v1.0/config/alarm/count", api_alarm_config_count)
	r.POST("/api/v1.0/config/alarm/add", api_alarm_config_add)
	r.POST("/api/v1.0/config/alarm/update", api_alarm_config_update)
	r.POST("/api/v1.0/config/alarm/del", api_alarm_config_del)
	r.POST("/api/v1.0/config/alarm/search", api_alarm_config_search)

	// 报警数据
	r.POST("/api/v1.0/config/alarm-data/query", api_alarm_data_query)
	r.POST("/api/v1.0/config/alarm-data/count", api_alarm_data_count)

	// 历史配置
	r.POST("/api/v1.0/config/history/query", api_history_config_query)
	r.POST("/api/v1.0/config/history/count", api_history_config_count)
	r.POST("/api/v1.0/config/history/add", api_history_config_add)
	r.POST("/api/v1.0/config/history/update", api_history_config_update)
	r.POST("/api/v1.0/config/history/del", api_history_config_del)

	// 导出 CSV
	r.GET("/api/v1.0/config/export", api_config_export)

	// 导入 Excel（以 Id 为准同步：新增/更新/删除）
	r.POST("/api/v1.0/config/import", api_config_import)
}
