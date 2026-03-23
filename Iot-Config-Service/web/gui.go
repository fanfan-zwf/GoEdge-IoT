/*
* 日期: 2025.12.23 16:50
* 作者: 范范zwf
* 作用: api-用户相关
 */
package web

import (
	db_mysql "main/db/mysql"

	"database/sql"

	"github.com/gin-gonic/gin"
)

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
	if err == sql.ErrNoRows || count == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
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
	if err == sql.ErrNoRows || len(config_list) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
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

// 采集-》增加配置 传递: config 配置数组形式 返回: err 错误
func Collector_Info__Del(ctx *gin.Context) {
	var jsondata struct {
		id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Collector_Info__Del(jsondata.id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

func Drive_Config__Count(ctx *gin.Context) {
	var jsondata struct {
		Page         uint
		Page_Size    uint
		Collector_Id uint
		Drive_Type   string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	count, err := db_mysql.Drive_Config__Count(jsondata.Collector_Id, jsondata.Drive_Type, jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows || count == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", count})
}
func gui_api(r *gin.Engine) {
	r.POST("/api/gui/v1.0/login/name", Collector_Info__Count)
	r.POST("/api/gui/v1.0/login/name", Collector_Info__Query)
	r.POST("/api/gui/v1.0/login/name", Collector_Info__Add)
	r.POST("/api/gui/v1.0/login/name", Collector_Info__Del)

}
