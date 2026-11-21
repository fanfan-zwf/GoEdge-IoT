/*
* 日期: 2026.11.11
* 作者: 范范zwf
* 作用: 统一鉴权。单个中间件自动识别两种凭证（服务间 HMAC-SHA256 签名 / 前端 Bearer token），
*       校验通过后把 access_key 写入上下文，业务代码不区分凭证来源，共用一套权限与逻辑。
*       同时提供账号密码登录、退出登录、刷新令牌、内部同步 AKSK（免鉴权）四个接口。
*       令牌以 JSON 形式绑定客户端 IP 与 User-Agent，请求时校验 IP 一致，不一致则当前账号全部退出。
 */
package handler

import (
	"bytes"
	"collector/internal/repo/mysql"
	"collector/internal/repo/redis"
	rbacsvc "collector/internal/service/rbac"
	"collector/pkg/security"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ======== 鉴权相关常量 ========
const (
	// 请求头（通路 1：服务间 HMAC 签名）
	HeaderAccessKey = "X-AccessKey"
	HeaderTimestamp = "X-Timestamp"
	HeaderNonce     = "X-Nonce"
	HeaderSignature = "X-Signature"

	// Gin 上下文键：校验通过后写入的 access_key，后续业务代码统一读取
	CtxKeyAccessKey = "Access_Key"

	// 免鉴权路由（登录 / 登出 / 刷新 / 内部同步）+ 免鉴权的接口说明页
	PathLogin    = "/api/v1.0/auth/login"
	PathLogout   = "/api/v1.0/auth/logout"
	PathRefresh  = "/api/v1.0/auth/refresh-token"
	PathSyncAkSk = "/api/v1.0/internal/sync-aksk"

	// GenKey 生成 token/nonce 使用的随机字节数
	Token_Bytes = 32
)

// HMAC 时间窗口与 nonce 防重放 TTL（协议级常量，不区分账号）
var (
	Hmac_Sign_Window = 60 * time.Second     // 允许的请求时间偏差 ±60 秒
	Auth_Nonce_TTL   = 2 * Hmac_Sign_Window // nonce 记录保留 2 倍窗口，覆盖重放区间

	// ApiKey_Creds_TTL AK/SK 验签凭据在 Redis 的缓存时长：到期回源 MySQL 重建；管理端增改/重置/同步会即时刷新
	ApiKey_Creds_TTL = 30 * time.Minute
)

// ======== 统一鉴权中间件 ========

// Auth_Middleware 单中间件识别两种凭证：
//  1. Authorization: Bearer {access_token} —— 前端用户认证，去 Redis 校验 token；
//  2. X-AccessKey / X-Timestamp / X-Nonce / X-Signature —— 服务间 HMAC-SHA256 签名校验；
//     签名原文顺序：timestamp + nonce + request path + request body，时间窗口 ±60s，nonce 防重放。
//
// 校验成功后把 access_key 写入上下文（CtxKeyAccessKey），供后续权限判断与业务代码复用。
func Auth_Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 免鉴权路径直接放行
		if authExemptPath(ctx.Request.URL.Path) {
			ctx.Next()
			return
		}

		// ---- 通路 2：Bearer token ----
		token := bearerToken(ctx)
		// WebSocket 握手由浏览器发起、无法携带 Authorization 头，仅对该端点改从子协议位取令牌
		if token == "" && ctx.Request.URL.Path == PathPointWS {
			token = wsSubprotocolToken(ctx)
		}
		if token != "" {
			info, ok, err := redis.Auth_Access_Token_Get(token)
			if err != nil {
				ctx.Set("Response", []any{521, "鉴权服务暂时不可用"})
				ctx.Abort()
				return
			}
			if !ok {
				ctx.Set("Response", []any{401, "访问令牌无效或已过期"})
				ctx.Abort()
				return
			}
			// IP 一致性校验：与签发时不一致则视为会话异常，该账号全部令牌立即失效
			if ipChanged(info.IP, ctx.ClientIP()) {
				if err := redis.Auth_Account_Token_Logout(info.Access_Key); err != nil {
					log.Printf("WARN 账号全量登出失败 access_key=%s: %v", info.Access_Key, err)
				}
				ctx.Set("Response", []any{401, "登录 IP 已变化，该账号已全部退出，请重新登录"})
				ctx.Abort()
				return
			}
			// 主体 id 于登录/刷新时写入令牌，请求期直接取用，不再查库
			//（禁用/删除用户于 access 令牌自然过期或下次 refresh 时生效）
			ctx.Set(CtxKeyAccessKey, info.Access_Key)
			setLogActor(ctx, mysql.Log_Source_User, info.User_Id) // 操作日志归属 + RBAC 主体：登录用户
			ctx.Next()
			return
		}

		// ---- 通路 1：HMAC-SHA256 签名 ----
		accessKey := ctx.GetHeader(HeaderAccessKey)
		timestamp := ctx.GetHeader(HeaderTimestamp)
		nonce := ctx.GetHeader(HeaderNonce)
		signature := ctx.GetHeader(HeaderSignature)

		if accessKey != "" && timestamp != "" && nonce != "" && signature != "" {
			// 1. 时间窗口校验
			if !verifyTimestamp(timestamp) {
				ctx.Set("Response", []any{401, "时间戳超出允许窗口（±60s）"})
				ctx.Abort()
				return
			}

			// 2. AK/SK 凭据（优先 Redis 缓存，未命中回源 MySQL 并写回）+ 状态校验，取 Secret_Key 供验签
			creds, found, err := loadApiKeyCreds(accessKey)
			if err != nil {
				ctx.Set("Response", []any{520, "鉴权查询失败"})
				ctx.Abort()
				return
			}
			if !found {
				ctx.Set("Response", []any{401, "access_key 无效"})
				ctx.Abort()
				return
			}
			if denyResp := apiKeyCredsDeny(creds); denyResp != nil {
				ctx.Set("Response", denyResp)
				ctx.Abort()
				return
			}

			// 3. nonce 防重放（按 access_key 隔离）
			fresh, err := redis.Auth_Nonce_Add(accessKey, nonce, Auth_Nonce_TTL)
			if err != nil {
				ctx.Set("Response", []any{521, "鉴权服务暂时不可用"})
				ctx.Abort()
				return
			}
			if !fresh {
				ctx.Set("Response", []any{401, "请求重放（nonce 已使用）"})
				ctx.Abort()
				return
			}

			// 4. 读取 body 计算签名，随后回填给下游业务处理器
			body, err := io.ReadAll(ctx.Request.Body)
			if err != nil {
				ctx.Set("Response", []any{417, "读取请求体失败"})
				ctx.Abort()
				return
			}
			_ = ctx.Request.Body.Close()
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))

			// 5. 校验签名（常量时间比较）
			expect := hmacSign(creds.Secret, timestamp, nonce, ctx.Request.URL.Path, body)
			if !hmac.Equal([]byte(expect), []byte(strings.ToLower(signature))) {
				ctx.Set("Response", []any{401, "签名校验失败"})
				ctx.Abort()
				return
			}

			ctx.Set(CtxKeyAccessKey, accessKey)
			setLogActor(ctx, mysql.Log_Source_AKSK, creds.Id) // 操作日志归属：AK/SK 接口账号

			// AK/SK 没有登录接口：节流窗口内首次使用该凭证视为一次“登陆”，只落一条日志
			if first, tErr := redis.Auth_AkSk_Touch_Throttle(accessKey, AkSk_Login_Log_Window); tErr == nil && first {
				logLogin(ctx, mysql.Log_Source_AKSK, creds.Id, accessKey, gin.H{"action": "aksk_first_use"})
			}

			ctx.Next()
			return
		}

		// ---- 无有效凭证 ----
		ctx.Set("Response", []any{401, "缺少访问凭证（Bearer token 或 HMAC 签名头）"})
		ctx.Abort()
	}
}

