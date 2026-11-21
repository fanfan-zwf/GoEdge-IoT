/*
* 日期: 2026.11.19
* 作用: 极简 RBAC 管理端 API——角色(业务/API)、权限字典、以及三类绑定
*       (角色↔权限、主体↔角色、主体↔权限直授) 的增删改查。
*       受统一鉴权 + RBAC 中间件保护（/config/rbac/*）；写操作后权限判定结果缓存靠 TTL 自然过期（无主动失效）。
*       响应沿用全站范式 ctx.Set("Response", []any{code, msg[, data]})。
 */
package handler

import (
	"collector/internal/repo/mysql"
	rbacmatch "collector/pkg/rbac"
	"strings"

	"github.com/gin-gonic/gin"
)

// ======== 入参结构 ========

type rbacRoleReq struct {
	RoleType uint64 `json:"role_type"` // 1业务 2API
	Id       uint64 `json:"id"`
	RoleCode string `json:"role_code"`
	RoleName string `json:"role_name"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"` // 1启用 0禁用
	Remark   string `json:"remark"`
	Keyword  string `json:"keyword"` // 查询：按角色编码/名称模糊匹配
	Page     uint   `json:"page"`
	PageSize uint   `json:"pageSize"`
}

type rbacPermReq struct {
	Id        uint64 `json:"id"`
	PermCode  string `json:"perm_code"`
	TopicKey  string `json:"topic_key"`
	MatchSort int    `json:"match_sort"`
	Effect    int    `json:"effect"` // 1allow 0deny
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
	Keyword   string `json:"keyword"` // 查询：按权限编码/资源主题模糊匹配
	Page      uint   `json:"page"`
	PageSize  uint   `json:"pageSize"`
}

// rbacBindReq 三种集合替换共用的入参：主体/角色定位 + 目标 id 集合。
type rbacBindReq struct {
	SubjectType uint64   `json:"subject_type"` // 1业务用户 2API客户端
	SubjectId   uint64   `json:"subject_id"`
	RoleType    uint64   `json:"role_type"` // 1业务角色 2API角色
	RoleId      uint64   `json:"role_id"`
	Ids         []uint64 `json:"ids"` // 要设定的 role_id 或 perm_id 集合
}

// ======== 小工具 ========

// rbacTypeOK 角色/主体类型仅允许 1(业务) 或 2(API)。
func rbacTypeOK(t uint64) bool { return t == mysql.Rbac_Subject_Biz || t == mysql.Rbac_Subject_Api }

// rbacDupErr 把唯一索引冲突转成友好提示，其余原样返回 520。
func rbacDupErr(err error) []any {
	if strings.Contains(err.Error(), "Duplicate entry") {
		return []any{417, "编码已存在，请更换后重试"}
	}
	return []any{520, err.Error()}
}

// ======== 角色 ========

func api_rbac_role_query(ctx *gin.Context) {
	var req rbacRoleReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.RoleType) {
		ctx.Set("Response", []any{417, "role_type 只能为 1(业务) 或 2(API)"})
		return
	}
	list, err := mysql.RbacRole__Query(req.RoleType, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", rbacDupErr(err))
		return
	}
	ctx.Set("Response", []any{200, "ok", list})
}

func api_rbac_role_count(ctx *gin.Context) {
	var req rbacRoleReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.RoleType) {
		ctx.Set("Response", []any{417, "role_type 非法"})
		return
	}
	n, err := mysql.RbacRole__Count(req.RoleType, req.Keyword)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", n})
}

func api_rbac_role_add(ctx *gin.Context) {
	var req rbacRoleReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.RoleType) || req.RoleCode == "" {
		ctx.Set("Response", []any{417, "role_type 非法或角色编码为空"})
		return
	}
	id, err := mysql.RbacRole__Add(req.RoleType, req.RoleCode, req.RoleName, req.Sort, req.Status, req.Remark)
	if err != nil {
		ctx.Set("Response", rbacDupErr(err))
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Add, gin.H{"table": "rbac_role", "role_type": req.RoleType, "id": id, "role_code": req.RoleCode})
	ctx.Set("Response", []any{200, "ok", gin.H{"id": id}})
}

