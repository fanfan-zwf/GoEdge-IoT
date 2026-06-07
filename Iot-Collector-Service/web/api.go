/*
* 日期: 2026.02.15 	PM11:06
* 作者: 范范zwf
* 作用: 外部api
 */
package web

import (
	"main/IO/manager/fullConfig"
	"main/db/db_point"
	"time"

	"encoding/json"
	"reflect"

	"github.com/gin-gonic/gin"
)

// ======== 全局只定义一次 ========
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

func api_point_write_value(ctx *gin.Context) {
	// 第一步：用 json.RawMessage 临时接收，实现“不解析Value”
	var tempData []struct {
		Tag   string
		Value json.RawMessage
		Type  string
		// Msg   string
		// Time  time.Time
	}
	// 绑定JSON，只解析Type，不解析Value
	err := ctx.BindJSON(&tempData)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 第二步：转成你需要的最终结构体，并根据Type解析Value
	var jsondata []fullConfig.Value_type
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
		jsondata = append(jsondata, fullConfig.Value_type{
			Tag:   item.Tag,  // 点位名称
			Value: realValue, // 点位值
			Type:  item.Type, // 输出类型
			// Msg:   item.Msg,  // 状态信息
			Time: time.Now(), // 读取时间
		})
	}

	err = db_point.Write_value_Publisher(jsondata)
	if err != nil {
		ctx.Set("Response", []any{404, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok"})
}

func api_alarm_tatus(ctx *gin.Context) {
	var tags []string
	err := ctx.BindJSON(&tags)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	r := db_point.Alarm_Config__Query_list(tags)
	ctx.Set("Response", []any{200, "ok", r})

}
func gui_api(r *gin.Engine) {
	r.POST("/api/v1.0/point/write/value", api_point_write_value)
	r.POST("/api/v1.0/alarm/status", api_alarm_tatus)
}
