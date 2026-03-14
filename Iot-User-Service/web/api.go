/*
* 日期: 2026.02.15 	PM11:06
* 作者: 范范zwf
* 作用: 外部api
 */
package web

import (
	"database/sql"
	"fmt"
	"log"
	"main/Init"
	db_mysql "main/db/mysql"
	db_redis "main/db/redis"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// 获取刷新令牌
func Api__Login_Refresh_Token(ctx *gin.Context) {
	var jsondata struct {
		ApiKey string
		Secret string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 判断用户名密码是否正确
	db_api, err := db_mysql.Api__Query_ApiKey(jsondata.ApiKey)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{403, "不存在"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	if db_api.Secret != jsondata.Secret {
		ctx.Set("Response", []any{403, "Secret错误"})
		return
	}

	var ClientIP = ctx.ClientIP() // 获取客户端ip
	if db_api.Allow_Ip != ClientIP && db_api.Allow_Ip != "" {
		ctx.Set("Response", []any{403, fmt.Sprintf("ip:%s 禁止请求", db_api.Allow_Ip)})
		return
	}

	if db_api.Discontinued {
		ctx.Set("Response", []any{403, "此key已禁用"})
		return
	}

	// 生成RSA密钥对
	PrivateKey, PublicKey, err := GenerateRSAKeyPair(db_api.Refresh_Token_bits)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 将RSA密钥对转换为PEM格式字符串
	PrivateKey_str, err := PrivateKeyToPEM(PrivateKey)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 将RSA密钥对转换为PEM格式字符串
	PublicKey_str, err := PublicKeyToPEM(PublicKey)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 生成随即刷新令牌
	Aes_Key, err := GenerateSecureRandomString(Refresh_Token_Salt_Length) // 生成AES加密随机盐
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 将接口信息结构体转json并AES加密
	encrypted, err := token_Info__Json_AES_Encrypt(Aes_Key, Token_Api_Info{
		api_id: db_api.Id,
	})
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	timeNow := time.Now()
	Refresh_Token_Time := timeNow.Add(time.Duration(db_api.Refresh_Token_Time) * time.Second)
	// 生成随即刷新令牌
	Refresh_Token, err := CreateShortToken(
		Refresh_Token_Salt_Length,
		PrivateKey,
		encrypted,
		timeNow,
		Refresh_Token_Time,
	)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 记录日志
	db_mysql.Log__Add2(
		0,
		"api_login",
		fmt.Sprintf("ApiKey:%s,IP:%s", jsondata.ApiKey, ClientIP),
	)

	// 将刷新令牌存入redis
	err = db_redis.Api_Refresh_Token_Add(Refresh_Token, db_redis.Api_Refresh_Token_redis_type{
		Api_Id:          db_api.Id,          // 接口id
		Expires_in:      Refresh_Token_Time, // 访问令牌过期时间
		Allow_Ip:        db_api.Allow_Ip,    // 允许ip
		Login_Ip:        ClientIP,           // 登录ip
		Salt:            Aes_Key,            // 随机盐
		RSA_Private_Key: PrivateKey_str,     // RSA私钥
		RSA_Public_Key:  PublicKey_str,      // RSA公钥
	})
	if err != nil {
		ctx.Set("Response", []any{StatusRedis, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", gin.H{
		"Api_Id":              db_api.Id,
		"F_Api_Refresh_Token": Refresh_Token,
		"F_Api_Expires_in":    Refresh_Token_Time.Format(time.RFC3339Nano),
	}})
}

// 获取访问令牌
func Api__Access_Token_Query(ctx *gin.Context) {
	var jsondata struct {
		Api_Id              uint
		F_Api_Refresh_Token string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 判断刷新令牌是否过期
	Refresh_Token_redis, err := db_redis.Api_Refresh_Token_Query(jsondata.Api_Id, jsondata.F_Api_Refresh_Token)
	if err == redis.Nil {
		ctx.Set("Response", []any{401, "刷新令牌过期"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusRedis, err.Error()})
		return
	}

	if Refresh_Token_redis.Api_Id != jsondata.Api_Id {
		ctx.Set("Response", []any{500, "请求API ID和缓存ID不一致"})
		return
	}

	var ClientIP = ctx.ClientIP() // 获取客户端ip
	if Refresh_Token_redis.Login_Ip != ClientIP && Refresh_Token_redis.Login_Ip != "" {
		ctx.Set("Response", []any{403, fmt.Sprintf("ip:%s 禁止请求", ClientIP)})
		return
	}

	// 查询访问令牌的RSA密钥长度
	Access_Token_bits, err := db_mysql.Api__Query_Id__AccessTokenbits(Refresh_Token_redis.Api_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	// 生成RSA密钥对
	PrivateKey, PublicKey, err := GenerateRSAKeyPair(Access_Token_bits)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 将RSA密钥对转换为PEM格式字符串
	PrivateKey_str, err := PrivateKeyToPEM(PrivateKey)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 将RSA密钥对转换为PEM格式字符串
	PublicKey_str, err := PublicKeyToPEM(PublicKey)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 生成随即访问令牌
	Aes_Key, err := GenerateSecureRandomString(Refresh_Token_Salt_Length) // 生成AES加密随机盐
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 将接口信息结构体转json并AES加密
	encrypted, err := token_Info__Json_AES_Encrypt(Aes_Key, Token_Api_Info{
		api_id: Refresh_Token_redis.Api_Id,
	})
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	timeNow := time.Now()                                                                                // 当前时间
	Access_Token_Time := timeNow.Add(time.Duration(Init.Config.SET.Api_Access_Token_Time) * time.Second) // 访问令牌过期时间

	// 生成随即刷新令牌
	Access_Token, err := CreateShortToken(
		Refresh_Token_Salt_Length,
		PrivateKey,
		encrypted,
		timeNow,
		Access_Token_Time,
	)

	// 将访问令牌存入redis
	err = db_redis.Api_Access_Token_Add(
		Access_Token,
		db_redis.Api_Access_Token_redis_type{
			Api_Id:        jsondata.Api_Id,
			Expires_in:    Access_Token_Time, // 访问令牌过期时间
			Refresh_Token: jsondata.F_Api_Refresh_Token,
			Allow_Ip:      Refresh_Token_redis.Allow_Ip, // 允许的ip --- IGNORE ---
			Login_Ip:      ClientIP,                     // 登录ip --- IGNORE ---

			Salt:            Aes_Key,        // 随机盐
			RSA_Private_Key: PrivateKey_str, // RSA私钥
			RSA_Public_Key:  PublicKey_str,  // RSA公钥
		},
	)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", gin.H{
		"F_Api_Access_Token": Access_Token,
		"F_Api_Expires_in":   Access_Token_Time.Format(time.RFC3339Nano),
	}})
}

// 获取访问令牌
func Api__User_status(ctx *gin.Context) {
	var jsondata struct {
		User__Access_Token string // 用户刷新令牌
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if jsondata.User__Access_Token == "" {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	Access_Token_redis, err := db_redis.Access_Token_Query(jsondata.User__Access_Token)
	if err != nil {
		fmt.Print(err, "token无效\n")
	}
	if err == redis.Nil {
		ctx.Set("Response", []any{200, "ok", gin.H{
			"Code":          403,   // 执行码
			"Msg":           "未登陆", // 执行说明
			"User_Id":       0,     // 用户id
			"Expires_in":    "",    // 访问令牌过期时间
			"Refresh_Token": "",    // 本访问令牌的刷新令牌
		}})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusRedis, err.Error(), gin.H{
			"Code":          403,         // 执行码
			"Msg":           err.Error(), // 执行说明
			"User_Id":       0,           // 用户id
			"Expires_in":    "",          // 访问令牌过期时间
			"Refresh_Token": "",          // 本访问令牌的刷新令牌
		}})
		return
	}

	ctx.Set("Response", []any{200, "ok", gin.H{
		"Code":          200,                              // 执行码
		"Msg":           "已登陆",                            // 执行说明
		"User_Id":       Access_Token_redis.User_Id,       // 用户id
		"Expires_in":    Access_Token_redis.Expires_in,    // 访问令牌过期时间
		"Refresh_Token": Access_Token_redis.Refresh_Token, // 本访问令牌的刷新令牌
	}})

}

// 获取访问令牌
func Api__User_Authority_Exist(ctx *gin.Context) {

	var jsondata struct {
		User__Access_Token string // 用户刷新令牌
		Authority_Theme    string // 权限主题
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	defer func(err error) {
		if err != nil {
			log.Panicf("ERROR %s", err)
		}
	}(err)

	var Access_Token_redis db_redis.Access_Token_redis_type
	Access_Token_redis, err = db_redis.Access_Token_Query(jsondata.User__Access_Token)
	if err == redis.Nil {
		ctx.Set("Response", []any{200, "当前用户没有访问令牌"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusRedis, err.Error()})
		return
	}

	var Exist bool
	Exist, err = db_mysql.Authority_User__Query_AuthorityTheme_Exist(Access_Token_redis.User_Id, jsondata.Authority_Theme)
	if err != nil && err != sql.ErrNoRows {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", gin.H{
		"Authority_Exist":    Exist,
		"User__Access_Token": jsondata.User__Access_Token,
		"Authority_Theme":    jsondata.Authority_Theme,
	}})

}

// sdk_api 注册外部api路由
func sdk_api(r *gin.Engine) {
	r.POST("/api/v1.0/login/refresh_token", Api__Login_Refresh_Token) // 获取刷新令牌
	r.POST("/api/v1.0/login/access_token", Api__Access_Token_Query)   // 获取访问令牌

	r.POST("/api/v1.0/user/login/status", Api__User_status)       // 查询当前用户登陆状态
	r.POST("/api/v1.0/user/authority", Api__User_Authority_Exist) // 查询当前用户是否有这个权限
}