// ======== 鉴权处理器 ========

// loginRequest 账号密码登录请求
type loginRequest struct {
	AccessKey string `json:"access_key"`
	Password  string `json:"password"`
}

// api_auth_login 账号密码登录：校验 access_key + bcrypt 密码，成功后下发 access/refresh token
func api_auth_login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.AccessKey == "" || req.Password == "" {
		ctx.Set("Response", []any{417, "用户名和密码不能为空"})
		return
	}

	acc, found, err := mysql.User__Query_By_Access_Key(req.AccessKey)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	// 用户名不存在、未设置登录密码、或密码错误，统一返回模糊提示，避免账号枚举
	if !found || acc.Login_Pwd_Hash == "" || !security.VerifyPassword(acc.Login_Pwd_Hash, req.Password) {
		ctx.Set("Response", []any{401, "用户名或密码错误"})
		return
	}
	if denyResp := userStatusDeny(acc); denyResp != nil {
		ctx.Set("Response", denyResp)
		return
	}

	accessToken, err := security.GenKey(Token_Bytes)
	if err != nil {
		ctx.Set("Response", []any{500, "生成访问令牌失败"})
		return
	}
	refreshToken, err := security.GenKey(Token_Bytes)
	if err != nil {
		ctx.Set("Response", []any{500, "生成刷新令牌失败"})
		return
	}

	accessTTL := time.Duration(acc.Access_Token_TTL)
	refreshTTL := time.Duration(acc.Refresh_Token_TTL)
	// 令牌绑定当前请求的客户端 IP 与 User-Agent，后续请求会校验 IP 一致性
	session := newTokenInfo(ctx, acc.Id, acc.Access_Key)
	if err := redis.Auth_Access_Token_Add(accessToken, session, accessTTL); err != nil {
		ctx.Set("Response", []any{521, "写入访问令牌失败"})
		return
	}
	if err := redis.Auth_Refresh_Token_Add(refreshToken, session, refreshTTL); err != nil {
		ctx.Set("Response", []any{521, "写入刷新令牌失败"})
		return
	}

	// 绝不返回 secret_key / login_pwd_hash
	logLogin(ctx, mysql.Log_Source_User, acc.Id, acc.Access_Key, gin.H{"action": "password_login"}) // 登陆成功写日志（异步，失败不影响响应）
	// 回填前端菜单/按钮可见性所需的角色码与权限码（仅供 UI；API 级拦截仍由 RBAC 中间件独立裁决）。
	// 取数失败不阻断登录，降级为空集合（前端仅显示不受控菜单，按钮自然隐藏）。
	rbacRoles, rbacPerms, rbacErr := rbacsvc.SubjectAuth(mysql.Rbac_Subject_Biz, acc.Id)
	if rbacErr != nil {
		log.Printf("WARN 登录回填 RBAC 权限失败(降级为空) | user=%d | err=%v", acc.Id, rbacErr)
		rbacRoles, rbacPerms = []string{}, []string{}
	}
	ctx.Set("Response", []any{200, "ok", gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    int64(time.Duration(acc.Access_Token_TTL).Seconds()), // access_token 有效期（秒）
		"access_key":    acc.Access_Key,
		"roles":         rbacRoles,
		"permissions":   rbacPerms,
	}})
}

