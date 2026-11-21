/*
* 日期: 2026.11.11
* 作者: 范范zwf
* 作用: AK/SK 接口账号（api_key 表）管理 API。受统一鉴权中间件保护（不豁免）。
*       secret_key 仅在新增/重置时由服务端生成并一次性返回，列表与后续查询永不返回 SK。
 */
package handler

import (
	"collector/internal/repo/mysql"
	"collector/internal/repo/redis"
	"collector/pkg/conv"
	"collector/pkg/security"
	"time"

	"github.com/gin-gonic/gin"
)

// Secret_Bytes 服务端生成 secret_key 使用的随机字节数（base64url 后约 32 字符，落在 varchar(128) 内）
const Secret_Bytes = 24

// apiKeyQueryRequest 列表/计数入参
type apiKeyQueryRequest struct {
	Page     uint `json:"page"`
	PageSize uint `json:"pageSize"`
}

// apiKeyAddRequest 新增 AK/SK 入参
type apiKeyAddRequest struct {
	AccessKey       string        `json:"access_key"`
	Name            string        `json:"name"`              // 名称/备注（便于识别与搜索，可空）
	Enable          int           `json:"enable"`            // 1=启用 0=禁用（前端总是显式传递）
	ExpireAt        time.Time     `json:"expire_at"`         // RFC3339；零值=永不过期
	AccessTokenTTL  conv.Duration `json:"access_token_ttl"`  // duration 字符串，如 "15m0s"；空=默认
	RefreshTokenTTL conv.Duration `json:"refresh_token_ttl"` // duration 字符串，如 "168h0m0s"；空=默认
}

// apiKeyUpdateRequest 更新 AK/SK 入参（不改 secret_key，SK 只能通过 reset-secret 变更；前后端约定全量提交）
type apiKeyUpdateRequest struct {
	AccessKey       string        `json:"access_key"`
	Name            string        `json:"name"`
	Enable          int           `json:"enable"`
	ExpireAt        time.Time     `json:"expire_at"`
	AccessTokenTTL  conv.Duration `json:"access_token_ttl"`
	RefreshTokenTTL conv.Duration `json:"refresh_token_ttl"`
}

// apiKeyResetSecretRequest 重置 secret_key 入参
type apiKeyResetSecretRequest struct {
	AccessKey string `json:"access_key"`
}

// apiKeyDelRequest 批量删除入参
type apiKeyDelRequest struct {
	Ids []uint `json:"ids"`
}

// api_apikey_query 分页查询 AK/SK 列表（脱敏，不含 secret_key）
func api_apikey_query(ctx *gin.Context) {
	var req apiKeyQueryRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	list, err := mysql.ApiKey__Query(req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", list})
}

// api_apikey_count AK/SK 总数
func api_apikey_count(ctx *gin.Context) {
	var req apiKeyQueryRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	count, err := mysql.ApiKey__Count(req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// api_apikey_add 新增 AK/SK：服务端生成 secret_key 并一次性返回
func api_apikey_add(ctx *gin.Context) {
	var req apiKeyAddRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.AccessKey == "" {
		ctx.Set("Response", []any{417, "access_key 不能为空"})
		return
	}

	// 唯一性校验（Access_Key 在 api_key 内唯一）
	if _, found, err := mysql.ApiKey__Query_By_Access_Key(req.AccessKey); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	} else if found {
		ctx.Set("Response", []any{417, "access_key 已存在"})
		return
	}

	secretKey, err := security.GenKey(Secret_Bytes)
	if err != nil {
		ctx.Set("Response", []any{500, "生成 secret_key 失败"})
		return
	}

	cfg := mysql.ApiKey_Add_type{
		Access_Key:        req.AccessKey,
		Name:              req.Name,
		Secret_Key:        secretKey,
		Enable:            req.Enable,
		Expire_At:         req.ExpireAt,
		Access_Token_TTL:  req.AccessTokenTTL,
		Refresh_Token_TTL: req.RefreshTokenTTL,
	}
	if err := mysql.ApiKey__Add(cfg); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	refreshApiKeyCreds(req.AccessKey) // 写穿验签缓存，新 SK 即时可验签

	// secret_key 仅此一次返回，绝不写入日志
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Add, gin.H{
		"table":             "api_key",
		"access_key":        req.AccessKey,
		"name":              req.Name,
		"enable":            req.Enable,
		"expire_at":         req.ExpireAt,
		"access_token_ttl":  req.AccessTokenTTL,
		"refresh_token_ttl": req.RefreshTokenTTL,
	})
	ctx.Set("Response", []any{200, "ok", gin.H{
		"access_key": req.AccessKey,
		"secret_key": secretKey,
	}})
}

// api_apikey_update 更新 AK/SK 可变字段（不改 secret_key；未传字段保持原值）
func api_apikey_update(ctx *gin.Context) {
	var req apiKeyUpdateRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.AccessKey == "" {
		ctx.Set("Response", []any{417, "access_key 不能为空"})
		return
	}

	acc, found, err := mysql.ApiKey__Query_By_Access_Key(req.AccessKey)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	if !found {
		ctx.Set("Response", []any{417, "AK/SK 不存在"})
		return
	}

	// 无指针：前端编辑弹窗会加载并全量提交当前值，未填字段即为零值
	cfg := mysql.ApiKey_Add_type{
		Access_Key:        req.AccessKey,
		Name:              req.Name,
		Secret_Key:        acc.Secret_Key, // 保持不变
		Enable:            req.Enable,
		Expire_At:         req.ExpireAt,
		Access_Token_TTL:  req.AccessTokenTTL,
		Refresh_Token_TTL: req.RefreshTokenTTL,
	}
	if err := mysql.ApiKey__Update_Mutable(cfg); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	refreshApiKeyCreds(req.AccessKey) // 启用/过期变更即时生效
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Update, gin.H{
		"table":             "api_key",
		"id":                acc.Id,
		"access_key":        req.AccessKey,
		"name":              req.Name,
		"enable":            req.Enable,
		"expire_at":         req.ExpireAt,
		"access_token_ttl":  req.AccessTokenTTL,
		"refresh_token_ttl": req.RefreshTokenTTL,
	})
	ctx.Set("Response", []any{200, "ok"})
}