func api_rbac_role_update(ctx *gin.Context) {
	var req rbacRoleReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.RoleType) || req.Id == 0 || req.RoleCode == "" {
		ctx.Set("Response", []any{417, "参数不合法"})
		return
	}
	if err := mysql.RbacRole__Update(req.RoleType, req.Id, req.RoleCode, req.RoleName, req.Sort, req.Status, req.Remark); err != nil {
		ctx.Set("Response", rbacDupErr(err))
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Update, gin.H{"table": "rbac_role", "role_type": req.RoleType, "id": req.Id, "role_code": req.RoleCode})
	ctx.Set("Response", []any{200, "ok"})
}

func api_rbac_role_del(ctx *gin.Context) {
	var req rbacRoleReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.RoleType) || req.Id == 0 {
		ctx.Set("Response", []any{417, "参数不合法"})
		return
	}
	if err := mysql.RbacRole__Del(req.RoleType, req.Id); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Del, gin.H{"table": "rbac_role", "role_type": req.RoleType, "id": req.Id})
	ctx.Set("Response", []any{200, "ok"})
}

// ======== 权限字典 ========

func api_rbac_perm_query(ctx *gin.Context) {
	var req rbacPermReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	list, err := mysql.RbacPerm__Query(req.Keyword, req.Page, req.PageSize)
	if err != nil {
		ctx.Set("Response", rbacDupErr(err))
		return
	}
	ctx.Set("Response", []any{200, "ok", list})
}

func api_rbac_perm_count(ctx *gin.Context) {
	var req rbacPermReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	n, err := mysql.RbacPerm__Count(req.Keyword)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", n})
}

// validPermReq 校验权限主题合法性（# 仅段尾、+ 独占段等），返回错误提示或 nil。
func validPermReq(req *rbacPermReq) string {
	if req.PermCode == "" || req.TopicKey == "" {
		return "权限编码与主题不能为空"
	}
	if err := rbacmatch.ValidatePattern(req.TopicKey); err != nil {
		return "主题格式非法：" + err.Error()
	}
	return ""
}

func api_rbac_perm_add(ctx *gin.Context) {
	var req rbacPermReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if msg := validPermReq(&req); msg != "" {
		ctx.Set("Response", []any{417, msg})
		return
	}
	id, err := mysql.RbacPerm__Add(req.PermCode, req.TopicKey, req.MatchSort, req.Effect, req.Status, req.Remark)
	if err != nil {
		ctx.Set("Response", rbacDupErr(err))
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Add, gin.H{"table": "t_permission", "id": id, "perm_code": req.PermCode, "topic_key": req.TopicKey})
	ctx.Set("Response", []any{200, "ok", gin.H{"id": id}})
}

func api_rbac_perm_update(ctx *gin.Context) {
	var req rbacPermReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.Id == 0 {
		ctx.Set("Response", []any{417, "id 不能为空"})
		return
	}
	if msg := validPermReq(&req); msg != "" {
		ctx.Set("Response", []any{417, msg})
		return
	}
	if err := mysql.RbacPerm__Update(req.Id, req.PermCode, req.TopicKey, req.MatchSort, req.Effect, req.Status, req.Remark); err != nil {
		ctx.Set("Response", rbacDupErr(err))
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Update, gin.H{"table": "t_permission", "id": req.Id, "perm_code": req.PermCode})
	ctx.Set("Response", []any{200, "ok"})
}

func api_rbac_perm_del(ctx *gin.Context) {
	var req rbacPermReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if req.Id == 0 {
		ctx.Set("Response", []any{417, "id 不能为空"})
		return
	}
	if err := mysql.RbacPerm__Del(req.Id); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Del, gin.H{"table": "t_permission", "id": req.Id})
	ctx.Set("Response", []any{200, "ok"})
}

// ======== 绑定：角色↔权限、主体↔角色、主体↔权限直授 ========

