/*
* 日期: 2026.09.22
* 作者: 范范zwf
* 作用: 配置管理 API（驱动/点位/报警/历史 的增删改查）
 */
package web

import (
	"log"
	"main/IO/manager"
	"main/db/db_point"
	"main/db/mysql"

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

	configs, err := mysql.Drive_Config__Query(req.Page, req.PageSize)
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
	ctx.Set("Response", []any{200, "ok"})
}

// api_drive_config_update 更新驱动配置
func api_drive_config_update(ctx *gin.Context) {
	var configs []mysql.Drive_Config_Update_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if err := mysql.Drive_Config__Update(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
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

	if err := mysql.Drive_Config__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
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

	count, err := mysql.Drive_Config__Count(req.Page, req.PageSize)
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
	if manager.IsDriverBusy(req.Id) {
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

	// 异步执行重启
	go func() {
		if err := manager.DriveResetConfig(req.Id, driveCfg, points); err != nil {
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

	configs, err := mysql.Drive_Config__Search(req.Name, req.Types, req.Quantity)
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

	configs, err := mysql.Point_Config__Query(req.DriveId, req.Page, req.PageSize)
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
			points, err := mysql.Point_Config__Query([]uint{did}, 0, 0)
			if err != nil {
				log.Printf("ERROR 驱动 id=%d 点位查询失败: %v", did, err)
				return
			}
			if err := manager.DriveUpdatePoints(did, points); err != nil {
				log.Printf("WARN 驱动 id=%d 点位动态更新失败: %v", did, err)
			} else {
				log.Printf("INFO 驱动 id=%d 点位动态更新成功，点位数=%d", did, len(points))
			}
		}(driveId)
	}
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

	// 收集受影响的驱动ID，异步刷新点位
	var driveIds []uint
	for _, c := range configs {
		driveIds = append(driveIds, c.Drive_Id)
	}
	asyncRefreshDrivePoints(driveIds)

	ctx.Set("Response", []any{200, "ok"})
}

// api_point_config_update 更新点位配置
func api_point_config_update(ctx *gin.Context) {
	var configs []mysql.Point_Config_Update_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if err := mysql.Point_Config__Update(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 收集受影响的驱动ID，异步刷新点位
	var driveIds []uint
	for _, c := range configs {
		driveIds = append(driveIds, c.Drive_Id)
	}
	asyncRefreshDrivePoints(driveIds)

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

	// 删除前查询受影响的驱动ID
	allPoints, _ := mysql.Point_Config__Query(nil, 0, 0)
	affectedDriveIds := make(map[uint]bool)
	for _, p := range allPoints {
		for _, id := range req.Ids {
			if p.Id == id {
				affectedDriveIds[p.Drive_Id] = true
			}
		}
	}

	if err := mysql.Point_Config__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 异步刷新受影响的驱动点位
	var driveIds []uint
	for id := range affectedDriveIds {
		driveIds = append(driveIds, id)
	}
	asyncRefreshDrivePoints(driveIds)

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

	count, err := mysql.Point_Config__Count(req.DriveId, req.Page, req.PageSize)
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

	configs, err := mysql.Point_Config__Search(req.Tag, req.DriveIds, req.Quantity)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// ======== 报警配置 API ========

// api_alarm_config_query 查询报警配置
func api_alarm_config_query(ctx *gin.Context) {
	var req struct {
		PointId  []uint `json:"pointId"`
		Page     uint   `json:"page"`
		PageSize uint   `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	configs, err := mysql.Alarm_Config__Query(req.PointId, req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// api_alarm_config_add 新增报警配置
func api_alarm_config_add(ctx *gin.Context) {
	var configs []mysql.Alarm_Config_Add_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if err := mysql.Alarm_Config__Add(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	db_point.Alarm_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_alarm_config_update 更新报警配置
func api_alarm_config_update(ctx *gin.Context) {
	var configs []mysql.Alarm_Config_Update_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if err := mysql.Alarm_Config__Update(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	db_point.Alarm_Config__Reload()
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

	if err := mysql.Alarm_Config__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	db_point.Alarm_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_alarm_config_count 查询报警配置数量
func api_alarm_config_count(ctx *gin.Context) {
	var req struct {
		PointId  []uint `json:"pointId"`
		Page     uint   `json:"page"`
		PageSize uint   `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := mysql.Alarm_Config__Count(req.PointId, req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// api_alarm_config_search 搜索报警配置（Tag 模糊 + Point_Id 过滤 + Group 过滤）
func api_alarm_config_search(ctx *gin.Context) {
	var req struct {
		Tag      string `json:"tag"`
		PointIds []uint `json:"pointIds"`
		Groups   []int  `json:"groups"`
		Quantity uint   `json:"quantity"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	configs, err := mysql.Alarm_Config__Search(req.Tag, req.PointIds, req.Groups, req.Quantity)
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

// api_history_config_query 查询历史配置
func api_history_config_query(ctx *gin.Context) {
	var req struct {
		PointId  []uint `json:"pointId"`
		Page     uint   `json:"page"`
		PageSize uint   `json:"pageSize"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	configs, err := mysql.History_Config__Query(req.PointId, req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", configs})
}

// api_history_config_add 新增历史配置
func api_history_config_add(ctx *gin.Context) {
	var configs []mysql.History_Config_Add_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if err := mysql.History_Config__Add(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	db_point.History_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_history_config_update 更新历史配置
func api_history_config_update(ctx *gin.Context) {
	var configs []mysql.History_Config_Update_type
	if err := ctx.BindJSON(&configs); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if err := mysql.History_Config__Update(configs...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	db_point.History_Config__Reload()
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

	if err := mysql.History_Config__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	db_point.History_Config__Reload()
	ctx.Set("Response", []any{200, "ok"})
}

// api_history_config_count 查询历史配置数量
func api_history_config_count(ctx *gin.Context) {
	var req struct {
		PointId  []uint `json:"pointId"`
		Page     uint   `json:"page"`
		PageSize uint   `json:"pageSize"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := mysql.History_Config__Count(req.PointId, req.Page, req.PageSize)
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
