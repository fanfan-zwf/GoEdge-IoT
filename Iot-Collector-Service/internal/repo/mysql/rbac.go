/*
* 日期: 2026.11.19
* 作用: 极简 RBAC 数据访问层（纯 SQL，无业务判定）。
*       主体复用 user_account(业务,id 即 user_account.Id) 与 api_key(API,id 即 api_key.Id)。
*       加载某主体的有效权限规则：主体 → 启用未过期角色 → 启用未过期角色权限 → 启用权限主表，
*       再并上直接授权；过滤条件（status/is_enabled/expire_at）全部在 SQL 完成。
*       所有查询参数化，业务/API 两条链路用固定两套表名分支，绝不拼接用户输入。
 */

package mysql

import (
	"collector/pkg/rbac"
	"database/sql"
	"fmt"
)

// RBAC 主体/角色类型：与 t_role_permission.role_type、t_direct_permission.subject_type 取值一致。
// 业务与 API 两套角色分表存放，仅靠该类型区分，权限主表 t_permission 全局复用。
const (
	Rbac_Subject_Biz uint64 = 1 // 业务用户 / 业务角色
	Rbac_Subject_Api uint64 = 2 // API 客户端 / API 角色
)

// 权限效果
const (
	Rbac_Effect_Allow int = 1 // allow
	Rbac_Effect_Deny  int = 0 // deny
)

// bizRuleQuery 加载业务用户有效规则：角色授权 UNION 直接授权（subject_type=1）。
const bizRuleQuery = `
	SELECT p.id, p.perm_code, p.topic_key, p.match_sort, p.effect, r.role_code
	FROM t_biz_user_role ur
	JOIN t_biz_role r        ON r.id = ur.role_id AND r.status = 1
	JOIN t_role_permission rp ON rp.role_type = 1 AND rp.role_id = r.id
	     AND rp.is_enabled = 1 AND (rp.expire_at IS NULL OR rp.expire_at > NOW())
	JOIN t_permission p      ON p.id = rp.perm_id AND p.status = 1
	WHERE ur.user_id = ? AND ur.is_enabled = 1 AND (ur.expire_at IS NULL OR ur.expire_at > NOW())
	UNION
	SELECT p.id, p.perm_code, p.topic_key, p.match_sort, p.effect, 'direct' AS role_code
	FROM t_direct_permission dp
	JOIN t_permission p ON p.id = dp.perm_id AND p.status = 1
	WHERE dp.subject_type = 1 AND dp.subject_id = ? AND dp.is_enabled = 1
	     AND (dp.expire_at IS NULL OR dp.expire_at > NOW())
`

// apiRuleQuery 加载 API 客户端有效规则：角色授权 UNION 直接授权（subject_type=2）。
const apiRuleQuery = `
	SELECT p.id, p.perm_code, p.topic_key, p.match_sort, p.effect, r.role_code
	FROM t_api_client_role cr
	JOIN t_api_role r        ON r.id = cr.role_id AND r.status = 1
	JOIN t_role_permission rp ON rp.role_type = 2 AND rp.role_id = r.id
	     AND rp.is_enabled = 1 AND (rp.expire_at IS NULL OR rp.expire_at > NOW())
	JOIN t_permission p      ON p.id = rp.perm_id AND p.status = 1
	WHERE cr.client_id = ? AND cr.is_enabled = 1 AND (cr.expire_at IS NULL OR cr.expire_at > NOW())
	UNION
	SELECT p.id, p.perm_code, p.topic_key, p.match_sort, p.effect, 'direct' AS role_code
	FROM t_direct_permission dp
	JOIN t_permission p ON p.id = dp.perm_id AND p.status = 1
	WHERE dp.subject_type = 2 AND dp.subject_id = ? AND dp.is_enabled = 1
	     AND (dp.expire_at IS NULL OR dp.expire_at > NOW())
`

// scanRules 把查询行转为应用层规则集合，effect→Allow。
func scanRules(rows *sql.Rows) ([]rbac.Rule, error) {
	var rules []rbac.Rule
	for rows.Next() {
		var (
			ru     rbac.Rule
			effect int
		)
		if err := rows.Scan(&ru.PermId, &ru.PermCode, &ru.TopicKey, &ru.MatchSort, &effect, &ru.Role); err != nil {
			return nil, err
		}
		ru.Action = "*" // 动作维度已并入主题，规则不再单列动作，统一 "*" 命中任意动作
		ru.Allow = effect == Rbac_Effect_Allow
		rules = append(rules, ru)
	}
	return rules, rows.Err()
}

