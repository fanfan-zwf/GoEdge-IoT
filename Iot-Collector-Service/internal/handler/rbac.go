/*
* 日期: 2026.11.20
* 作用: RBAC 权限校验接入层（合并自原 rbac_middleware / rbac_instance / rbac_routes / rbac_scope_ctx）。
*       紧随统一鉴权之后由 Rbac_Middleware 承接：读取鉴权写入上下文的主体身份
*       （Log_Source_User→业务用户 id / Log_Source_AKSK→API 客户端 id），把请求路径映射为
*       资源主题后调用校验入口。恒拦截：判定拒绝一律 403（校验自身故障 520 fail-closed）、缺主体 401。
*       实例级端点（驱动/点位/报警/历史 config + alarm-data + 实时读写值/WS）做集中式 id 授权：
*       缓冲并解析 body → 计算授权 id 集合 → 经 ctx 下发给 handler 过滤。详见 internal/handler/rbac_guide.md。
 */

package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strconv"
	"strings"

	"collector/internal/repo/mysql"
	rbacsvc "collector/internal/service/rbac"
	rbacmatch "collector/pkg/rbac"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 路由映射：把 HTTP 请求解析为资源主题
// ============================================================

// ApiPrefix 业务 API 统一前缀；资源主题由此前缀之后的路径段构成。
const ApiPrefix = "/api/v1.0"

// resourceTopic 由请求路径得到资源主题，如 "/api/v1.0/config/drive/query" → "/config/drive/query"。
func resourceTopic(path string) string {
	return strings.TrimPrefix(path, ApiPrefix)
}

// actionFromPath 依据路径动词推断动作。判断顺序体现优先级（先匹配更具体的动词）。
// 注：action 维度已废弃，仅作日志展示，不参与 RBAC 匹配。
func actionFromPath(path string) string {
	switch {
	case strings.Contains(path, "/export"):
		return "export"
	case strings.Contains(path, "/del"):
		return "delete"
	// 写值/写配置/重启/刷新/导入/重建/重置密钥/集合替换(set)/增改 → edit（read 必须先于通用判断，read 归 view）
	case strings.Contains(path, "/read"):
		return "view"
	case containsAny(path, "/write", "/import", "/restart", "/reload", "/rebuild", "/reset-secret", "/set", "/add", "/update"):
		return "edit"
	case containsAny(path, "/query", "/count", "/search", "/children", "/points", "/status", "/info", "/ws"):
		return "view"
	default:
		// 兜底按 view：未知接口不自动放行，需显式授予 view（或 *）权限
		return "view"
	}
}

// containsAny 判断 s 是否包含子串列表中的任意一个。
func containsAny(s string, subs ...string) bool {
	for _, s2 := range subs {
		if strings.Contains(s, s2) {
			return true
		}
	}
	return false
}

// ============================================================
// 权限校验中间件（须注册在 Auth_Middleware 之后，依赖其写入的主体上下文）
// ============================================================

