/*
***************操作日志（单表 log_operate）***************
	所有审计事件共用一张表：配置增删改查、登陆/登出、写值、重启、导入/导出。
	两个维度定位事件：Type（操作对象）+ Operate（操作方式），两列都直接存可读文本（drive / add / …），
	库里看到的值和代码里的常量一字不差，不必再查数字对照表；
	Operator（操作人）区分通路主体，格式固定为 user:<id> / aksk:<id> / internalmqtt；
	其余细节（通路、请求路径、目标 id/tag、变更快照、结果）全部序列化进 Content，字段不再拆表。
*/

package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// Log_Source 日志来源通路：接口账号（api_key）、登录用户（user_account）、内部 MQTT 下行指令
type Log_Source string

const (
	Log_Source_AKSK Log_Source = "aksk"         // AK/SK 接口账号
	Log_Source_User Log_Source = "user"         // 网页登录用户
	Log_Source_Mqtt Log_Source = "internalmqtt" // 内部 MQTT 通道（无账号凭证）
)

// Operator 拼出 Operator 列的操作人标识：user:1 / aksk:2 / internalmqtt
// MQTT 通道不区分账号，一律记通道名，具体指令与参数在 Content 里
func (s Log_Source) Operator(accountId uint) string {
	if s == Log_Source_Mqtt {
		return string(Log_Source_Mqtt)
	}
	return fmt.Sprintf("%s:%d", string(s), accountId)
}

// Log_Type 操作对象（log_operate.Type）。列值即可读文本，与常量名一一对应；
// 单独设成命名类型而非裸 string，是为了让 Type / Operate 传反时编译器直接报错
// （两个埋点函数都是 typ、operate 相邻同型参数）。
type Log_Type string

// all 表示一次动作跨多类对象或整体动作（如整包配置导入失败）
const (
	Log_Type_All     Log_Type = "all"     // 全部配置 / 未细分
	Log_Type_Drive   Log_Type = "drive"   // 驱动
	Log_Type_Point   Log_Type = "point"   // 点位
	Log_Type_Alarm   Log_Type = "alarm"   // 报警
	Log_Type_Log     Log_Type = "log"     // 日志（日志自身的查询与清理）
	Log_Type_History Log_Type = "history" // 历史配置
	Log_Type_Account Log_Type = "account" // 账号与权限（用户 / AKSK / RBAC）
	Log_Type_System  Log_Type = "system"  // 系统配置（config.yaml）
	Log_Type_Service Log_Type = "service" // 服务进程（重启 / 退出）
)

// Log_Operate 操作方式（log_operate.Operate）
type Log_Operate string

const (
	Log_Operate_Add     Log_Operate = "add"     // 增
	Log_Operate_Del     Log_Operate = "del"     // 删
	Log_Operate_Update  Log_Operate = "update"  // 改
	Log_Operate_Query   Log_Operate = "query"   // 查
	Log_Operate_Write   Log_Operate = "write"   // 写值
	Log_Operate_Login   Log_Operate = "login"   // 登陆（含远程签发令牌）
	Log_Operate_Logout  Log_Operate = "logout"  // 登出
	Log_Operate_Restart Log_Operate = "restart" // 重启
	Log_Operate_Import  Log_Operate = "import"  // 导入
	Log_Operate_Export  Log_Operate = "export"  // 导出
)

// Log_Row 操作日志一行（对应 log_operate 表）
type Log_Row struct {
	Id          uint        // 日志 id（新增传 0 走自增）
	Type        Log_Type    // 操作对象：Log_Type_*
	Operate     Log_Operate // 操作方式：Log_Operate_*
	Operator    string      // 操作人：user:1 / aksk:1 / internalmqtt
	Content     string      // 详情 JSON（允许为空）
	Ip          string      // 来源 IP；MQTT 通路无来源 IP 传空串
	Create_Time time.Time   // 时间戳（零值则取当前时间）
}

// Log_Filter 日志查询条件；各切片/字符串为空表示该维度不限
type Log_Filter struct {
	Types     []Log_Type    // 操作对象
	Operates  []Log_Operate // 操作方式
	Operators []string      // 操作人（精确匹配，如 user:1）
	Keyword   string        // 在 Operator / Content 上做 LIKE 模糊匹配
	Start     time.Time     // 起始时间（含）
	End       time.Time     // 结束时间（含）
	Page      uint          // 页码，0 = 不分页
	PageSize  uint          // 每页条数
}

// resolveLogTime 零值时间回退为当前时间
func resolveLogTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