// LoadEffectiveRules 按主体类型加载其全部有效权限规则（含角色授权与直接授权，已过滤启用/过期）。
// 传入：subjectType — Rbac_Subject_Biz / Rbac_Subject_Api；subjectId — user_account.Id 或 api_key.Id。
func LoadEffectiveRules(subjectType, subjectId uint64) (rules []rbac.Rule, err error) {
	var query string
	switch subjectType {
	case Rbac_Subject_Biz:
		query = bizRuleQuery
	case Rbac_Subject_Api:
		query = apiRuleQuery
	default:
		return nil, fmt.Errorf("LoadEffectiveRules: 非法主体类型 %d", subjectType)
	}
	rows, err := DB.Query(query, subjectId, subjectId)
	if err != nil {
		return nil, fmt.Errorf("[LoadEffectiveRules] 查询失败 | type=%d id=%d | err=%w", subjectType, subjectId, err)
	}
	defer rows.Close()
	return scanRules(rows)
}

// bizRoleCodeQuery / apiRoleCodeQuery 取主体已启用未过期的角色编码（登录时回填前端 roles，用于菜单可见性）。
const bizRoleCodeQuery = `
	SELECT r.role_code
	FROM t_biz_user_role ur
	JOIN t_biz_role r ON r.id = ur.role_id AND r.status = 1
	WHERE ur.user_id = ? AND ur.is_enabled = 1 AND (ur.expire_at IS NULL OR ur.expire_at > NOW())
`
const apiRoleCodeQuery = `
	SELECT r.role_code
	FROM t_api_client_role cr
	JOIN t_api_role r ON r.id = cr.role_id AND r.status = 1
	WHERE cr.client_id = ? AND cr.is_enabled = 1 AND (cr.expire_at IS NULL OR cr.expire_at > NOW())
`

// LoadRoleCodes 按主体类型加载其有效角色编码集合。
func LoadRoleCodes(subjectType, subjectId uint64) (codes []string, err error) {
	var query string
	switch subjectType {
	case Rbac_Subject_Biz:
		query = bizRoleCodeQuery
	case Rbac_Subject_Api:
		query = apiRoleCodeQuery
	default:
		return nil, fmt.Errorf("LoadRoleCodes: 非法主体类型 %d", subjectType)
	}
	rows, err := DB.Query(query, subjectId)
	if err != nil {
		return nil, fmt.Errorf("[LoadRoleCodes] 查询失败 | type=%d id=%d | err=%w", subjectType, subjectId, err)
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		codes = append(codes, c)
	}
	return codes, rows.Err()
}

// ============================================================
// 写操作（供示例 / 管理端调用；调用后需清除权限缓存，见 service/rbac）
// 唯一索引已建，授权/绑定用 ON DUPLICATE KEY 复启用，天然幂等。
// ============================================================