// logoutRequest 登出请求（可选携带 refresh_token，以便一并使其失效）
type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// api_auth_logout 退出登录：使当前 access_token（取自 Authorization）及可选 refresh_token 在服务端立即失效。
// 采用尽力而为策略：即使令牌已过期或 body 为空也返回成功，便于前端无条件清理本地状态。
func api_auth_logout(ctx *gin.Context) {
	var (
		info    redis.Auth_Token_Info
		hasInfo bool
	)
	access := bearerToken(ctx)
	if access != "" {
		// 先取会话信息再删：登出路由不走鉴权中间件，日志的操作人只能从令牌里解析
		if got, ok, _ := redis.Auth_Access_Token_Get(access); ok {
			info, hasInfo = got, true
		}
		_ = redis.Auth_Access_Token_Del(access)
	}
	var req logoutRequest
	_ = ctx.ShouldBindJSON(&req) // 允许空 body，忽略解析错误
	if req.RefreshToken != "" {
		_ = redis.Auth_Refresh_Token_Del(req.RefreshToken)
	}
	if hasInfo {
		logLogout(ctx, mysql.Log_Source_User, info.User_Id, info.Access_Key, gin.H{
			"action":       "logout",
			"with_refresh": req.RefreshToken != "",
		})
	}
	ctx.Set("Response", []any{200, "ok"})
}