// Rbac_Middleware 权限校验中间件。
func Rbac_Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		// 登录/登出/刷新/内部同步/说明页等免鉴权路径同时免权限校验
		if authExemptPath(path) || !strings.HasPrefix(path, ApiPrefix) {
			ctx.Next()
			return
		}

		// 主体身份由鉴权中间件写入；拿不到说明异常，兜底拒绝
		srcVal, hasSrc := ctx.Get(CtxKeyLogSource)
		idVal, hasId := ctx.Get(CtxKeyLogAccount)
		if !hasSrc || !hasId {
			ctx.Set("Response", []any{401, "缺少鉴权主体"})
			ctx.Abort()
			return
		}
		src := srcVal.(mysql.Log_Source)
		subjectId := idVal.(uint)

		resource := resourceTopic(path)

		// 实例级端点（驱动/点位/报警/历史 config + alarm-data 的读写）走集中式 id 授权：
		// 缓冲 body、算授权集合、经 ctx 下发；handled=true 时不再走下方接口级逻辑。
		if rbacInstanceMiddleware(ctx, resource, subjectTypeFromSrc(src), subjectId) {
			return
		}

		action := actionFromPath(path)

		var (
			d   rbacmatch.Decision
			err error
		)
		if src == mysql.Log_Source_User {
			d, err = rbacsvc.CheckBizUser(subjectId, resource, action, nil)
		} else {
			d, err = rbacsvc.CheckApiClient(subjectId, resource, action, nil)
		}

		if err != nil {
			// 校验自身故障：从严拒绝（fail-closed）
			log.Printf("ERROR RBAC 校验异常 | src=%s id=%d | %s %s | err=%v", src, subjectId, action, resource, err)
			ctx.Set("Response", []any{520, "权限校验暂时不可用"})
			ctx.Abort()
			return
		}

		if !d.Allowed {
			log.Printf("WARN RBAC 拒绝 | src=%s id=%d | %s %s | reason=%s", src, subjectId, action, resource, d.Reason)
			ctx.Set("Response", []any{403, "无权限访问该资源"})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// ============================================================
// 实例级（接口 + 资源行 id）授权
//   读(query/count/search 合并) 基名 /config/{res}/query，实例段 {id} 默认 *；
//   写 add/update/del/restart 基名 /config/{res}/{verb}，实例段 {id} 默认 *；
//   add 新建无行 id → 仅通配 /add/* 放行（Can(base+"/0") 探测）；
//   批量 update/del → 逐 id 判定，只操作有权限者；全无权限 → 403；restart → 单 id，无权直接 403。
// ============================================================

// ctx 下发键：列表读授权 id 集合（[]uint，nil/缺失=全量不过滤）；批量写可操作子集（[]uint）。
const (
	CtxKeyRbacScopeIds = "Rbac_Scope_Ids"
	CtxKeyRbacOpIds    = "Rbac_Op_Ids"
)

// instanceKind 实例级端点的处理类型。
type instanceKind string

const (
	kindList       instanceKind = "list"       // query/count/search：算可见集合，注入 scope
	kindListGate   instanceKind = "listGate"   // alarm-data query/count：仅通配读权限门禁，不做行过滤
	kindAdd        instanceKind = "add"        // add：要求通配放行
	kindUpdate     instanceKind = "update"     // update：body 为对象数组(含 Id)，逐 id 求子集
	kindDel        instanceKind = "del"        // del：body 为 {ids:[]}，逐 id 求子集
	kindRestart    instanceKind = "restart"    // restart：body 单 id，无权 403
	kindValueDefer instanceKind = "valueDefer" // 实时点位读值/写值/WS：接口级放行(不整单 403)，由 handler 按解析出的点位 id 逐个过滤
)

// instancePolicy 单个实例级端点的策略。
type instancePolicy struct {
	kind   instanceKind
	base   string // id 后缀拼接的基名，如 /config/drive/query（写操作为其自身动词）
	action string // 仅用于日志（action 维度已退化，不参与匹配）
	table  string // 列表枚举候选 id 的来源表（kindList 必填）
}

// instancePolicies 键为去掉 ApiPrefix 后的请求主题。合并读权限使 count/search 的 base 指向 query。
var instancePolicies = map[string]instancePolicy{
	// —— 驱动 ——
	"/config/drive/query":   {kindList, "/config/drive/query", "view", "drive"},
	"/config/drive/count":   {kindList, "/config/drive/query", "view", "drive"},
	"/config/drive/search":  {kindList, "/config/drive/query", "view", "drive"},
	"/config/drive/add":     {kindAdd, "/config/drive/add", "edit", ""},
	"/config/drive/update":  {kindUpdate, "/config/drive/update", "edit", ""},
	"/config/drive/del":     {kindDel, "/config/drive/del", "delete", ""},
	"/config/drive/restart": {kindRestart, "/config/drive/restart", "edit", ""},
	// —— 点位 ——
	"/config/point/query":  {kindList, "/config/point/query", "view", "point"},
	"/config/point/count":  {kindList, "/config/point/query", "view", "point"},
	"/config/point/search": {kindList, "/config/point/query", "view", "point"},
	"/config/point/add":    {kindAdd, "/config/point/add", "edit", ""},
	"/config/point/update": {kindUpdate, "/config/point/update", "edit", ""},
	"/config/point/del":    {kindDel, "/config/point/del", "delete", ""},
	// —— 报警配置 ——
	"/config/alarm/query":  {kindList, "/config/alarm/query", "view", "alarm"},
	"/config/alarm/count":  {kindList, "/config/alarm/query", "view", "alarm"},
	"/config/alarm/search": {kindList, "/config/alarm/query", "view", "alarm"},
	"/config/alarm/add":    {kindAdd, "/config/alarm/add", "edit", ""},
	"/config/alarm/update": {kindUpdate, "/config/alarm/update", "edit", ""},
	"/config/alarm/del":    {kindDel, "/config/alarm/del", "delete", ""},
	// —— 报警数据（仅读，通配门禁，不做行级过滤）——
	"/config/alarm-data/query": {kindListGate, "/config/alarm-data/query", "view", ""},
	"/config/alarm-data/count": {kindListGate, "/config/alarm-data/query", "view", ""},
	// —— 历史配置 ——
	"/config/history/query":  {kindList, "/config/history/query", "view", "history"},
	"/config/history/count":  {kindList, "/config/history/query", "view", "history"},
	"/config/history/add":    {kindAdd, "/config/history/add", "edit", ""},
	"/config/history/update": {kindUpdate, "/config/history/update", "edit", ""},
	"/config/history/del":    {kindDel, "/config/history/del", "delete", ""},
	// —— 实时点位读写值 / WS 订阅（接口级放行，逐 id 过滤交由 point.go handler）——
	//    读值与 WS 共用 /point/read/{id}，写值用 /point/write/{id}；base 仅供日志，实际过滤在 handler。
	"/point/read/value":  {kindValueDefer, "/point/read", "view", ""},
	"/point/ws":          {kindValueDefer, "/point/read", "view", ""},
	"/point/write/value": {kindValueDefer, "/point/write", "edit", ""},
}

// allIdsForTable 按表名取全部行 Id（列表枚举候选）。
func allIdsForTable(table string) ([]uint, error) {
	switch table {
	case "drive":
		return mysql.Drive_Config__AllIds()
	case "point":
		return mysql.Point_Config__AllIds()
	case "alarm":
		return mysql.Alarm_Config__AllIds()
	case "history":
		return mysql.History_Config__AllIds()
	}
	return nil, nil
}

// parseBodyIds 依 kind 从原始 body 解析待操作 id（update: 对象数组 Id；del: {ids:[]}; restart: 单 id）。
func parseBodyIds(kind instanceKind, raw []byte) []uint {
	if len(raw) == 0 {
		return nil
	}
	switch kind {
	case kindUpdate:
		var rows []struct{ Id uint }
		if err := json.Unmarshal(raw, &rows); err == nil {
			ids := make([]uint, 0, len(rows))
			for _, r := range rows {
				if r.Id > 0 {
					ids = append(ids, r.Id)
				}
			}
			return ids
		}
	case kindDel:
		var b struct{ Ids []uint }
		if err := json.Unmarshal(raw, &b); err == nil {
			ids := make([]uint, 0, len(b.Ids))
			for _, id := range b.Ids {
				if id > 0 {
					ids = append(ids, id)
				}
			}
			return ids
		}
	case kindRestart:
		var b struct{ Id uint }
		if err := json.Unmarshal(raw, &b); err == nil && b.Id > 0 {
			return []uint{b.Id}
		}
	}
	return nil
}

// rbacInstanceCheck 处理实例级端点：判定放行/拒绝，并在放行时把授权 id 集合写入 ctx。
// 返回 deny=true 表示应拒绝；denyStatus/denyMsg 为拒绝时下发给统一响应中间件的内容。
func rbacInstanceCheck(ctx *gin.Context, p instancePolicy, subjectType uint64, subjectId uint, raw []byte) (deny bool, denyStatus int, denyMsg string) {
	switch p.kind {
	case kindList:
		if rbacsvc.ScopeAll(subjectType, subjectId, p.base, p.action) {
			return false, 0, "" // 全量可见：不注入 scope
		}
		ids, err := allIdsForTable(p.table)
		if err != nil {
			log.Printf("ERROR RBAC 实例级枚举失败 | table=%s id=%d | err=%v", p.table, subjectId, err)
			return true, 520, "权限校验暂时不可用"
		}
		allowed := rbacsvc.FilterIds(subjectType, subjectId, p.base, p.action, ids)
		if len(allowed) == 0 {
			return true, 403, "无权限访问该资源"
		}
		ctx.Set(CtxKeyRbacScopeIds, allowed)
		return false, 0, ""

	case kindListGate:
		// alarm-data：仅接受通配读权限（未实现行级过滤），否则拒绝
		if !rbacsvc.ScopeAll(subjectType, subjectId, p.base, p.action) {
			return true, 403, "无权限访问该资源"
		}
		return false, 0, ""

	case kindAdd:
		// 新建无行 id：仅当持有通配 /add/* 放行（哨兵 0 只可能被宽规则命中）
		if !rbacsvc.Can(subjectType, subjectId, p.base+"/0", p.action) {
			return true, 403, "无权限执行该操作"
		}
		return false, 0, ""

	case kindUpdate, kindDel:
		ids := parseBodyIds(p.kind, raw)
		if len(ids) == 0 {
			return false, 0, "" // 无有效 id：交 handler 按原逻辑处理（多为空操作/参数错误）
		}
		if rbacsvc.ScopeAll(subjectType, subjectId, p.base, p.action) {
			ctx.Set(CtxKeyRbacOpIds, ids)
			return false, 0, ""
		}
		subset := rbacsvc.FilterIds(subjectType, subjectId, p.base, p.action, ids)
		if len(subset) == 0 {
			return true, 403, "无权限操作所选资源"
		}
		ctx.Set(CtxKeyRbacOpIds, subset)
		return false, 0, ""

	case kindRestart:
		ids := parseBodyIds(p.kind, raw)
		if len(ids) == 0 {
			return false, 0, "" // 缺 id：交 handler 参数校验
		}
		if !rbacsvc.Can(subjectType, subjectId, p.base+"/"+strconv.FormatUint(uint64(ids[0]), 10), p.action) {
			return true, 403, "无权限执行该操作"
		}
		return false, 0, ""

	case kindValueDefer:
		// 实时读写值/WS：接口级不拦截（否则只有显式 /point/read/{id} 的主体会在 .../value 段被提前 403），
		// 交由 handler 把 body/query 解析成点位 id 后逐个判权并过滤（见 point.go）。
		return false, 0, ""
	}
	return false, 0, ""
}

// rbacInstanceMiddleware 对实例级端点做集中判定；命中策略时缓冲 body 供解析后原样回灌给 handler。
// 返回 handled=true 表示该请求属实例级端点且已处理（可能已 Abort）；handled=false 时交回接口级逻辑。
func rbacInstanceMiddleware(ctx *gin.Context, resource string, subjectType uint64, subjectId uint) (handled bool) {
	p, ok := instancePolicies[resource]
	if !ok {
		return false
	}

	// 缓冲并回灌 body：保证 handler 仍可正常 BindJSON / ShouldBindJSON
	var raw []byte
	if ctx.Request.Body != nil {
		raw, _ = io.ReadAll(ctx.Request.Body)
		_ = ctx.Request.Body.Close()
		ctx.Request.Body = io.NopCloser(bytes.NewReader(raw))
	}

	deny, status, msg := rbacInstanceCheck(ctx, p, subjectType, subjectId, raw)
	if deny {
		// 恒拦截：无权限一律按判定状态（403/520）返回并中断
		log.Printf("WARN RBAC(实例级) 拒绝 | type=%d id=%d | %s %s | reason=%s", subjectType, subjectId, p.action, resource, msg)
		ctx.Set("Response", []any{status, msg})
		ctx.Abort()
		return true
	}
	ctx.Next()
	return true
}

// subjectTypeFromSrc 把鉴权主体来源映射为 RBAC 主体类型。
func subjectTypeFromSrc(src mysql.Log_Source) uint64 {
	if src == mysql.Log_Source_User {
		return mysql.Rbac_Subject_Biz
	}
	return mysql.Rbac_Subject_Api
}

// ============================================================
// handler 侧读取实例级中间件经 ctx 下发的授权集合
// ============================================================

// scopeIdsFromCtx 读取列表读的授权 id 集合（透传给 repo 的 idScope）。
// 缺失/nil = 全量不过滤（ScopeAll 命中、超管、或灰度观测模式）。
func scopeIdsFromCtx(ctx *gin.Context) []uint {
	if v, ok := ctx.Get(CtxKeyRbacScopeIds); ok {
		if ids, ok := v.([]uint); ok {
			return ids
		}
	}
	return nil
}

// opIdsFromCtx 读取批量 update/del 的可操作 id 子集。
// present=false 表示未做实例级限制（超管/通配/灰度观测），handler 按原始入参全量处理。
func opIdsFromCtx(ctx *gin.Context) (ids []uint, present bool) {
	if v, ok := ctx.Get(CtxKeyRbacOpIds); ok {
		if s, ok := v.([]uint); ok {
			return s, true
		}
	}
	return nil, false
}

// idSet 构造 id 查找集合。
func idSet(ids []uint) map[uint]bool {
	m := make(map[uint]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}
