/*
* 日期: 2026.09.22
* 作者: 范范zwf
* 作用: 配置导出 Excel（Drive_Config、Point_Config、Alarm_Config、History_Config）
*        支持同时导出多种配置，每种配置一个 Sheet
 */
package web

import (
	"fmt"
	"main/db/mysql"
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

	headers := []string{"Id", "Type", "Config", "Points_Length"}
	var rows [][]string
	for _, c := range configs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.Id),
			c.Type,
			c.Config,
			fmt.Sprintf("%d", c.Points_Length),
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

	headers := []string{"Id", "Drive_Id", "Config", "RW_Cancel", "Value_Type"}
	var rows [][]string
	for _, c := range configs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.Id),
			fmt.Sprintf("%d", c.Drive_Id),
			c.Config,
			fmt.Sprintf("%d", c.RW_Cancel),
			fmt.Sprintf("%d", c.Value_Type),
		})
	}
	return headers, rows, nil
}

// exportAlarmConfig 导出报警配置
func exportAlarmConfig() ([]string, [][]string, error) {
	configs, err := mysql.Alarm_Config__Query(nil, 0, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("查询报警配置失败: %w", err)
	}

	headers := []string{"Id", "Point_Id", "Config", "Group"}
	var rows [][]string
	for _, c := range configs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.Id),
			fmt.Sprintf("%d", c.Point_Id),
			c.Config,
			fmt.Sprintf("%d", c.Group),
		})
	}
	return headers, rows, nil
}

// exportHistoryConfig 导出历史配置
func exportHistoryConfig() ([]string, [][]string, error) {
	configs, err := mysql.History_Config__Query(nil, 0, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("查询历史配置失败: %w", err)
	}

	headers := []string{"Id", "Point_Id", "Config"}
	var rows [][]string
	for _, c := range configs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.Id),
			fmt.Sprintf("%d", c.Point_Id),
			c.Config,
		})
	}
	return headers, rows, nil
}