// refreshRequest 刷新令牌请求
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// api_auth_refresh 刷新 access_token：校验 refresh_token 后轮换（旧 refresh 失效，发新 access/refresh）
func api_auth_refresh(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.RefreshToken == "" {
		ctx.Set("Response", []any{417, "refresh_token 不能为空"})
		return
	}

	info, ok, err := redis.Auth_Refresh_Token_Get(req.RefreshToken)
	if err != nil {
		ctx.Set("Response", []any{521, "鉴权服务暂时不可用"})
		return
	}
	if !ok {
		ctx.Set("Response", []any{401, "刷新令牌无效或已过期"})
		return
	}
	// IP 一致性校验：与签发时不一致则不轮换，直接将该账号全部令牌失效
	if ipChanged(info.IP, ctx.ClientIP()) {
		if err := redis.Auth_Account_Token_Logout(info.Access_Key); err != nil {
			log.Printf("WARN 账号全量登出失败 access_key=%s: %v", info.Access_Key, err)
		}
		ctx.Set("Response", []any{401, "登录 IP 已变化，该账号已全部退出，请重新登录"})
		return
	}
	accessKey := info.Access_Key

	acc, found, err := mysql.User__Query_By_Access_Key(accessKey)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	if !found {
		ctx.Set("Response", []any{401, "账号不存在"})
		return
	}
	if denyResp := userStatusDeny(acc); denyResp != nil {
		ctx.Set("Response", denyResp)
		return
	}

	newAccess, err := security.GenKey(Token_Bytes)
	if err != nil {
		ctx.Set("Response", []any{500, "生成访问令牌失败"})
		return
	}
	newRefresh, err := security.GenKey(Token_Bytes)
	if err != nil {
		ctx.Set("Response", []any{500, "生成刷新令牌失败"})
		return
	}

	accessTTL := time.Duration(acc.Access_Token_TTL)
	refreshTTL := time.Duration(acc.Refresh_Token_TTL)
	// 轮换时以当前请求环境重新绑定会话信息
	session := newTokenInfo(ctx, acc.Id, acc.Access_Key)
	if err := redis.Auth_Access_Token_Add(newAccess, session, accessTTL); err != nil {
		ctx.Set("Response", []any{521, "写入访问令牌失败"})
		return
	}
	if err := redis.Auth_Refresh_Token_Add(newRefresh, session, refreshTTL); err != nil {
		ctx.Set("Response", []any{521, "写入刷新令牌失败"})
		return
	}
	// 轮换：旧刷新令牌失效
	if err := redis.Auth_Refresh_Token_Del(req.RefreshToken); err != nil {
		log.Printf("WARN 删除旧刷新令牌失败: %v", err)
	}
	// 令牌续期也是一次登陆事件（不记则只能看到首次登录，看不出会话一直活着）
	logLogin(ctx, mysql.Log_Source_User, acc.Id, acc.Access_Key, gin.H{"action": "refresh_token"})

	ctx.Set("Response", []any{200, "ok", gin.H{
		"access_token":  newAccess,
		"refresh_token": newRefresh,
		"token_type":    "Bearer",
		"expires_in":    int64(time.Duration(acc.Access_Token_TTL).Seconds()),
		"access_key":    acc.Access_Key,
	}})
}

// syncAccount 内部同步的单个账号入参
type syncAccount struct {
	AccessKey       string     `json:"access_key"`
	SecretKey       string     `json:"secret_key"`
	LoginPwdHash    string     `json:"login_pwd_hash"` // 已哈希的密码（与 password 二选一，优先 password）
	Password        string     `json:"password"`       // 明文密码，服务端 bcrypt 后存储
	Enable          *int       `json:"enable"`         // 缺省视为 1（启用）
	ExpireAt        *time.Time `json:"expire_at"`      // RFC3339；缺省=永不过期
	AccessTokenTTL  uint       `json:"access_token_ttl"`
	RefreshTokenTTL uint       `json:"refresh_token_ttl"`
}

