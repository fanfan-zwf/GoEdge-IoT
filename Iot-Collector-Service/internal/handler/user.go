/*
* 日期: 2026.11.11
* 作者: 范范zwf
* 作用: 网页登录用户（user_account 表）管理 API。受统一鉴权中间件保护（不豁免）。
*       列表/查询绝不返回密码哈希。新增必填密码，编辑可重置密码（留空保持原值）。
 */
package handler

import (
	"collector/internal/repo/mysql"
	"collector/pkg/conv"
	"collector/pkg/security"
	"time"

	"github.com/gin-gonic/gin"
)

// userQueryRequest 列表/计数入参
type userQueryRequest struct {
	Page     uint `json:"page"`
	PageSize uint `json:"pageSize"`
}

// userAddRequest 新增登录用户入参（前端新增表单会全量提交各字段）
type userAddRequest struct {
	AccessKey       string        `json:"access_key"`
	Password        string        `json:"password"` // 明文，服务端 bcrypt（登录用户必填）
	Enable          int           `json:"enable"`   // 1=启用 0=禁用（前端总是显式传递）
	ExpireAt        time.Time     `json:"expire_at"`
	AccessTokenTTL  conv.Duration `json:"access_token_ttl"`  // duration 字符串，如 "15m0s"；空=默认
	RefreshTokenTTL conv.Duration `json:"refresh_token_ttl"` // duration 字符串，如 "168h0m0s"；空=默认
}

// userUpdateRequest 更新登录用户入参（access_key 定位记录；前端编辑弹窗会全量提交当前值）
type userUpdateRequest struct {
	AccessKey       string        `json:"access_key"`
	Password        string        `json:"password"`         // 非空则重设密码哈希；空=保持原值
	Enable          int           `json:"enable"`           // 1=启用 0=禁用
	ExpireAt        time.Time     `json:"expire_at"`        // 零值/null=永不过期（前端清空即置为永不过期）
	AccessTokenTTL  conv.Duration `json:"access_token_ttl"` // duration 字符串；空/0=保持原值
	RefreshTokenTTL conv.Duration `json:"refresh_token_ttl"`
}

// userDelRequest 批量删除入参
type userDelRequest struct {
	Ids []uint `json:"ids"`
}

// api_user_query 分页查询登录用户列表（脱敏）
func api_user_query(ctx *gin.Context) {
	var req userQueryRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	list, err := mysql.User__Query(req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", list})
}

// api_user_count 登录用户总数
func api_user_count(ctx *gin.Context) {
	var req userQueryRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	count, err := mysql.User__Count(req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", count})
}

// api_user_add 新增登录用户（密码必填）
func api_user_add(ctx *gin.Context) {
	var req userAddRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.AccessKey == "" {
		ctx.Set("Response", []any{417, "用户名不能为空"})
		return
	}
	if req.Password == "" {
		ctx.Set("Response", []any{417, "登录密码不能为空"})
		return
	}

	// 唯一性校验（用户名在 user_account 内唯一）
	if _, found, err := mysql.User__Query_By_Access_Key(req.AccessKey); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	} else if found {
		ctx.Set("Response", []any{417, "用户名已存在"})
		return
	}

	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		ctx.Set("Response", []any{500, "密码哈希失败"})
		return
	}

	cfg := mysql.User_Add_type{
		Access_Key:        req.AccessKey,
		Login_Pwd_Hash:    hashed,
		Enable:            req.Enable,
		Expire_At:         req.ExpireAt,
		Access_Token_TTL:  req.AccessTokenTTL,
		Refresh_Token_TTL: req.RefreshTokenTTL,
	}
	if err := mysql.User__Add(cfg); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	// 日志不记密码，只记账号属性
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Add, gin.H{
		"table":             "user_account",
		"access_key":        req.AccessKey,
		"enable":            req.Enable,
		"expire_at":         req.ExpireAt,
		"access_token_ttl":  req.AccessTokenTTL,
		"refresh_token_ttl": req.RefreshTokenTTL,
	})
	ctx.Set("Response", []any{200, "ok", gin.H{"access_key": req.AccessKey}})
}

// api_user_update 更新登录用户（不改用户名；未传字段保持原值）
func api_user_update(ctx *gin.Context) {
	var req userUpdateRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.AccessKey == "" {
		ctx.Set("Response", []any{417, "用户名不能为空"})
		return
	}

	acc, found, err := mysql.User__Query_By_Access_Key(req.AccessKey)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	if !found {
		ctx.Set("Response", []any{417, "用户不存在"})
		return
	}

	// 安全护栏：禁止禁用当前登录账号（避免自锁）
	enable := req.Enable
	if enable != 1 && req.AccessKey == ctx.GetString(CtxKeyAccessKey) {
		ctx.Set("Response", []any{403, "不能禁用当前登录账号"})
		return
	}

	// 密码：非空则重设哈希，否则保持原值
	pwdHash := acc.Login_Pwd_Hash
	if req.Password != "" {
		hashed, err := security.HashPassword(req.Password)
		if err != nil {
			ctx.Set("Response", []any{500, "密码哈希失败"})
			return
		}
		pwdHash = hashed
	}

	// 到期时间：前端清空（null）即置为永不过期（零值→落库 NULL）
	expire := req.ExpireAt

	// 时效：0 保持原值
	accessTTL := req.AccessTokenTTL
	if accessTTL == 0 {
		accessTTL = acc.Access_Token_TTL
	}
	refreshTTL := req.RefreshTokenTTL
	if refreshTTL == 0 {
		refreshTTL = acc.Refresh_Token_TTL
	}

	cfg := mysql.User_Add_type{
		Access_Key:        req.AccessKey,
		Login_Pwd_Hash:    pwdHash,
		Enable:            enable,
		Expire_At:         expire,
		Access_Token_TTL:  accessTTL,
		Refresh_Token_TTL: refreshTTL,
	}
	if err := mysql.User__Update_Mutable(cfg); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Update, gin.H{
		"table":             "user_account",
		"id":                acc.Id,
		"access_key":        req.AccessKey,
		"enable":            enable,
		"expire_at":         expire,
		"access_token_ttl":  accessTTL,
		"refresh_token_ttl": refreshTTL,
		"password_reset":    req.Password != "", // 只记是否重设了密码，不记密码本身
	})
	ctx.Set("Response", []any{200, "ok"})
}

// api_user_del 批量删除登录用户（禁止删除当前登录账号）
func api_user_del(ctx *gin.Context) {
	var req userDelRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if len(req.Ids) == 0 {
		ctx.Set("Response", []any{417, "ids 不能为空"})
		return
	}

	if currentKey := ctx.GetString(CtxKeyAccessKey); currentKey != "" {
		if acc, found, err := mysql.User__Query_By_Access_Key(currentKey); err == nil && found {
			for _, id := range req.Ids {
				if id == acc.Id {
					ctx.Set("Response", []any{403, "不能删除当前登录账号"})
					return
				}
			}
		}
	}

	if err := mysql.User__Del(req.Ids...); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Del, gin.H{
		"table": "user_account",
		"ids":   req.Ids,
	})
	ctx.Set("Response", []any{200, "ok"})
}

// user_api_register 注册登录用户管理路由（受统一鉴权中间件保护）
func user_api_register(r *gin.Engine) {
	r.POST("/api/v1.0/config/user/query", api_user_query)
	r.POST("/api/v1.0/config/user/count", api_user_count)
	r.POST("/api/v1.0/config/user/add", api_user_add)
	r.POST("/api/v1.0/config/user/update", api_user_update)
	r.POST("/api/v1.0/config/user/del", api_user_del)
}
