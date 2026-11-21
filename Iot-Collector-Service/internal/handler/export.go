/*
* 日期: 2026.09.22
* 作者: 范范zwf
* 作用: 配置导出/导入 Excel（Drive_Config、Point_Config、Alarm_Config、History_Config）
*        导出：支持同时导出多种配置，每种配置一个 Sheet
*        导入：以 Id 为准，数据库有但导入没有的删除，导入有但数据库没有的新增，都有的更新
 */
package handler

import (
	"bytes"
	"collector/internal/repo/mysql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// api_config_export 导出配置为 Excel 文件
// GET /api/v1.0/config/export?type=drive,point,alarm,history
// 或 GET /api/v1.0/config/export?type=drive&type=point
func api_config_export(ctx *gin.Context) {
	// 支持数组和逗号分隔两种方式
	types := ctx.QueryArray("type")
	if len(types) == 0 {
		types = strings.Split(ctx.DefaultQuery("type", ""), ",")
	}
	if len(types) == 0 || (len(types) == 1 && types[0] == "") {
		ctx.Set("Response", []any{417, "缺少 type 参数（drive/point/alarm/history）"})
		return
	}

	f := excelize.NewFile()
	// 删除默认空 Sheet
	f.DeleteSheet("Sheet1")

	for _, t := range types {
		t = strings.TrimSpace(t)
		var headers []string
		var rows [][]string
		var err error

		switch t {
		case "drive":
			headers, rows, err = exportDriveConfig()
		case "point":
			headers, rows, err = exportPointConfig()
		case "alarm":
			headers, rows, err = exportAlarmConfig()
		case "history":
			headers, rows, err = exportHistoryConfig()
		default:
			ctx.Set("Response", []any{417, "不支持的 type：" + t})
			return
		}

		if err != nil {
			ctx.Set("Response", []any{500, err.Error()})
			return
		}
		if ct := configTypeByName(t); ct != "" {
			logOperate(ctx, ct, mysql.Log_Operate_Export, gin.H{"action": "export", "type": t, "rows": len(rows)})
		}

		// 创建 Sheet
		sheetName := t + "_config"
		sheetIdx, _ := f.NewSheet(sheetName)
		f.SetActiveSheet(sheetIdx)

		// 写表头
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, h)
		}
		// 写数据行
		for r, row := range rows {
			for c, val := range row {
				cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
				f.SetCellValue(sheetName, cell, val)
			}
		}
	}

	// 设置响应头
	filename := fmt.Sprintf("config_export_%s.xlsx", time.Now().Format("20060102_150405"))
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// 写入响应
	if err := f.Write(ctx.Writer); err != nil {
		return
	}
}

// exportDriveConfig 导出驱动配置
func exportDriveConfig() ([]string, [][]string, error) {
	configs, err := mysql.Drive_Config__Query(0, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("查询驱动配置失败: %w", err)
	}

	headers := []string{"Id", "Type", "Name", "Config", "Points_Length", "Creation_Time"}
	var rows [][]string
	for _, c := range configs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.Id),
			c.Type,
			c.Name,
			c.Config,
			fmt.Sprintf("%d", c.Points_Length),
			c.Creation_Time.Format("2006-01-02 15:04:05"),
		})
	}
	return headers, rows, nil
}

// exportPointConfig 导出点位配置
func exportPointConfig() ([]string, [][]string, error) {
	configs, err := mysql.Point_Config__Query(nil, 0, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("查询点位配置失败: %w", err)
	}

	headers := []string{"Id", "Drive_Id", "Tag", "Description", "Config", "Conversion", "RW_Cancel", "Value_Type", "Creation_Time"}
	var rows [][]string
	for _, c := range configs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.Id),
			fmt.Sprintf("%d", c.Drive_Id),
			c.Tag,
			c.Description,
			c.Config,
			c.Conversion,
			fmt.Sprintf("%d", c.RW_Cancel),
			fmt.Sprintf("%d", c.Value_Type),
			c.Creation_Time.Format("2006-01-02 15:04:05"),
		})
	}
	return headers, rows, nil
}