// syncAkSkRequest 内部同步请求体
type syncAkSkRequest struct {
	Accounts []syncAccount `json:"accounts"`
}

// api_internal_sync_aksk 内部同步账号 AKSK（不加载鉴权中间件）：按 access_key upsert
func api_internal_sync_aksk(ctx *gin.Context) {
	var req syncAkSkRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if len(req.Accounts) == 0 {
		ctx.Set("Response", []any{417, "accounts 不能为空"})
		return
	}

	list := make([]mysql.ApiKey_Add_type, 0, len(req.Accounts))
	for i, a := range req.Accounts {
		if a.AccessKey == "" || a.SecretKey == "" {
			ctx.Set("Response", []any{417, "第" + strconv.Itoa(i+1) + "条 access_key / secret_key 不能为空"})
			return
		}

		enable := 1
		if a.Enable != nil {
			enable = *a.Enable
		}

		var expire time.Time
		if a.ExpireAt != nil {
			expire = *a.ExpireAt
		}

		list = append(list, mysql.ApiKey_Add_type{
			Access_Key: a.AccessKey,
			Secret_Key: a.SecretKey,
			Enable:     enable,
			Expire_At:  expire,
		})
	}

	added, updated, err := mysql.ApiKey__Sync(list)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	// 同步后立即刷新各 access_key 的验签缓存，使新增/变更即时生效
	for _, a := range req.Accounts {
		refreshApiKeyCreds(a.AccessKey)
	}
	ctx.Set("Response", []any{200, "同步成功", gin.H{"added": added, "updated": updated}})
}

// ======== 内部辅助函数 ========

// auth_api_register 注册鉴权与内部同步路由（这三条被中间件按路径豁免）
func auth_api_register(r *gin.Engine) {
	r.POST(PathLogin, api_auth_login)
	r.POST(PathLogout, api_auth_logout)
	r.POST(PathRefresh, api_auth_refresh)
	r.POST(PathSyncAkSk, api_internal_sync_aksk)
}

// authExemptPath 判断是否为免鉴权路径。
// 登录/登出/刷新本身就是发凭证的入口，内部同步由可信配置服务调用，
// 接口说明页需要浏览器直接打开（只列清单，不输出任何业务数据与凭证）。
func authExemptPath(path string) bool {
	switch path {
	case PathLogin, PathLogout, PathRefresh, PathSyncAkSk,
		PathDocs, PathDocsRoot, PathDocsSlash:
		return true
	}
	return false
}

// bearerToken 从 Authorization 头解析 Bearer token，无则返回空串
func bearerToken(ctx *gin.Context) string {
	auth := ctx.GetHeader("Authorization")
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return strings.TrimSpace(auth[len(prefix):])
	}
	return ""
}

// wsSubprotocolToken 取浏览器借 Sec-WebSocket-Protocol 子协议位携带的 access_token。
// 选子协议头而非 URL 查询参数，是为了避免令牌被访问日志、代理日志与浏览器历史明文记录；
// 令牌为 RawURLEncoding（[A-Za-z0-9-_]），符合 RFC 6455 子协议 token 语法。
func wsSubprotocolToken(ctx *gin.Context) string {
	for _, p := range strings.Split(ctx.GetHeader("Sec-WebSocket-Protocol"), ",") {
		if s := strings.TrimSpace(p); s != "" {
			return s
		}
	}
	return ""
}

// newTokenInfo 构造令牌在 Redis 中的会话绑定信息（主体 id + 归属账号 + 当前请求的客户端 IP / User-Agent + 签发时间）
func newTokenInfo(ctx *gin.Context, userId uint, accessKey string) redis.Auth_Token_Info {
	return redis.Auth_Token_Info{
		User_Id:    userId,
		Access_Key: accessKey,
		IP:         normalizeIP(ctx.ClientIP()),
		User_Agent: ctx.Request.UserAgent(),
		Issued_At:  time.Now(), // 序列化由 time.Time 接管，为 RFC3339Nano（保留完整精度，不做截断）
	}
}

