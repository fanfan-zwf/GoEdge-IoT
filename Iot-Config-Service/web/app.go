/*
* 日期: 2025.10.08 PM10:08
* 作者: 范范zwf
* 作用: app接口
 */
package web

import (
	"encoding/base64"
	"encoding/json"
	"main/db/mysql"
	"main/db/redis"
	"strings"
	"time"

	"fmt"
	"math/rand"

	"github.com/gin-gonic/gin"
)

// 字符集：大小写字母+数字
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandStr 生成长度为n的随机字符串
func RandStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// 获取basic auth通用函数
func collector_GetBasicAuth(ctx *gin.Context) {
	// 从header获取basic auth（base64编码）
	authHeader := ctx.GetHeader("Collector_Basic")
	if authHeader == "" {
		ctx.Set("Response", []any{401, "需要basic auth"})
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(authHeader)
	if err != nil {
		ctx.Set("Response", []any{401, "basic auth解码失败"})
		return
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		ctx.Set("Response", []any{401, "需要basic auth"})
		return
	}
	username, passwd := parts[0], parts[1]

	q, err := mysql.Collector_Config__Query_MqttByLabel(username)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if q.Passwd != passwd {
		err = fmt.Errorf("密码错误")
		ctx.Set("Response", []any{401, "密码错误"})
		return
	}
	jsondata, err := json.Marshal(q)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	token := RandStr(256)
	key := fmt.Sprintf("Token:Collector:%s", token)
	err = redis.Write_Key(key, string(jsondata), 2*time.Hour)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", token})
}

func redis_token(ctx *gin.Context) (q mysql.Collector_Config__Query_MqttByLabel_type, err error) {
	token := ctx.GetHeader("Collector_Token")
	if token == "" {
		ctx.Set("Response", []any{401, "需要Collector_Token"})
		err = fmt.Errorf("需要Collector_Token")
		return
	}

	key := fmt.Sprintf("Token:Collector:%s", token)
	var jsondata string
	jsondata, err = redis.Read_Key(key)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	err = json.Unmarshal([]byte(jsondata), &q)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}
	return
}

// 给采集服务获取驱动配置接口
func collector_Drive_Config__Query(ctx *gin.Context) {
	q, err := redis_token(ctx)
	if err != nil {
		return
	}

	list, err := mysql.CollectorGet_Drive_Config__Query([]string{q.Label}, []uint{q.Id})
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", list})
}

// 给采集服务获取点位配置接口
func collector_Point_Config__Query(ctx *gin.Context) {
	q, err := redis_token(ctx)
	if err != nil {
		return
	}

	list, err := mysql.CollectorGet_Point_Config__Query([]string{q.Label}, []uint{q.Id}, []uint{})
	if err == mysql.ErrNoRows {
		err = nil
	} else if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", list})
}

func app_api(r *gin.Engine) {
	r.POST("/api/app/v1.0/login", collector_GetBasicAuth)

	r.POST("/api/app/v1.0/collector/drive/config/query", collector_Drive_Config__Query)
	r.POST("/api/app/v1.0/collector/point/config/query", collector_Point_Config__Query)
}