// exportAlarmConfig 导出报警配置
// 除了表里的 Point_Id，额外导出一列 Point_Tag（点位标识）：
// 报警配置自身的 Tag 列存的是「报警规则名」，与点位标识不是一回事，因此点位标识单独占一列，
// 导入时按 Point_Tag 反查 Point_Id，跨设备迁移模板不必再对 id
func exportAlarmConfig() ([]string, [][]string, error) {
	configs, err := mysql.Alarm_Config__Query(nil, 0, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("查询报警配置失败: %w", err)
	}

	// 点位标识属于附加列：反查失败只记 WARN、该列留空，不阻断导出
	pointIds := make([]uint, 0, len(configs))
	for _, c := range configs {
		pointIds = append(pointIds, c.Point_Id)
	}
	tagMap := tagsByIdsSoft(pointIds)

	headers := []string{"Id", "Point_Id", "Point_Tag", "Tag", "Config", "Group", "Creation_Time"}
	var rows [][]string
	for _, c := range configs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.Id),
			fmt.Sprintf("%d", c.Point_Id),
			tagMap[c.Point_Id], // 点位标识（按 Point_Id 反查）
			c.Tag,              // 报警规则名
			c.Config,
			fmt.Sprintf("%d", c.Group),
			c.Creation_Time.Format("2006-01-02 15:04:05"),
		})
	}
	return headers, rows, nil
}

// exportHistoryConfig 导出历史配置
// History_Config 表只有 Point_Id，这里按 id 补出一列 Tag（点位标识），
// 导入时可直接按 Tag 定位点位，不必为它改表结构
func exportHistoryConfig() ([]string, [][]string, error) {
	configs, err := mysql.History_Config__Query(nil, 0, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("查询历史配置失败: %w", err)
	}

	pointIds := make([]uint, 0, len(configs))
	for _, c := range configs {
		pointIds = append(pointIds, c.Point_Id)
	}
	tagMap := tagsByIdsSoft(pointIds)

	headers := []string{"Id", "Point_Id", "Tag", "Config", "Creation_Time"}
	var rows [][]string
	for _, c := range configs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.Id),
			fmt.Sprintf("%d", c.Point_Id),
			tagMap[c.Point_Id], // 点位标识（按 Point_Id 反查）
			c.Config,
			c.Creation_Time.Format("2006-01-02 15:04:05"),
		})
	}
	return headers, rows, nil
}

// ======== 配置导入 ========

// importResult 导入结果统计
type importResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}

// api_config_import 从 Excel 导入配置（以 Id 为准同步：新增/更新/删除）
// POST /api/v1.0/config/import
// Content-Type: multipart/form-data, Field: "file"
func api_config_import(ctx *gin.Context) {
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.Set("Response", []any{417, "请上传 Excel 文件（字段名: file）"})
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		ctx.Set("Response", []any{417, "无法读取 Excel 文件: " + err.Error()})
		return
	}
	defer f.Close()

	results, err := runConfigImportSheets(f)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	if len(results) == 0 {
		ctx.Set("Response", []any{417, "Excel 中没有找到可导入的 Sheet（支持: drive_config, point_config, alarm_config, history_config）"})
		return
	}

	// 导入属于批量覆写，按 Sheet 各记一条“导入”日志（只记影响条数，不复制全量数据）
	for name, res := range results {
		if res == nil {
			continue
		}
		if ct := configTypeByName(strings.TrimSuffix(name, "_config")); ct != "" {
			logOperate(ctx, ct, mysql.Log_Operate_Import, gin.H{
				"action": "import", "sheet": name,
				"added": res.Added, "updated": res.Updated, "deleted": res.Deleted,
			})
		}
	}

	// 若本次导入了 point_config（有增/改/删），同步重建点位树，保持层级与点位一致
	if res := results["point_config"]; res != nil && (res.Added != 0 || res.Updated != 0 || res.Deleted != 0) {
		refreshTagTree()
	}

	ctx.Set("Response", []any{200, "导入成功", results})
}

// ImportResult 单个 Sheet 的导入统计（对外导出，供内部 MQTT 等复用）
type ImportResult struct {
	Sheet   string `json:"sheet"`
	Added   int    `json:"added"`
	Updated int    `json:"updated"`
	Deleted int    `json:"deleted"`
}

