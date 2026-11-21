/*
* 日期: 2026.11.19
* 作用: 极简 RBAC 管理端数据访问层（纯 SQL，无业务判定）——供权限管理 CRUD 接口调用。
*       与 rbac.go(校验取数) 区分：本文件只做“配置数据的增删改查”，不做权限判定。
*       安全要点：
*         1) 所有值均参数化；表名/列名不接受用户输入，仅由 subjectType/roleType 白名单映射得到；
*         2) 删除(角色/权限)在事务中级联清理其关联绑定，避免残留脏引用；
*         3) 绑定采用“集合替换”语义：新增用 ON DUPLICATE 复启用、缺失的删除，天然幂等。
 */

package mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// RbacRole_Query 角色行（业务/API 共用同一结构，表由 roleType 决定）。
type RbacRole_Query struct {
	Id       uint64 `json:"id"`
	RoleCode string `json:"role_code"`
	RoleName string `json:"role_name"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

// RbacPerm_Query 权限主表行。
type RbacPerm_Query struct {
	Id         uint64    `json:"id"`
	PermCode   string    `json:"perm_code"`
	TopicKey   string    `json:"topic_key"`
	MatchSort  int       `json:"match_sort"`
	Effect     int       `json:"effect"`
	Status     int       `json:"status"`
	Remark     string    `json:"remark"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

// trans 在单个事务内执行 fn：fn 返回 error 或发生 panic 则回滚，否则提交。
func trans(fn func(tx *sql.Tx) error) (err error) {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	err = fn(tx)
	return err
}

// rbacRoleTable 角色类型→角色表名白名单（1业务 2API）；非法值返回 error，杜绝拼接注入。
func rbacRoleTable(roleType uint64) (string, error) {
	switch roleType {
	case Rbac_Subject_Biz:
		return "t_biz_role", nil
	case Rbac_Subject_Api:
		return "t_api_role", nil
	default:
		return "", fmt.Errorf("rbacRoleTable: 非法角色类型 %d", roleType)
	}
}

// rbacSubjectRoleTable 主体类型→主体角色关联表名白名单（1业务用户 2API客户端）。
func rbacSubjectRoleTable(subjectType uint64) (string, string, error) {
	switch subjectType {
	case Rbac_Subject_Biz:
		return "t_biz_user_role", "user_id", nil
	case Rbac_Subject_Api:
		return "t_api_client_role", "client_id", nil
	default:
		return "", "", fmt.Errorf("rbacSubjectRoleTable: 非法主体类型 %d", subjectType)
	}
}

// ======== 角色 CRUD ========

// rbacLikeArg 把关键字包成 LIKE 参数；空串返回 ok=false（调用方据此跳过过滤）。
func rbacLikeArg(keyword string) (string, bool) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return "", false
	}
	return "%" + kw + "%", true
}

// RbacRole__Query 分页查询某类角色（roleType: 1业务 2API）。keyword 非空时按编码/名称模糊过滤。
// page/pageSize 任一为 0 → 不分页返回全部（供穿梭框等一次性取全量场景）。
func RbacRole__Query(roleType uint64, keyword string, page, pageSize uint) ([]RbacRole_Query, error) {
	table, err := rbacRoleTable(roleType)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("SELECT id, role_code, role_name, sort, status, remark FROM %s", table)
	args := make([]any, 0, 4)
	if like, ok := rbacLikeArg(keyword); ok {
		q += " WHERE role_code LIKE ? OR role_name LIKE ?"
		args = append(args, like, like)
	}
	q += " ORDER BY sort ASC, id ASC"
	if page > 0 && pageSize > 0 {
		q += " LIMIT ? OFFSET ?"
		args = append(args, pageSize, (page-1)*pageSize)
	}
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("RbacRole__Query 失败 | type=%d | err=%w", roleType, err)
	}
	defer rows.Close()
	var out []RbacRole_Query
	for rows.Next() {
		var r RbacRole_Query
		if err := rows.Scan(&r.Id, &r.RoleCode, &r.RoleName, &r.Sort, &r.Status, &r.Remark); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RbacRole__Count 某类角色总数；keyword 非空时与 Query 采用同一过滤条件，保证分页总数一致。
func RbacRole__Count(roleType uint64, keyword string) (int64, error) {
	table, err := rbacRoleTable(roleType)
	if err != nil {
		return 0, err
	}
	q := "SELECT COUNT(*) FROM " + table
	args := make([]any, 0, 2)
	if like, ok := rbacLikeArg(keyword); ok {
		q += " WHERE role_code LIKE ? OR role_name LIKE ?"
		args = append(args, like, like)
	}
	var n int64
	if err := DB.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("RbacRole__Count 失败 | type=%d | err=%w", roleType, err)
	}
	return n, nil
}