// inCond 把某一维度的 IN 条件拼成片段；列表为空返回空串，调用方据此跳过（值一律走占位符）
func inCond[T ~string](col string, vals []T) (string, []interface{}) {
	if len(vals) == 0 {
		return "", nil
	}
	ph := strings.TrimSuffix(strings.Repeat("?,", len(vals)), ",")
	args := make([]interface{}, 0, len(vals))
	for _, v := range vals {
		args = append(args, string(v))
	}
	return fmt.Sprintf("`%s` IN (%s)", col, ph), args
}

// buildWhere 把查询条件拼成 WHERE 子句（列名固定，值一律走占位符）
func (f Log_Filter) buildWhere() (where string, args []interface{}) {
	var conds []string

	if clause, vals := inCond("Type", f.Types); clause != "" {
		conds = append(conds, clause)
		args = append(args, vals...)
	}
	if clause, vals := inCond("Operate", f.Operates); clause != "" {
		conds = append(conds, clause)
		args = append(args, vals...)
	}
	if clause, vals := inCond("Operator", f.Operators); clause != "" {
		conds = append(conds, clause)
		args = append(args, vals...)
	}
	if f.Keyword != "" {
		conds = append(conds, "(`Operator` LIKE ? OR `Content` LIKE ?)")
		like := "%" + f.Keyword + "%"
		args = append(args, like, like)
	}
	if !f.Start.IsZero() {
		conds = append(conds, "`Create_Time` >= ?")
		args = append(args, f.Start)
	}
	if !f.End.IsZero() {
		conds = append(conds, "`Create_Time` <= ?")
		args = append(args, f.End)
	}

	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	return where, args
}

// Log__Add 批量新增操作日志
func Log__Add(data ...Log_Row) (err error) {
	if len(data) == 0 {
		return fmt.Errorf("批量新增操作日志失败：数据列表为空")
	}
	const baseQuery = "INSERT INTO `log_operate` (Id, `Type`, `Operate`, `Operator`, `Content`, `Ip`, Create_Time) VALUES "
	var args []interface{}
	var valuePlaceholders []string
	for i, d := range data {
		if d.Operator == "" {
			return fmt.Errorf("批量新增失败：第[%d]条操作日志 Operator 为空", i)
		}
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			d.Id, d.Type, d.Operate, d.Operator, d.Content, d.Ip,
			resolveLogTime(d.Create_Time),
		)
	}
	query := baseQuery + strings.Join(valuePlaceholders, ", ")
	if _, err = DB.Exec(query, args...); err != nil {
		return fmt.Errorf("批量插入 log_operate 失败: %w", err)
	}
	return nil
}

// Log__Query_Callback 按条件分页回调查询操作日志（按 Id 倒序，即时间倒序）
func Log__Query_Callback(f Log_Filter, callback func(Log_Row)) (err error) {
	baseQuery := "SELECT Id, `Type`, `Operate`, `Operator`, `Content`, `Ip`, Create_Time FROM `log_operate`"
	where, args := f.buildWhere()
	baseQuery += where
	baseQuery += " ORDER BY Id DESC"
	if f.Page != 0 {
		offset := (f.Page - 1) * f.PageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, f.PageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询操作日志失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			d       Log_Row
			content sql.NullString
		)
		err = rows.Scan(&d.Id, &d.Type, &d.Operate, &d.Operator, &content, &d.Ip, &d.Create_Time)
		if err != nil {
			err = fmt.Errorf("ERROR 遍历操作日志失败，错误:%w", err)
			log.Print(err)
			return
		}
		d.Content = content.String
		callback(d)
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("ERROR 操作日志结果集异常，错误:%w", err)
		log.Print(err)
	}
	return
}

// Log__Query 按条件分页查询操作日志
func Log__Query(f Log_Filter) (data []Log_Row, err error) {
	err = Log__Query_Callback(f, func(d Log_Row) {
		data = append(data, d)
	})
	return
}

// Log__Count 按条件统计操作日志条数（分页参数与项目内其他 Count 保持一致口径）
func Log__Count(f Log_Filter) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `log_operate`"
	where, args := f.buildWhere()
	baseQuery += where
	if f.Page != 0 {
		offset := (f.Page - 1) * f.PageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, f.PageSize)
	}
	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[Log__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return count, err
}

// Log_Del 按 Id 批量删除操作日志
func Log_Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return nil
	}
	for idx, id := range ids {
		if id == 0 {
			return fmt.Errorf("ERROR 第%d条操作日志ID(Id)不能为空", idx+1)
		}
		query := "DELETE FROM `log_operate` WHERE `Id` = ?"
		if _, execErr := DB.Exec(query, id); execErr != nil {
			return fmt.Errorf("ERROR 第%d条操作日志删除失败, ID:%d, 错误:%w", idx+1, id, execErr)
		}
	}
	return nil
}