// ImportConfigBytes 从 Excel(xlsx) 字节流全量导入配置（以 Id 为准同步：新增/更新/删除）。
// 供内部 MQTT “更新所有配置”从 http 地址下载表格后复用；返回每个 Sheet 的导入统计。
func ImportConfigBytes(data []byte) ([]ImportResult, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("无法读取 Excel 数据: %w", err)
	}
	defer f.Close()

	results, err := runConfigImportSheets(f)
	if err != nil {
		return nil, err
	}

	list := make([]ImportResult, 0, len(results))
	for name, res := range results {
		if res == nil {
			continue
		}
		list = append(list, ImportResult{Sheet: name, Added: res.Added, Updated: res.Updated, Deleted: res.Deleted})
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("Excel 中没有找到可导入的 Sheet（支持: drive_config, point_config, alarm_config, history_config）")
	}
	return list, nil
}

// runConfigImportSheets 按固定顺序处理 Excel 中存在的配置 Sheet，返回各 Sheet 的导入结果。
// 供 HTTP 上传导入与内部 MQTT 下载导入共用，不含日志/响应处理。
func runConfigImportSheets(f *excelize.File) (map[string]*importResult, error) {
	results := make(map[string]*importResult)

	// 按顺序处理各 Sheet
	sheetHandlers := []struct {
		name    string
		handler func(*excelize.File) (*importResult, error)
	}{
		{"drive_config", importDriveConfig},
		{"point_config", importPointConfig},
		{"alarm_config", importAlarmConfig},
		{"history_config", importHistoryConfig},
	}

	// 获取所有 Sheet 名
	sheetList := f.GetSheetList()
	sheetSet := make(map[string]bool, len(sheetList))
	for _, s := range sheetList {
		sheetSet[s] = true
	}

	for _, sh := range sheetHandlers {
		if !sheetSet[sh.name] {
			continue
		}
		result, err := sh.handler(f)
		if err != nil {
			return nil, fmt.Errorf("导入 %s 失败: %s", sh.name, err.Error())
		}
		results[sh.name] = result
	}
	return results, nil
}

// configTypeByName 配置对象短名（drive/point/alarm/history）→ 日志 Type；未知返回空串（不记日志）
func configTypeByName(name string) mysql.Log_Type {
	switch name {
	case "drive":
		return mysql.Log_Type_Drive
	case "point":
		return mysql.Log_Type_Point
	case "alarm":
		return mysql.Log_Type_Alarm
	case "history":
		return mysql.Log_Type_History
	default:
		return ""
	}
}

// readSheetRows 读取 Excel Sheet 的表头和行数据
func readSheetRows(f *excelize.File, sheet string) ([]string, [][]string, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("读取 Sheet %s 失败: %w", sheet, err)
	}
	if len(rows) < 2 {
		return nil, nil, nil // 只有表头或空 Sheet
	}
	return rows[0], rows[1:], nil
}

// colIndex 根据表头找到列索引（-1 表示不存在）
func colIndex(headers []string, name string) int {
	for i, h := range headers {
		if strings.TrimSpace(h) == name {
			return i
		}
	}
	return -1
}

// getCell 安全获取单元格值
func getCell(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

// getCellUint 安全获取单元格 uint 值
func getCellUint(row []string, idx int) (uint, error) {
	s := getCell(row, idx)
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析 '%s' 为 uint 失败: %w", s, err)
	}
	return uint(v), nil
}

// getCellInt 安全获取单元格 int 值
func getCellInt(row []string, idx int) (int, error) {
	s := getCell(row, idx)
	if s == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("解析 '%s' 为 int 失败: %w", s, err)
	}
	return v, nil
}