// api_apikey_reset_secret 重置 secret_key：服务端生成新值并一次性返回
func api_apikey_reset_secret(ctx *gin.Context) {
	var req apiKeyResetSecretRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.AccessKey == "" {
		ctx.Set("Response", []any{417, "access_key 不能为空"})
		return
	}

	acc, found, err := mysql.ApiKey__Query_By_Access_Key(req.AccessKey)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	if !found {
		ctx.Set("Response", []any{417, "AK/SK 不存在"})
		return
	}

	newSecret, err := security.GenKey(Secret_Bytes)
	if err != nil {
		ctx.Set("Response", []any{500, "生成 secret_key 失败"})
		return
	}

	cfg := mysql.ApiKey_Add_type{
		Access_Key:        req.AccessKey,
		Name:              acc.Name, // 重置 SK 时保留原其余字段
		Secret_Key:        newSecret,
		Enable:            acc.Enable,
		Expire_At:         acc.Expire_At,
		Access_Token_TTL:  acc.Access_Token_TTL,
		Refresh_Token_TTL: acc.Refresh_Token_TTL,
	}
	if err := mysql.ApiKey__Update_Mutable(cfg); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	refreshApiKeyCreds(req.AccessKey) // 新 SK 写穿缓存，旧 SK 立即失效
	ctx.Set("Response", []any{200, "ok", gin.H{
		"access_key": req.AccessKey,
		"secret_key": newSecret,
	}})
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Update, gin.H{
		"table":      "api_key",
		"id":         acc.Id,
		"access_key": req.AccessKey,
		"action":     "reset_secret", // 只记重置信令这一事实，新 SK 不入库
	})
}

// api_apikey_del 批量删除 AK/SK
func api_apikey_del(ctx *gin.Context) {
	var req apiKeyDelRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if len(req.Ids) == 0 {
		ctx.Set("Response", []any{417, "ids 不能为空"})
		return
	}
	// 删除前先取出 access_key，删库后逐个清理其 Redis 验签缓存（避免已删密钥凭 TTL 前仍可验签）
	keys, _ := mysql.ApiKey__AccessKeysByIds(req.Ids)
	if err := mysql.ApiKey__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	for _, k := range keys {
		_ = redis.Auth_ApiKey_Creds_Del(k)
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Del, gin.H{
		"table": "api_key",
		"ids":   req.Ids,
	})
	ctx.Set("Response", []any{200, "ok"})
}

// apikey_api_register 注册 AK/SK 管理路由（受统一鉴权中间件保护）
func apikey_api_register(r *gin.Engine) {
	r.POST("/api/v1.0/config/apikey/query", api_apikey_query)
	r.POST("/api/v1.0/config/apikey/count", api_apikey_count)
	r.POST("/api/v1.0/config/apikey/add", api_apikey_add)
	r.POST("/api/v1.0/config/apikey/update", api_apikey_update)
	r.POST("/api/v1.0/config/apikey/reset-secret", api_apikey_reset_secret)
	r.POST("/api/v1.0/config/apikey/del", api_apikey_del)
}