// RbacRole__Add 新增角色，返回自增 id。role_code 表内唯一，冲突由调用方转成友好提示。
func RbacRole__Add(roleType uint64, code, name string, sort, status int, remark string) (uint64, error) {
	table, err := rbacRoleTable(roleType)
	if err != nil {
		return 0, err
	}
	res, err := DB.Exec(fmt.Sprintf("INSERT INTO %s (role_code, role_name, sort, status, remark) VALUES (?,?,?,?,?)", table),
		code, name, sort, status, remark)
	if err != nil {
		return 0, fmt.Errorf("RbacRole__Add 失败 | type=%d code=%s | err=%w", roleType, code, err)
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

// RbacRole__Update 按 id 更新角色（不改 id）。
func RbacRole__Update(roleType uint64, id uint64, code, name string, sort, status int, remark string) error {
	table, err := rbacRoleTable(roleType)
	if err != nil {
		return err
	}
	if _, err := DB.Exec(fmt.Sprintf("UPDATE %s SET role_code=?, role_name=?, sort=?, status=?, remark=? WHERE id=?", table),
		code, name, sort, status, remark, id); err != nil {
		return fmt.Errorf("RbacRole__Update 失败 | type=%d id=%d | err=%w", roleType, id, err)
	}
	return nil
}

// RbacRole__Del 删除角色，并在同一事务内清理其权限授予与主体绑定，避免脏引用。
func RbacRole__Del(roleType, id uint64) error {
	table, err := rbacRoleTable(roleType)
	if err != nil {
		return err
	}
	return trans(func(tx *sql.Tx) error {
		if _, err := tx.Exec("DELETE FROM t_role_permission WHERE role_type=? AND role_id=?", roleType, id); err != nil {
			return err
		}
		subTable, _, err := rbacSubjectRoleTable(roleType)
		if err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM "+subTable+" WHERE role_id=?", id); err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM "+table+" WHERE id=?", id)
		return err
	})
}

// ======== 权限主表 CRUD ========

// RbacPerm__Query 分页查询权限字典。keyword 非空时按权限编码/资源主题模糊过滤；page/pageSize 任一为 0 → 返回全部。
func RbacPerm__Query(keyword string, page, pageSize uint) ([]RbacPerm_Query, error) {
	q := "SELECT id, perm_code, topic_key, match_sort, effect, status, remark, create_time, update_time FROM t_permission"
	args := make([]any, 0, 4)
	if like, ok := rbacLikeArg(keyword); ok {
		q += " WHERE perm_code LIKE ? OR topic_key LIKE ?"
		args = append(args, like, like)
	}
	q += " ORDER BY match_sort ASC, id ASC"
	if page > 0 && pageSize > 0 {
		q += " LIMIT ? OFFSET ?"
		args = append(args, pageSize, (page-1)*pageSize)
	}
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("RbacPerm__Query 失败 | err=%w", err)
	}
	defer rows.Close()
	var out []RbacPerm_Query
	for rows.Next() {
		var p RbacPerm_Query
		if err := rows.Scan(&p.Id, &p.PermCode, &p.TopicKey, &p.MatchSort, &p.Effect, &p.Status, &p.Remark, &p.CreateTime, &p.UpdateTime); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RbacPerm__Count 权限总数；keyword 非空时与 Query 采用同一过滤条件。
func RbacPerm__Count(keyword string) (int64, error) {
	q := "SELECT COUNT(*) FROM t_permission"
	args := make([]any, 0, 2)
	if like, ok := rbacLikeArg(keyword); ok {
		q += " WHERE perm_code LIKE ? OR topic_key LIKE ?"
		args = append(args, like, like)
	}
	var n int64
	if err := DB.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("RbacPerm__Count 失败 | err=%w", err)
	}
	return n, nil
}

// RbacPerm__Add 新增权限，返回自增 id。主题合法性由 handler 层用 rbac.ValidatePattern 先校验。
func RbacPerm__Add(code, topicKey string, matchSort, effect, status int, remark string) (uint64, error) {
	res, err := DB.Exec(`INSERT INTO t_permission (perm_code, topic_key, match_sort, effect, status, remark) VALUES (?,?,?,?,?,?)`,
		code, topicKey, matchSort, effect, status, remark)
	if err != nil {
		return 0, fmt.Errorf("RbacPerm__Add 失败 | code=%s | err=%w", code, err)
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

// RbacPerm__Update 按 id 更新权限。
func RbacPerm__Update(id uint64, code, topicKey string, matchSort, effect, status int, remark string) error {
	if _, err := DB.Exec(`UPDATE t_permission SET perm_code=?, topic_key=?, match_sort=?, effect=?, status=?, remark=? WHERE id=?`,
		code, topicKey, matchSort, effect, status, remark, id); err != nil {
		return fmt.Errorf("RbacPerm__Update 失败 | id=%d | err=%w", id, err)
	}
	return nil
}

// RbacPerm__Del 删除权限，并级联清理角色权限关联与直接授权。
func RbacPerm__Del(id uint64) error {
	return trans(func(tx *sql.Tx) error {
		if _, err := tx.Exec("DELETE FROM t_role_permission WHERE perm_id=?", id); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM t_direct_permission WHERE perm_id=?", id); err != nil {
			return err
		}
		_, err := tx.Exec("DELETE FROM t_permission WHERE id=?", id)
		return err
	})
}

// ======== 绑定：集合替换（角色↔权限、主体↔角色、主体↔权限直授）========

// uint64List 扫描单列 id 集合。
func uint64List(query string, args ...any) ([]uint64, error) {
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// replaceSet 在事务内把 (表, 定位条件) 下关联的 id 集合替换为 wantIds：
// 先删除不在 wantIds 中的，再对 wantIds 逐个 ON DUPLICATE 复启用。wantIds 为空即清空。
// linkCol 为关联列名（perm_id/role_id，均为内部白名单常量，非用户输入）。
func replaceSet(table, linkCol, where string, whereArgs []any, wantIds []uint64, insertFn func(*sql.Tx, uint64) error) error {
	return trans(func(tx *sql.Tx) error {
		del := "DELETE FROM " + table + " WHERE " + where
		if len(wantIds) > 0 {
			ph := strings.TrimSuffix(strings.Repeat("?,", len(wantIds)), ",")
			del += " AND " + linkCol + " NOT IN (" + ph + ")"
			if _, err := tx.Exec(del, append(append([]any{}, whereArgs...), toAnySlice(wantIds)...)...); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(del, whereArgs...); err != nil {
				return err
			}
		}
		for _, id := range wantIds {
			if err := insertFn(tx, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// toAnySlice 把 []uint64 转为 []any 供 Exec 展开。
func toAnySlice(ids []uint64) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// RolePerm__List 列出角色已授予的权限 id。
func RolePerm__List(roleType, roleId uint64) ([]uint64, error) {
	return uint64List("SELECT perm_id FROM t_role_permission WHERE role_type=? AND role_id=?", roleType, roleId)
}

// RolePerm__Replace 用集合替换语义设定角色的权限集。
func RolePerm__Replace(roleType, roleId uint64, permIds []uint64) error {
	where := "role_type=? AND role_id=?"
	whereArgs := []any{roleType, roleId}
	return replaceSet("t_role_permission", "perm_id", where, whereArgs, permIds,
		func(tx *sql.Tx, permId uint64) error {
			_, err := tx.Exec(`INSERT INTO t_role_permission (role_type, role_id, perm_id, is_enabled) VALUES (?,?,?,1)
				ON DUPLICATE KEY UPDATE is_enabled=1`, roleType, roleId, permId)
			return err
		})
}

// SubjectRole__List 列出主体（用户/客户端）已绑定的角色 id。
func SubjectRole__List(subjectType, subjectId uint64) ([]uint64, error) {
	table, idCol, err := rbacSubjectRoleTable(subjectType)
	if err != nil {
		return nil, err
	}
	return uint64List("SELECT role_id FROM "+table+" WHERE "+idCol+"=?", subjectId)
}

// SubjectRole__Replace 设定主体的角色绑定集合。
func SubjectRole__Replace(subjectType, subjectId uint64, roleIds []uint64) error {
	table, idCol, err := rbacSubjectRoleTable(subjectType)
	if err != nil {
		return err
	}
	where := idCol + "=?"
	whereArgs := []any{subjectId}
	return replaceSet(table, "role_id", where, whereArgs, roleIds,
		func(tx *sql.Tx, roleId uint64) error {
			_, err := tx.Exec(fmt.Sprintf("INSERT INTO %s (%s, role_id, is_enabled) VALUES (?,?,1)\n				ON DUPLICATE KEY UPDATE is_enabled=1", table, idCol),
				subjectId, roleId)
			return err
		})
}

// SubjectPerm__List 列出主体（用户/客户端）的直接授权权限 id。
func SubjectPerm__List(subjectType, subjectId uint64) ([]uint64, error) {
	return uint64List("SELECT perm_id FROM t_direct_permission WHERE subject_type=? AND subject_id=?", subjectType, subjectId)
}

// SubjectPerm__Replace 设定主体的直接授权集合。
func SubjectPerm__Replace(subjectType, subjectId uint64, permIds []uint64) error {
	where := "subject_type=? AND subject_id=?"
	whereArgs := []any{subjectType, subjectId}
	return replaceSet("t_direct_permission", "perm_id", where, whereArgs, permIds,
		func(tx *sql.Tx, permId uint64) error {
			_, err := tx.Exec(`INSERT INTO t_direct_permission (subject_type, subject_id, perm_id, is_enabled) VALUES (?,?,?,1)
				ON DUPLICATE KEY UPDATE is_enabled=1`, subjectType, subjectId, permId)
			return err
		})
}