// importDriveConfig 导入驱动配置（以 Id 为准同步）
func importDriveConfig(f *excelize.File) (*importResult, error) {
	headers, dataRows, err := readSheetRows(f, "drive_config")
	if err != nil {
		return nil, err
	}
	if len(dataRows) == 0 {
		return &importResult{}, nil
	}

	idIdx := colIndex(headers, "Id")
	typeIdx := colIndex(headers, "Type")
	nameIdx := colIndex(headers, "Name")
	configIdx := colIndex(headers, "Config")
	if idIdx < 0 || typeIdx < 0 || configIdx < 0 {
		return nil, fmt.Errorf("缺少必要列: Id, Type, Config")
	}

	// 解析导入数据
	type row struct {
		Id     uint
		Type   string
		Name   string
		Config string
	}
	var incoming []row
	for i, r := range dataRows {
		id, err := getCellUint(r, idIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行: %w", i+2, err)
		}
		if id == 0 {
			continue
		}
		incoming = append(incoming, row{
			Id:     id,
			Type:   getCell(r, typeIdx),
			Name:   getCell(r, nameIdx),
			Config: getCell(r, configIdx),
		})
	}

	// 读取现有数据
	existing, err := mysql.Drive_Config__Query(0, 0)
	if err != nil {
		return nil, fmt.Errorf("读取现有驱动配置失败: %w", err)
	}
	existingMap := make(map[uint]mysql.Drive_Config_Query_type, len(existing))
	for _, cfg := range existing {
		existingMap[cfg.Id] = cfg
	}

	// 比对
	incomingIds := make(map[uint]bool, len(incoming))
	var toAdd []row
	var toUpdate []row
	for _, r := range incoming {
		incomingIds[r.Id] = true
		if ex, ok := existingMap[r.Id]; !ok {
			toAdd = append(toAdd, r)
		} else if r.Type != ex.Type || r.Name != ex.Name || r.Config != ex.Config {
			toUpdate = append(toUpdate, r)
		}
	}
	var toDel []uint
	for _, cfg := range existing {
		if !incomingIds[cfg.Id] {
			toDel = append(toDel, cfg.Id)
		}
	}

	// 执行（事务内保留 Id）
	result := &importResult{}
	tx, err := mysql.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if result == nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()
	for _, r := range toAdd {
		_, err = tx.Exec(
			"INSERT INTO `Drive_Config`(`Id`,`Type`,`Name`,`Config`,`Points_Length`,`Creation_Time`) VALUES (?,?,?,?,?,?)",
			r.Id, r.Type, r.Name, r.Config, 0, now,
		)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("新增 Id=%d 失败: %w", r.Id, err)
		}
		result.Added++
	}
	for _, r := range toUpdate {
		_, err = tx.Exec(
			"UPDATE `Drive_Config` SET `Type`=?,`Name`=?,`Config`=? WHERE `Id`=?",
			r.Type, r.Name, r.Config, r.Id,
		)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("更新 Id=%d 失败: %w", r.Id, err)
		}
		result.Updated++
	}
	for _, id := range toDel {
		_, err = tx.Exec("DELETE FROM `Drive_Config` WHERE `Id`=?", id)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("删除 Id=%d 失败: %w", id, err)
		}
		result.Deleted++
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	log.Printf("导入驱动配置：新增 %d，更新 %d，删除 %d", result.Added, result.Updated, result.Deleted)
	return result, nil
}