// BizRole__Add 新增业务角色，返回自增 id。
func BizRole__Add(roleCode, roleName string, sort int) (uint64, error) {
	res, err := DB.Exec(`INSERT INTO t_biz_role (role_code, role_name, sort) VALUES (?, ?, ?)`, roleCode, roleName, sort)
	if err != nil {
		return 0, fmt.Errorf("BizRole__Add 失败 | role_code=%s | err=%w", roleCode, err)
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

// ApiRole__Add 新增 API 角色，返回自增 id。
func ApiRole__Add(roleCode, roleName string, sort int) (uint64, error) {
	res, err := DB.Exec(`INSERT INTO t_api_role (role_code, role_name, sort) VALUES (?, ?, ?)`, roleCode, roleName, sort)
	if err != nil {
		return 0, fmt.Errorf("ApiRole__Add 失败 | role_code=%s | err=%w", roleCode, err)
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

// Permission__Add 新增权限主表记录，返回自增 id。effect: 1=allow 0=deny。
func Permission__Add(permCode, topicKey string, matchSort, effect int) (uint64, error) {
	if err := rbac.ValidatePattern(topicKey); err != nil {
		return 0, fmt.Errorf("Permission__Add 主题非法: %w", err)
	}
	res, err := DB.Exec(`INSERT INTO t_permission (perm_code, topic_key, match_sort, effect) VALUES (?, ?, ?, ?)`,
		permCode, topicKey, matchSort, effect)
	if err != nil {
		return 0, fmt.Errorf("Permission__Add 失败 | perm_code=%s | err=%w", permCode, err)
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

// RolePermission__Grant 给角色授予权限（roleType: 1业务 2API）；重复授权则复启用并清空过期时间。
func RolePermission__Grant(roleType, roleId, permId uint64, expireAt sql.NullTime) error {
	_, err := DB.Exec(`INSERT INTO t_role_permission (role_type, role_id, perm_id, is_enabled, expire_at)
		VALUES (?, ?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE is_enabled = 1, expire_at = VALUES(expire_at)`,
		roleType, roleId, permId, expireAt)
	if err != nil {
		return fmt.Errorf("RolePermission__Grant 失败 | roleType=%d roleId=%d permId=%d | err=%w", roleType, roleId, permId, err)
	}
	return nil
}

// RolePermission__Revoke 撤销角色权限（置 is_enabled=0，不做物理删除）。
func RolePermission__Revoke(roleType, roleId, permId uint64) error {
	_, err := DB.Exec(`UPDATE t_role_permission SET is_enabled = 0 WHERE role_type = ? AND role_id = ? AND perm_id = ?`,
		roleType, roleId, permId)
	if err != nil {
		return fmt.Errorf("RolePermission__Revoke 失败 | roleType=%d roleId=%d permId=%d | err=%w", roleType, roleId, permId, err)
	}
	return nil
}

// BizUserRole__Bind 绑定业务用户与角色；重复绑定复启用。
func BizUserRole__Bind(userId, roleId uint64, expireAt sql.NullTime) error {
	_, err := DB.Exec(`INSERT INTO t_biz_user_role (user_id, role_id, is_enabled, expire_at)
		VALUES (?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE is_enabled = 1, expire_at = VALUES(expire_at)`,
		userId, roleId, expireAt)
	if err != nil {
		return fmt.Errorf("BizUserRole__Bind 失败 | userId=%d roleId=%d | err=%w", userId, roleId, err)
	}
	return nil
}

// ApiClientRole__Bind 绑定 API 客户端与角色；重复绑定复启用。
func ApiClientRole__Bind(clientId, roleId uint64, expireAt sql.NullTime) error {
	_, err := DB.Exec(`INSERT INTO t_api_client_role (client_id, role_id, is_enabled, expire_at)
		VALUES (?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE is_enabled = 1, expire_at = VALUES(expire_at)`,
		clientId, roleId, expireAt)
	if err != nil {
		return fmt.Errorf("ApiClientRole__Bind 失败 | clientId=%d roleId=%d | err=%w", clientId, roleId, err)
	}
	return nil
}

// DirectPermission__Grant 直接授权给主体（subjectType: 1业务用户 2API客户端）；重复授权复启用。
func DirectPermission__Grant(subjectType, subjectId, permId uint64, expireAt sql.NullTime) error {
	_, err := DB.Exec(`INSERT INTO t_direct_permission (subject_type, subject_id, perm_id, is_enabled, expire_at)
		VALUES (?, ?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE is_enabled = 1, expire_at = VALUES(expire_at)`,
		subjectType, subjectId, permId, expireAt)
	if err != nil {
		return fmt.Errorf("DirectPermission__Grant 失败 | type=%d id=%d permId=%d | err=%w", subjectType, subjectId, permId, err)
	}
	return nil
}

// Permission__SetStatus 启用/停用权限（status: 1启用 0停用）；停用会即时影响所有主体，需清缓存。
func Permission__SetStatus(permId uint64, status int) error {
	_, err := DB.Exec(`UPDATE t_permission SET status = ? WHERE id = ?`, status, permId)
	if err != nil {
		return fmt.Errorf("Permission__SetStatus 失败 | permId=%d | err=%w", permId, err)
	}
	return nil
}