// normalizeIP 统一 IP 文本形式，使 ::ffff:127.0.0.1 与 127.0.0.1 等等价写法归一；无法解析则原样返回
func normalizeIP(s string) string {
	s = strings.TrimSpace(s)
	if ip := net.ParseIP(s); ip != nil {
		return ip.String()
	}
	return s
}

// ipChanged 判断令牌签发时的 IP 与当前请求是否不一致（任一侧为空时跳过校验，兼容历史令牌与代理取不到 IP 的情况）
func ipChanged(bound, current string) bool {
	b, c := normalizeIP(bound), normalizeIP(current)
	return b != "" && c != "" && b != c
}

// verifyTimestamp 校验 Unix 秒时间戳是否在允许窗口内
func verifyTimestamp(ts string) bool {
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	diff := time.Now().Unix() - sec
	if diff < 0 {
		diff = -diff
	}
	return diff <= int64(Hmac_Sign_Window.Seconds())
}

// hmacSign 计算 HMAC-SHA256 签名，原文顺序：timestamp + nonce + request path + request body；输出小写 hex
func hmacSign(secretKey, timestamp, nonce, path string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(nonce))
	mac.Write([]byte(path))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// userStatusDeny 校验登录用户是否启用且未过期；通过返回 nil，否则返回标准 Response 载荷
func userStatusDeny(acc mysql.User_type) []any {
	if acc.Enable != 1 {
		return []any{403, "账号已禁用"}
	}
	if !acc.Expire_At.IsZero() && time.Now().After(acc.Expire_At) {
		return []any{403, "账号已过期"}
	}
	return nil
}

// apiKeyCredsDeny 基于缓存凭据校验 AK/SK 是否启用且未过期；通过返回 nil，否则返回标准 Response 载荷
func apiKeyCredsDeny(c redis.Auth_ApiKey_Creds) []any {
	if c.Enable != 1 {
		return []any{403, "账号已禁用"}
	}
	if c.ExpireAt != 0 && time.Now().Unix() > c.ExpireAt {
		return []any{403, "账号已过期"}
	}
	return nil
}

// apiKeyToCreds 将 api_key 记录转为 Redis 缓存结构（到期时间以 Unix 秒存，零值=永不过期）
func apiKeyToCreds(acc mysql.ApiKey_type) redis.Auth_ApiKey_Creds {
	var exp int64
	if !acc.Expire_At.IsZero() {
		exp = acc.Expire_At.Unix()
	}
	return redis.Auth_ApiKey_Creds{Id: acc.Id, Secret: acc.Secret_Key, Enable: acc.Enable, ExpireAt: exp}
}

// loadApiKeyCreds 取 AK/SK 验签凭据：优先读 Redis 缓存；未命中或 Redis 故障时回源 MySQL 并写回缓存（cache-aside）
// found=false 表示 access_key 无效（MySQL 也不存在）；err!=nil 表示 MySQL 查询失败
func loadApiKeyCreds(accessKey string) (redis.Auth_ApiKey_Creds, bool, error) {
	if c, ok, err := redis.Auth_ApiKey_Creds_Get(accessKey); err == nil && ok {
		return c, true, nil
	}
	acc, found, err := mysql.ApiKey__Query_By_Access_Key(accessKey)
	if err != nil || !found {
		return redis.Auth_ApiKey_Creds{}, found, err
	}
	c := apiKeyToCreds(acc)
	_ = redis.Auth_ApiKey_Creds_Set(accessKey, c, ApiKey_Creds_TTL) // 写失败忽略：不影响本次验签
	return c, true, nil
}

// refreshApiKeyCreds 在 AK/SK 增改/重置/同步后回源 MySQL 刷新 Redis 验签缓存；账号已不存在则清缓存
func refreshApiKeyCreds(accessKey string) {
	acc, found, err := mysql.ApiKey__Query_By_Access_Key(accessKey)
	if err != nil || !found {
		_ = redis.Auth_ApiKey_Creds_Del(accessKey)
		return
	}
	_ = redis.Auth_ApiKey_Creds_Set(accessKey, apiKeyToCreds(acc), ApiKey_Creds_TTL)
}