// importPointConfig 导入点位配置（以 Id 为准同步）
func importPointConfig(f *excelize.File) (*importResult, error) {
	headers, dataRows, err := readSheetRows(f, "point_config")
	if err != nil {
		return nil, err
	}
	if len(dataRows) == 0 {
		return &importResult{}, nil
	}

	idIdx := colIndex(headers, "Id")
	driveIdx := colIndex(headers, "Drive_Id")
	tagIdx := colIndex(headers, "Tag")
	descIdx := colIndex(headers, "Description")
	configIdx := colIndex(headers, "Config")
	conversionIdx := colIndex(headers, "Conversion")
	rwIdx := colIndex(headers, "RW_Cancel")
	vtIdx := colIndex(headers, "Value_Type")
	if idIdx < 0 || driveIdx < 0 || configIdx < 0 || rwIdx < 0 || vtIdx < 0 {
		return nil, fmt.Errorf("缺少必要列: Id, Drive_Id, Config, RW_Cancel, Value_Type")
	}

	type row struct {
		Id          uint
		Drive_Id    uint
		Tag         string
		Description string
		Config      string
		Conversion  string
		RW_Cancel   int
		Value_Type  int
	}
	var incoming []row
	for i, r := range dataRows {
		id, err := getCellUint(r, idIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行: %w", i+2, err)
		}
		if id == 0 {
			continue
		}
		driveId, err := getCellUint(r, driveIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行 Drive_Id: %w", i+2, err)
		}
		rw, err := getCellInt(r, rwIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行 RW_Cancel: %w", i+2, err)
		}
		vt, err := getCellInt(r, vtIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行 Value_Type: %w", i+2, err)
		}
		incoming = append(incoming, row{
			Id:          id,
			Drive_Id:    driveId,
			Tag:         getCell(r, tagIdx),
			Description: getCell(r, descIdx),
			Config:      getCell(r, configIdx),
			Conversion:  getCell(r, conversionIdx),
			RW_Cancel:   rw,
			Value_Type:  vt,
		})
	}

	existing, err := mysql.Point_Config__Query(nil, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("读取现有点位配置失败: %w", err)
	}
	existingMap := make(map[uint]mysql.Point_Config_Query_type, len(existing))
	for _, cfg := range existing {
		existingMap[cfg.Id] = cfg
	}

	incomingIds := make(map[uint]bool, len(incoming))
	var toAdd []row
	var toUpdate []row
	for _, r := range incoming {
		incomingIds[r.Id] = true
		if ex, ok := existingMap[r.Id]; !ok {
			toAdd = append(toAdd, r)
		} else if r.Drive_Id != ex.Drive_Id || r.Tag != ex.Tag || r.Description != ex.Description || r.Config != ex.Config || r.Conversion != ex.Conversion || r.RW_Cancel != ex.RW_Cancel || r.Value_Type != ex.Value_Type {
			toUpdate = append(toUpdate, r)
		}
	}
	var toDel []uint
	for _, cfg := range existing {
		if !incomingIds[cfg.Id] {
			toDel = append(toDel, cfg.Id)
		}
	}

	result := &importResult{}
	tx, err := mysql.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if result == nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()
	for _, r := range toAdd {
		_, err = tx.Exec(
			"INSERT INTO `Point_Config`(`Id`,`Drive_Id`,`Tag`,`Description`,`Config`,`Conversion`,`RW_Cancel`,`Value_Type`,`Creation_Time`) VALUES (?,?,?,?,?,?,?,?,?)",
			r.Id, r.Drive_Id, r.Tag, r.Description, r.Config, r.Conversion, r.RW_Cancel, r.Value_Type, now,
		)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("新增 Id=%d 失败: %w", r.Id, err)
		}
		result.Added++
	}
	for _, r := range toUpdate {
		_, err = tx.Exec(
			"UPDATE `Point_Config` SET `Drive_Id`=?,`Tag`=?,`Description`=?,`Config`=?,`Conversion`=?,`RW_Cancel`=?,`Value_Type`=? WHERE `Id`=?",
			r.Drive_Id, r.Tag, r.Description, r.Config, r.Conversion, r.RW_Cancel, r.Value_Type, r.Id,
		)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("更新 Id=%d 失败: %w", r.Id, err)
		}
		result.Updated++
	}
	for _, id := range toDel {
		_, err = tx.Exec("DELETE FROM `Point_Config` WHERE `Id`=?", id)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("删除 Id=%d 失败: %w", id, err)
		}
		result.Deleted++
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	log.Printf("导入点位配置：新增 %d，更新 %d，删除 %d", result.Added, result.Updated, result.Deleted)
	return result, nil
}

