/*
* 日期: 2025.12.23 16:50
* 作者: 范范zwf
* 作用: api-用户相关
 */
package web

import (
	// "fmt"
	"main/app/mqtt_rpc"
	// "main/app/user_service"
	db_mysql "main/db/mysql"

	"database/sql"

	"github.com/gin-gonic/gin"
)

/*
***************采集配置接口***************
 */

// 采集-》查询数量 传递: page 页码, pageSize 每页数量 返回: Count 数量, err 错误
func Collector_Info__Count(ctx *gin.Context) {
	var jsondata struct {
		Page      uint
		Page_Size uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := db_mysql.Collector_Info__Count(jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if count == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// 采集-》查询配置 传递: page 页码, pageSize 每页数量 返回: configs 配置, err 错误
func Collector_Info__Query(ctx *gin.Context) {
	var jsondata struct {
		Page      uint
		Page_Size uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	config_list, err := db_mysql.Collector_Info__Query(jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if len(config_list) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", config_list})
}

// 采集-》增加配置 传递: config 配置数组形式 返回: err 错误
func Collector_Info__Add(ctx *gin.Context) {
	var jsondata db_mysql.Collector_Info_Add_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Collector_Info__Add(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

// 采集-》更新配置 传递：config 配置数组形式 返回：err 错误
func Collector_Info__Update(ctx *gin.Context) {
	var jsondata db_mysql.Collector_Info_Update_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Collector_Info__Update(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

// 采集-》增加删除 传递: Id 需要删除的id 返回: err 错误
func Collector_Info__Del(ctx *gin.Context) {
	var jsondata struct {
		Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Collector_Info__Del(jsondata.Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

// 采集-》搜索 传递：field quantity 数量，vague 模糊搜索字符串 返回：configs 配置，err 错误
func Collector_Info__Search_Field(ctx *gin.Context) {
	var jsondata struct {
		Field    string
		Quantity uint
		Vague    string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	config_list, err := db_mysql.Collector_Info__Search_Field(jsondata.Field, jsondata.Quantity, jsondata.Vague)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if len(config_list) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", config_list})
}

// 采集-》搜索 传递：field quantity 数量，vague 模糊搜索字符串 返回：configs 配置，err 错误
func Collector_Info__Search_Field_Blurred(ctx *gin.Context) {
	var jsondata struct {
		Quantity uint
		Vague    string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	config_list, err := db_mysql.Collector_Info__Search_Field_Blurred(jsondata.Quantity, jsondata.Vague)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if len(config_list) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", config_list})
}

/*
***************驱动配置接口***************
 */

// 驱动-》查询数量 传递: driveType 驱动类型, page 页码, pageSize 每页数量 返回: Count 数量, err 错误
func Drive_Config__Count(ctx *gin.Context) {
	var jsondata struct {
		Page         uint
		Page_Size    uint
		Collector_Id []uint
		Drive_Type   []string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := db_mysql.Drive_Config__Count(jsondata.Collector_Id, jsondata.Drive_Type, jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if count == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// 驱动-》查询配置 传递: driveType 驱动类型, page 页码, pageSize 每页数量 返回: configs 配置, err 错误
func Drive_Config__Query(ctx *gin.Context) {
	var jsondata struct {
		Page         uint
		Page_Size    uint
		Collector_Id []uint
		Drive_Type   []string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	config_list, err := db_mysql.Drive_Config__Query(jsondata.Collector_Id, jsondata.Drive_Type, jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if len(config_list) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", config_list})
}

// 驱动-》增加配置 传递: config 配置数组形式 返回: err 错误
func Drive_Config__Add(ctx *gin.Context) {
	var jsondata db_mysql.Drive_Config_Add_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Drive_Config__Add(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

// 驱动-》修改配置 传递: config 配置 返回: conid 获取自增的Id, err 错误
func Drive_Config__Update(ctx *gin.Context) {
	var jsondata db_mysql.Drive_Config_Update_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Drive_Config__Update(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

// 驱动-》删除配置 传递: ids 删除的id数组 返回: err 错误
func Drive_Config__Del(ctx *gin.Context) {
	var jsondata struct {
		Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Drive_Config__Del(jsondata.Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})

}

func Drive_Config__Search_Field(ctx *gin.Context) {
	var jsondata struct {
		Field    string
		Quantity uint
		Vague    string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	config_list, err := db_mysql.Drive_Config__Search_Field(jsondata.Field, jsondata.Quantity, jsondata.Vague)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if len(config_list) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", config_list})
}

// 驱动-》搜索 传递：field quantity 数量，vague 模糊搜索字符串 返回：configs 配置，err 错误
func Drive_Config__Search_Field_Blurred(ctx *gin.Context) {
	var jsondata struct {
		Quantity uint
		Vague    string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	config_list, err := db_mysql.Drive_Config__Search_Field_Blurred(jsondata.Quantity, jsondata.Vague)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if len(config_list) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", config_list})
}

/*
***************点位配置接口***************
 */
// 点位-》查询数量 传递: driveid 设备id, page 页码, pageSize 每页数量 返回: Count 数量, err 错误
func Points_Config__Count(ctx *gin.Context) {
	var jsondata struct {
		Page         uint
		Page_Size    uint
		Drive_Id     []uint
		Collector_Id []uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := db_mysql.Points_Config__Count(jsondata.Collector_Id, jsondata.Drive_Id, jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if count == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// 点位-》查询配置 传递: driveid 设备id, page 页码, pageSize 每页数量 返回: configs 配置, err 错误
func Points_Config__Query(ctx *gin.Context) {
	var jsondata struct {
		Page         uint
		Page_Size    uint
		Drive_Id     []uint
		Collector_Id []uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	config_list, err := db_mysql.Points_Config__Query(jsondata.Collector_Id, jsondata.Drive_Id, jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "查询不到"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	} else if len(config_list) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", config_list})
}

// 点位-》增加配置 传递: config 配置数组形式 返回: err 错误
func Points_Config__Add(ctx *gin.Context) {
	var jsondata db_mysql.Points_Config_Add_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Points_Config__Add(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

// 点位-》修改配置 传递: config 配置 返回: conid 获取自增的Id, err 错误
func Points_Config__Update(ctx *gin.Context) {
	var jsondata db_mysql.Points_Config_Update_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Points_Config__Update(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

// 点位-》删除配置 传递: ids 删除的id数组 返回: err 错误
func Points_Config__Del(ctx *gin.Context) {
	var jsondata struct {
		Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Points_Config__Del(jsondata.Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

func App_Restart(ctx *gin.Context) {
	var jsondata struct {
		Uuid string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	err = mqtt_rpc.App_Restart(jsondata.Uuid)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

// 采集服务同步
func Collector_Synchronise_Config(ctx *gin.Context) {
	var jsondata struct {
		Uuid string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	err = mqtt_rpc.Collector_Synchronise_Config(jsondata.Uuid)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}
func gui_api(r *gin.Engine) {
	r.POST("/api/gui/v1.0/collector/count", Collector_Info__Count)
	r.POST("/api/gui/v1.0/collector/query", Collector_Info__Query)
	r.POST("/api/gui/v1.0/collector/add", Collector_Info__Add)
	r.POST("/api/gui/v1.0/collector/update", Collector_Info__Update)
	r.POST("/api/gui/v1.0/collector/del", Collector_Info__Del)
	r.POST("/api/gui/v1.0/collector/search/field/vague", Collector_Info__Search_Field)
	r.POST("/api/gui/v1.0/collector/search/blurred", Collector_Info__Search_Field_Blurred)
	r.POST("/api/gui/v1.0/collector/synchronise", Collector_Synchronise_Config)

	r.POST("/api/gui/v1.0/config/drive/count", Drive_Config__Count)
	r.POST("/api/gui/v1.0/config/drive/query", Drive_Config__Query)
	r.POST("/api/gui/v1.0/config/drive/add", Drive_Config__Add)
	r.POST("/api/gui/v1.0/config/drive/update", Drive_Config__Update)
	r.POST("/api/gui/v1.0/config/drive/del", Drive_Config__Del)
	r.POST("/api/gui/v1.0/config/drive/search/field/vague", Drive_Config__Search_Field)
	r.POST("/api/gui/v1.0/config/drive/search/blurred", Drive_Config__Search_Field_Blurred)

	r.POST("/api/gui/v1.0/config/points/count", Points_Config__Count)
	r.POST("/api/gui/v1.0/config/points/query", Points_Config__Query)
	r.POST("/api/gui/v1.0/config/points/add", Points_Config__Add)
	r.POST("/api/gui/v1.0/config/points/update", Points_Config__Update)
	r.POST("/api/gui/v1.0/config/points/del", Points_Config__Del)

	r.POST("/api/gui/v1.0/app/restart", App_Restart)

}