func api_rbac_role_perms(ctx *gin.Context) {
	var req rbacBindReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.RoleType) || req.RoleId == 0 {
		ctx.Set("Response", []any{417, "参数不合法"})
		return
	}
	ids, err := mysql.RolePerm__List(req.RoleType, req.RoleId)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", gin.H{"ids": ids}})
}

func api_rbac_role_perms_set(ctx *gin.Context) {
	var req rbacBindReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.RoleType) || req.RoleId == 0 {
		ctx.Set("Response", []any{417, "参数不合法"})
		return
	}
	if err := mysql.RolePerm__Replace(req.RoleType, req.RoleId, req.Ids); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Update, gin.H{"table": "t_role_permission", "role_type": req.RoleType, "role_id": req.RoleId, "perm_ids": req.Ids})
	ctx.Set("Response", []any{200, "ok"})
}

func api_rbac_subject_roles(ctx *gin.Context) {
	var req rbacBindReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.SubjectType) || req.SubjectId == 0 {
		ctx.Set("Response", []any{417, "参数不合法"})
		return
	}
	ids, err := mysql.SubjectRole__List(req.SubjectType, req.SubjectId)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", gin.H{"ids": ids}})
}

func api_rbac_subject_roles_set(ctx *gin.Context) {
	var req rbacBindReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.SubjectType) || req.SubjectId == 0 {
		ctx.Set("Response", []any{417, "参数不合法"})
		return
	}
	if err := mysql.SubjectRole__Replace(req.SubjectType, req.SubjectId, req.Ids); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Update, gin.H{"table": "rbac_subject_role", "subject_type": req.SubjectType, "subject_id": req.SubjectId, "role_ids": req.Ids})
	ctx.Set("Response", []any{200, "ok"})
}

func api_rbac_subject_perms(ctx *gin.Context) {
	var req rbacBindReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.SubjectType) || req.SubjectId == 0 {
		ctx.Set("Response", []any{417, "参数不合法"})
		return
	}
	ids, err := mysql.SubjectPerm__List(req.SubjectType, req.SubjectId)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", gin.H{"ids": ids}})
}

func api_rbac_subject_perms_set(ctx *gin.Context) {
	var req rbacBindReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if !rbacTypeOK(req.SubjectType) || req.SubjectId == 0 {
		ctx.Set("Response", []any{417, "参数不合法"})
		return
	}
	if err := mysql.SubjectPerm__Replace(req.SubjectType, req.SubjectId, req.Ids); err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	logOperate(ctx, mysql.Log_Type_Account, mysql.Log_Operate_Update, gin.H{"table": "t_direct_permission", "subject_type": req.SubjectType, "subject_id": req.SubjectId, "perm_ids": req.Ids})
	ctx.Set("Response", []any{200, "ok"})
}

// rbac_admin_api_register 注册 RBAC 管理端路由（受统一鉴权 + RBAC 中间件保护）。
func rbac_admin_api_register(r *gin.Engine) {
	g := r.Group("/api/v1.0/config/rbac")
	{
		// 角色
		g.POST("/role/query", api_rbac_role_query)
		g.POST("/role/count", api_rbac_role_count)
		g.POST("/role/add", api_rbac_role_add)
		g.POST("/role/update", api_rbac_role_update)
		g.POST("/role/del", api_rbac_role_del)
		// 权限字典
		g.POST("/perm/query", api_rbac_perm_query)
		g.POST("/perm/count", api_rbac_perm_count)
		g.POST("/perm/add", api_rbac_perm_add)
		g.POST("/perm/update", api_rbac_perm_update)
		g.POST("/perm/del", api_rbac_perm_del)
		// 绑定
		g.POST("/role/perms", api_rbac_role_perms)
		g.POST("/role/perms/set", api_rbac_role_perms_set)
		g.POST("/subject/roles", api_rbac_subject_roles)
		g.POST("/subject/roles/set", api_rbac_subject_roles_set)
		g.POST("/subject/perms", api_rbac_subject_perms)
		g.POST("/subject/perms/set", api_rbac_subject_perms_set)
	}
}