// importAlarmConfig 导入报警配置（以 Id 为准同步）
func importAlarmConfig(f *excelize.File) (*importResult, error) {
	headers, dataRows, err := readSheetRows(f, "alarm_config")
	if err != nil {
		return nil, err
	}
	if len(dataRows) == 0 {
		return &importResult{}, nil
	}

	idIdx := colIndex(headers, "Id")
	pointIdx := colIndex(headers, "Point_Id")
	pointTagIdx := colIndex(headers, "Point_Tag")
	ruleTagIdx := colIndex(headers, "Tag")
	if ruleTagIdx < 0 {
		ruleTagIdx = colIndex(headers, "Name") // 兼容字段改名前的旧模板
	}
	configIdx := colIndex(headers, "Config")
	groupIdx := colIndex(headers, "Group")
	if idIdx < 0 || configIdx < 0 || groupIdx < 0 {
		return nil, fmt.Errorf("缺少必要列: Id, Config, Group")
	}
	// 点位定位列：Point_Tag（点位标识，推荐）与 Point_Id 至少有一个
	if pointIdx < 0 && pointTagIdx < 0 {
		return nil, fmt.Errorf("缺少必要列: Id, Point_Tag(或 Point_Id), Config, Group")
	}

	type row struct {
		Id        uint
		Point_Id  uint
		Point_Tag string // 点位标识：非空时优先用它定位点位
		Tag       string // 报警规则名
		Config    string
		Group     int
	}
	var incoming []row
	for i, r := range dataRows {
		id, err := getCellUint(r, idIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行: %w", i+2, err)
		}
		if id == 0 {
			continue
		}
		pointId, err := getCellUint(r, pointIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行 Point_Id: %w", i+2, err)
		}
		group, err := getCellInt(r, groupIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行 Group: %w", i+2, err)
		}
		pointTag := getCell(r, pointTagIdx)
		if pointId == 0 && pointTag == "" {
			return nil, fmt.Errorf("第 %d 行：Point_Tag 与 Point_Id 至少有一个", i+2)
		}
		incoming = append(incoming, row{
			Id:        id,
			Point_Id:  pointId,
			Point_Tag: pointTag,
			Tag:       getCell(r, ruleTagIdx),
			Config:    getCell(r, configIdx),
			Group:     group,
		})
	}

	// 点位标识优先：按 Point_Tag 反查 Point_Id（旧模板没这列时 getCell 返回空串，直接用 Point_Id）
	items := make([]pointLocator, 0, len(incoming))
	for _, r := range incoming {
		items = append(items, pointLocator{Tag: r.Point_Tag, Id: r.Point_Id})
	}
	ids, err := resolvePointIds(items)
	if err != nil {
		return nil, fmt.Errorf("报警配置点位定位失败: %w", err)
	}
	for i := range incoming {
		incoming[i].Point_Id = ids[i]
	}

	existing, err := mysql.Alarm_Config__Query(nil, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("读取现有报警配置失败: %w", err)
	}
	existingMap := make(map[uint]mysql.Alarm_Config_Query_type, len(existing))
	for _, cfg := range existing {
		existingMap[cfg.Id] = cfg
	}

	incomingIds := make(map[uint]bool, len(incoming))
	var toAdd []row
	var toUpdate []row
	for _, r := range incoming {
		incomingIds[r.Id] = true
		if ex, ok := existingMap[r.Id]; !ok {
			toAdd = append(toAdd, r)
		} else if r.Point_Id != ex.Point_Id || r.Tag != ex.Tag || r.Config != ex.Config || r.Group != ex.Group {
			toUpdate = append(toUpdate, r)
		}
	}
	var toDel []uint
	for _, cfg := range existing {
		if !incomingIds[cfg.Id] {
			toDel = append(toDel, cfg.Id)
		}
	}

	result := &importResult{}
	tx, err := mysql.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if result == nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()
	for _, r := range toAdd {
		_, err = tx.Exec(
			"INSERT INTO `Alarm_Config`(`Id`,`Point_Id`,`Tag`,`Config`,`Group`,`Creation_Time`) VALUES (?,?,?,?,?,?)",
			r.Id, r.Point_Id, r.Tag, r.Config, r.Group, now,
		)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("新增 Id=%d 失败: %w", r.Id, err)
		}
		result.Added++
	}
	for _, r := range toUpdate {
		_, err = tx.Exec(
			"UPDATE `Alarm_Config` SET `Point_Id`=?,`Tag`=?,`Config`=?,`Group`=? WHERE `Id`=?",
			r.Point_Id, r.Tag, r.Config, r.Group, r.Id,
		)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("更新 Id=%d 失败: %w", r.Id, err)
		}
		result.Updated++
	}
	for _, id := range toDel {
		_, err = tx.Exec("DELETE FROM `Alarm_Config` WHERE `Id`=?", id)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("删除 Id=%d 失败: %w", id, err)
		}
		result.Deleted++
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	log.Printf("导入报警配置：新增 %d，更新 %d，删除 %d", result.Added, result.Updated, result.Deleted)
	return result, nil
}

// importHistoryConfig 导入历史配置（以 Id 为准同步）
func importHistoryConfig(f *excelize.File) (*importResult, error) {
	headers, dataRows, err := readSheetRows(f, "history_config")
	if err != nil {
		return nil, err
	}
	if len(dataRows) == 0 {
		return &importResult{}, nil
	}

	idIdx := colIndex(headers, "Id")
	pointIdx := colIndex(headers, "Point_Id")
	tagIdx := colIndex(headers, "Tag")
	configIdx := colIndex(headers, "Config")
	if idIdx < 0 || configIdx < 0 {
		return nil, fmt.Errorf("缺少必要列: Id, Config")
	}
	// 点位定位列：Tag（点位标识，推荐）与 Point_Id 至少有一个
	if pointIdx < 0 && tagIdx < 0 {
		return nil, fmt.Errorf("缺少必要列: Id, Tag(或 Point_Id), Config")
	}

	type row struct {
		Id       uint
		Point_Id uint
		Tag      string // 点位标识：非空时优先用它定位点位（不落库）
		Config   string
	}
	var incoming []row
	for i, r := range dataRows {
		id, err := getCellUint(r, idIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行: %w", i+2, err)
		}
		if id == 0 {
			continue
		}
		pointId, err := getCellUint(r, pointIdx)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行 Point_Id: %w", i+2, err)
		}
		pointTag := getCell(r, tagIdx)
		if pointId == 0 && pointTag == "" {
			return nil, fmt.Errorf("第 %d 行：Tag 与 Point_Id 至少有一个", i+2)
		}
		incoming = append(incoming, row{
			Id:       id,
			Point_Id: pointId,
			Tag:      pointTag,
			Config:   getCell(r, configIdx),
		})
	}

	// 点位标识优先：按 Tag 反查 Point_Id，表里仍只存 Point_Id
	items := make([]pointLocator, 0, len(incoming))
	for _, r := range incoming {
		items = append(items, pointLocator{Tag: r.Tag, Id: r.Point_Id})
	}
	ids, err := resolvePointIds(items)
	if err != nil {
		return nil, fmt.Errorf("历史配置点位定位失败: %w", err)
	}
	for i := range incoming {
		incoming[i].Point_Id = ids[i]
	}

	existing, err := mysql.History_Config__Query(nil, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("读取现有历史配置失败: %w", err)
	}
	existingMap := make(map[uint]mysql.History_Config_Query_type, len(existing))
	for _, cfg := range existing {
		existingMap[cfg.Id] = cfg
	}

	incomingIds := make(map[uint]bool, len(incoming))
	var toAdd []row
	var toUpdate []row
	for _, r := range incoming {
		incomingIds[r.Id] = true
		if ex, ok := existingMap[r.Id]; !ok {
			toAdd = append(toAdd, r)
		} else if r.Point_Id != ex.Point_Id || r.Config != ex.Config {
			toUpdate = append(toUpdate, r)
		}
	}
	var toDel []uint
	for _, cfg := range existing {
		if !incomingIds[cfg.Id] {
			toDel = append(toDel, cfg.Id)
		}
	}

	result := &importResult{}
	tx, err := mysql.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if result == nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()
	for _, r := range toAdd {
		_, err = tx.Exec(
			"INSERT INTO `History_Config`(`Id`,`Point_Id`,`Config`,`Creation_Time`) VALUES (?,?,?,?)",
			r.Id, r.Point_Id, r.Config, now,
		)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("新增 Id=%d 失败: %w", r.Id, err)
		}
		result.Added++
	}
	for _, r := range toUpdate {
		_, err = tx.Exec(
			"UPDATE `History_Config` SET `Point_Id`=?,`Config`=? WHERE `Id`=?",
			r.Point_Id, r.Config, r.Id,
		)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("更新 Id=%d 失败: %w", r.Id, err)
		}
		result.Updated++
	}
	for _, id := range toDel {
		_, err = tx.Exec("DELETE FROM `History_Config` WHERE `Id`=?", id)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("删除 Id=%d 失败: %w", id, err)
		}
		result.Deleted++
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	log.Printf("导入历史配置：新增 %d，更新 %d，删除 %d", result.Added, result.Updated, result.Deleted)
	return result, nil
}
