/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: mysql 用户逻辑
 */

package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"
)

/*
***************采集配置结构体***************
 */

type Collector__Carry_type struct {
	Id   uint   // 采集器标识
	Name string // 采集器名称
	Uuid string // 采集器uuid
}

// 采集配置增加结构体
type Collector_Info_Add_type struct {
	Label   string // 标识
	Uuid    string // Uuid
	Name    string // 设备名称
	User_Id uint   // 用户id
}

type Collector_Info_Update_type struct {
	Id   uint   // 采集 Id
	Name string // 设备名称
}

// 采集配置结构体
type Collector_Info_type struct {
	Id                 uint      // 采集 Id
	Label              string    // 标识
	Creation_Time      time.Time // 创建时间
	Uuid               string    // Uuid (修正为 string)
	Sn                 string    // 设备 sn
	User_Id            uint      // 创建用户 id
	Version            string    // 版本
	Last_Activity_Time time.Time // 最后活动时间
	Equipment_Id       uint      // 设备 id
	Name               string    // 设备名称
}

/*
***************驱动配置结构体***************
 */

type Drive__Carry_type struct {
	Id   uint   // 驱动id唯一标识符
	Type string // 驱动类型
	Name string // 驱动名称
}

type Drive_Config_Add_type struct {
	Id            uint      // 驱动id
	Type          string    // 驱动类型
	Name          string    // 驱动名称
	Config        string    // json配置参数
	Creation_Time time.Time // 创建时间
	Collector_Id  uint      // 采集器标识
}
type Drive_Config_Update_type struct {
	Id     uint   // 驱动id
	Name   string // 驱动名称
	Config string // json配置参数
}
type Drive_Config_type struct {
	Drive_Config_Update_type
	Type          string    // 驱动类型
	Points_Length uint      // 点位数量
	Creation_Time time.Time // 创建时间
}

// 同步
type Drive_Config_Synchronization_type struct {
	Points_Length uint // 点位数量
	Drive_Config_Add_type
}

// 点位-》查询配置
// 传递：drive_type 设备类型
// 返回：configs 配置，err 错误
func Drive_Config__Query_DriveType(drive_type string) (configs []Drive_Config_type, err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Type,
			Drive_Config.Name,
			Drive_Config.Config,
			Drive_Config.Points_Length,
			Drive_Config.Creation_Time
		FROM
			Drive_Config
		WHERE
			Drive_Config.Type = ?
	`
	// 4. 执行查询（统一处理，减少重复代码）
	rows, err := DB.Query(baseQuery, drive_type)

	// 区分无数据和查询错误，日志补充上下文便于排查
	if err == sql.ErrNoRows {
		// log.Printf("查询驱动配置无数据，驱动类型：%s, 分页%d/%d", driveType, page, pageSize)
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败, 错误:%v, SQL:%s, drive_type:%s", err, baseQuery, drive_type)
		log.Print(err)
		return
	}

	defer func(rows *sql.Rows) {
		// 关闭rows时检查错误，避免资源泄漏且捕获隐藏错误
		closeErr := rows.Close()
		if closeErr != nil {
			log.Printf("ERROR 关闭rows失败: %v", closeErr)
		}
	}(rows)

	for rows.Next() {
		var config Drive_Config_type
		err = rows.Scan(
			&config.Id,
			&config.Type,
			&config.Name,
			&config.Config,
			&config.Points_Length,
			&config.Creation_Time,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		configs = append(configs, config)
	}
	return
}

// 驱动-》查询配置
// 传递：driveid 驱动id
// 返回：configs 配置，err 错误
func Drive_Config__Query_DriveId(driveid uint) (config Drive_Config_type, err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Type,
			Drive_Config.Name,
			Drive_Config.Config,
			Drive_Config.Points_Length,
			Drive_Config.Creation_Time
		FROM
			Drive_Config
		WHERE
			Drive_Config.Id = ?
	`

	// 2. 执行查询（统一处理，减少重复代码）
	err = DB.QueryRow(baseQuery, driveid).Scan(
		&config.Id,
		&config.Type,
		&config.Name,
		&config.Config,
		&config.Points_Length,
		&config.Creation_Time,
	)

	return
}

// 驱动-》查询数量
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Drive_Config__Count(collectorId []uint, driveType []string, page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `Drive_Config`"
	var whereConditions []string
	var args []interface{}

	// 处理多 Collector_Id IN 查询
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Id` IN (%s)", placeholders))
		for _, id := range collectorId {
			args = append(args, id)
		}
	}

	// 处理多 Type IN 查询（无 goto，更优雅）
	if len(driveType) > 0 {
		var validTypes []string
		for _, t := range driveType {
			if t != "" {
				validTypes = append(validTypes, t)
			}
		}

		// 只有存在有效类型时才拼接条件
		if len(validTypes) > 0 {
			placeholders := strings.TrimSuffix(strings.Repeat("?,", len(validTypes)), ",")
			whereConditions = append(whereConditions, fmt.Sprintf("`Type` IN (%s)", placeholders))
			for _, t := range validTypes {
				args = append(args, t)
			}
		}
	}

	// 拼接 WHERE
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// COUNT 查询 禁止加 LIMIT，已删除

	// 执行查询
	err = DB.QueryRow(baseQuery, args...).Scan(&count)

	if err == sql.ErrNoRows {
		count = 0
		log.Printf("[Drive_Config__Count] 无符合条件的数据 | collectorId=%v | driveType=%v", collectorId, driveType)
	} else if err != nil {
		err = fmt.Errorf("[Drive_Config__Count] 查询失败 | collectorId=%v | driveType=%v | SQL=%s | args=%v | err=%w",
			collectorId, driveType, baseQuery, args, err)
		log.Print(err)
	}

	return count, err
}

// 驱动 -》查询配置（回调）
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量，callback 回调函数
// 返回：err 错误
func Drive_Config__Query_Callback(id []uint, driveType []string, page uint, pageSize uint, callback func(Drive_Config_type)) (err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Type,
			Drive_Config.Name,
			Drive_Config.Config,
			Drive_Config.Points_Length,
			Drive_Config.Creation_Time
		FROM
			Drive_Config
	`

	var whereConditions []string // 存储WHERE子句的条件片段
	var args []interface{}       // 存储SQL参数，防止注入

	// 2. 拼接WHERE条件（统一收集条件，最后合并）
	// 处理多个 collectorId：IN 查询
	// if len(collectorId) > 0 {
	// 	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
	// 	whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Id` IN (%s)", placeholders))
	// 	for _, id := range collectorId {
	// 		args = append(args, id)
	// 	}
	// }

	// 3. 拼接WHERE条件（统一收集条件，最后合并）
	// 处理多个 collectorId：IN 查询
	if len(id) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(id)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Drive_Config.`Id` IN (%s)", placeholders))
		for _, id := range id {
			args = append(args, id)
		}
	}

	// 处理多个 driveType：IN 查询
	if len(driveType) > 0 {
		var validTypes []string
		for _, t := range driveType {
			if t != "" {
				validTypes = append(validTypes, t)
			}
		}

		if len(validTypes) > 0 {
			placeholders := strings.TrimSuffix(strings.Repeat("?,", len(validTypes)), ",")
			whereConditions = append(whereConditions, fmt.Sprintf("`Type` IN (%s)", placeholders))
			for _, t := range validTypes {
				args = append(args, t)
			}
		}
	}

	// 3. 合并WHERE条件（解决AND开头的语法错误）
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		// 分页计算：page从1开始的话，偏移量是 (page-1)*pageSize；page为0则不分页
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	// 4. 执行查询（统一处理，减少重复代码）
	rows, err := DB.Query(baseQuery, args...)

	// 修复：先检查 err，若出错则 rows 为 nil，不能执行 defer close
	if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}

	// 只有在 rows 不为 nil 时才注册 defer 关闭
	defer rows.Close()

	for rows.Next() {
		var config Drive_Config_type
		err = rows.Scan(
			&config.Id,
			&config.Type,
			&config.Name,
			&config.Config,
			&config.Points_Length,
			&config.Creation_Time,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		callback(config)
	}
	return
}

// 驱动 -》查询配置
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func Drive_Config__Query(id []uint, driveType []string, page uint, pageSize uint) (configs []Drive_Config_type, err error) {
	err = Drive_Config__Query_Callback(id, driveType, page, pageSize, func(config Drive_Config_type) {
		configs = append(configs, config)
	})
	return
}

// 必须在这里拼接字段名（防止SQL注入，只允许白名单）
var DriveConfig_AllowFields = map[string]bool{
	"Id":           true,
	"Type":         true,
	"Name":         true,
	"Config":       true,
	"Collector_Id": true,
}

// 驱动-》搜索
// 传递：field quantity 数量，vague 搜索字段
// 返回：configs 配置，err 错误
func Drive_Config__Search_Field(field string, quantity uint, value string) (configs []Drive_Config_type, err error) {
	if !DriveConfig_AllowFields[field] {
		return nil, fmt.Errorf("field 不合法：%s", field)
	}

	// 1. 初始化 SQL
	baseQuery := fmt.Sprintf(`
		SELECT
			Drive_Config.Id,
			Drive_Config.Type,
			Drive_Config.Name,
			Drive_Config.Config,
			Drive_Config.Points_Length,
			Drive_Config.Creation_Time
		FROM
			Drive_Config
		WHERE
			Drive_Config.%s = ? LIMIT ?
	`, field)

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, value, quantity)
	if err != nil {
		err = fmt.Errorf("ERROR 查询采集配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, []interface{}{value, quantity})
		log.Print(err)
		return nil, err
	}
	// 修复：仅在 err == nil 时 defer close，避免 panic
	defer rows.Close()

	for rows.Next() {
		var config Drive_Config_type
		err = rows.Scan(
			&config.Id,
			&config.Type,
			&config.Name,
			&config.Config,
			&config.Points_Length,
			&config.Creation_Time,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		configs = append(configs, config)
	}
	return
}

// 驱动-》搜索
// 传递：field quantity 数量，vague 模糊搜索字符串
// 返回：configs 配置，err 错误
func Drive_Config__Search_Field_Blurred(quantity uint, vague string) (configs []Drive_Config_type, err error) {
	// 1. 初始化 SQL
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Type,
			Drive_Config.Name,
			Drive_Config.Config,
			Drive_Config.Points_Length,
			Drive_Config.Creation_Time
		FROM
			Drive_Config
		WHERE
			Drive_Config.Name LIKE ?
		LIMIT ?
	`

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, vague, quantity)
	if err != nil {
		err = fmt.Errorf("ERROR 查询采集配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, []interface{}{vague, quantity})
		log.Print(err)
		return nil, err
	}
	// 修复：仅在 err == nil 时 defer close，避免 panic
	defer rows.Close()

	for rows.Next() {
		var config Drive_Config_type
		err = rows.Scan(
			&config.Id,
			&config.Type,
			&config.Name,
			&config.Config,
			&config.Points_Length,
			&config.Creation_Time,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		configs = append(configs, config)
	}
	return
}

// 驱动-》增加配置
// 传递：config 配置数组形式
// 返回：err 错误
func Drive_Config__Add(configs ...Drive_Config_Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		err = fmt.Errorf("批量新增失败：待新增配置列表为空")
		return
	}

	// 2. 遍历校验每个配置的参数合法性
	for i, cfg := range configs {
		if cfg.Type == "" || cfg.Name == "" || cfg.Config == "" || cfg.Collector_Id == 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置Type/Name/Config/Collector_Id不能为空 %v", i, cfg)
			return
		}
	}

	// 3. 拼接批量INSERT的SQL和参数
	baseQuery := "INSERT INTO `Drive_Config`(`Id`, `Type`, `Name`, `Config`, `Collector_Id`, `Creation_Time`) VALUES "
	var args []interface{}
	var valuePlaceholders []string

	// 遍历配置列表
	for _, cfg := range configs {
		// 处理时间：为空就用当前时间，否则用传入的
		var createTime time.Time
		if cfg.Creation_Time.IsZero() {
			createTime = time.Now()
		} else {
			createTime = cfg.Creation_Time
		}

		// 拼接占位符和参数
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?)")
		args = append(args,
			cfg.Id, // ID=0自增，非0使用传入
			cfg.Type,
			cfg.Name,
			cfg.Config,
			cfg.Collector_Id,
			createTime, // 智能时间
		)
	}

	// 拼接完整SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 4. 执行批量插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		err = fmt.Errorf("批量新增驱动配置失败: %v", err)
	}
	return
}

// 驱动-》修改配置
// 传递：config 配置
// 返回：err 错误
func Drive_Config__Update(configs ...Drive_Config_Update_type) (err error) {
	// 1. 空列表校验
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	// 2. 遍历逐个更新
	for idx, config := range configs {
		// 2.1 必传参数校验：ID不能为空
		if config.Id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}

		// 2.2 动态拼接SET子句：非空字段才加入更新
		var setClauses []string
		var args []interface{}

		// Name非空则更新Name字段
		if config.Name != "" {
			setClauses = append(setClauses, "`Name` = ?")
			args = append(args, config.Name)
		}

		// Config非空则更新Config字段
		if config.Config != "" {
			setClauses = append(setClauses, "`Config` = ?")
			args = append(args, config.Config)
		}

		setClauses = append(setClauses, "`Creation_Time` = ?")
		args = append(args, time.Now())

		// 2.3 校验：至少有一个更新字段（Name/Config二选一）
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段 Name/Config至少传一个非空值", idx+1)
			return
		}

		// 2.4 拼接SQL：WHERE条件指定ID
		query := fmt.Sprintf("UPDATE `Drive_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		args = append(args, config.Id) // 最后追加ID参数

		// 2.5 执行更新并捕获错误
		result, errExec := DB.Exec(query, args...)
		if errExec != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s, 参数:%v",
				idx+1, config.Id, errExec, query, args)
			return
		}

		// 可选：校验更新行数（确保有数据被更新）
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("WARNING 第%d条配置更新无生效行, ID:%d（可能ID不存在）", idx+1, config.Id)
		}
	}
	return
}

// 驱动-》修改配置
// 传递：config 配置
// 返回：err 错误
func Drive_Config__Synchronization(configs ...Drive_Config_Synchronization_type) (err error) {
	// 1. 空列表校验
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	// 2. 遍历逐个更新
	for idx, config := range configs {
		// 2.1 必传参数校验：ID不能为空
		if config.Id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}

		// 2.2 动态拼接SET子句：非空字段才加入更新
		var setClauses []string
		var args []interface{}

		// Name非空则更新Name字段
		if config.Name != "" {
			setClauses = append(setClauses, "`Name` = ?")
			args = append(args, config.Name)
		}

		// Config非空则更新Config字段
		if config.Config != "" {
			setClauses = append(setClauses, "`Config` = ?")
			args = append(args, config.Config)
		}

		// Type非空则更新Type字段
		if config.Type != "" {
			setClauses = append(setClauses, "`Type` = ?")
			args = append(args, config.Type)
		}

		// Collector_Id非空则更新Collector_Id字段
		if config.Collector_Id != 0 {
			setClauses = append(setClauses, "`Collector_Id` = ?")
			args = append(args, config.Collector_Id)
		}

		// Points_Length非空则更新Points_Length字段
		if config.Points_Length != 0 {
			setClauses = append(setClauses, "`Points_Length` = ?")
			args = append(args, config.Points_Length)
		}

		if config.Creation_Time.IsZero() {
			setClauses = append(setClauses, "`Creation_Time` = ?")
			args = append(args, time.Now())
		}

		// 2.3 校验：至少有一个更新字段（Name/Config二选一）
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段 Name/Config至少传一个非空值", idx+1)
			return
		}

		// 2.4 拼接SQL：WHERE条件指定ID
		query := fmt.Sprintf("UPDATE `Drive_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		args = append(args, config.Id) // 最后追加ID参数

		// 2.5 执行更新并捕获错误
		_, errExec := DB.Exec(query, args...)
		if errExec != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s, 参数:%v",
				idx+1, config.Id, errExec, query, args)
			return
		}
	}
	return
}

// 驱动-》删除配置
// 传递：ids 删除的id数组
// 返回：err 错误
func Drive_Config__Del(ids ...uint) (err error) {
	// 1. 遍历逐个
	for idx, id := range ids {
		// 1.1 单条配置参数校验
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}

		query := "DELETE FROM `Drive_Config` WHERE `Id` = ? "
		// 修改数据库
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s", idx, id, err, query)
			return
		}
	}
	return
}

// 驱动-》点位数量记录
// 传递：id 驱动ID, quantity 点位数量
// 返回：err 错误
func Drive_Config__Points_Length(ids ...uint) (err error) {
	if len(ids) == 0 {
		err = fmt.Errorf("ERROR 获取驱动点位数据失败，参数为空")
		log.Print(err)
		return
	}

	// 去重
	slices.Sort(ids)
	ids = slices.Compact(ids)

	for _, id := range ids {
		var quantity uint
		quantity, err = Points_Config__Count([]uint{id}, 0, 0)
		if err != nil {
			log.Print(err)
			continue
		}

		query := `UPDATE Drive_Config SET Points_Length = ? WHERE Id = ?`
		_, err = DB.Exec(query, quantity, id)
		if err != nil {
			err = fmt.Errorf("ERROR 修改点位数量错误 %s", err)
			log.Print(err)
		}
	}

	return
}

/*
***************点位配置结构体***************
 */
// 点位配置增加结构体
type Points_Config_Add_type struct {
	Id          uint   // 点位id
	Drive_Id    uint   // 驱动id唯一标识符
	Tag         string // 点位标识
	Description string // 点位描述
	RW_Cancel   string // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
	Value_Type  string // 输出类型
	Config      string
}

// 点位配置更新结构体
type Points_Config_Update_type struct {
	Id          uint   // 点位id
	Tag         string // 点位标识
	Description string // 点位描述
	RW_Cancel   string // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
	Value_Type  string // 输出类型
	Config      string
}

// 点位配置结构体
type Points_Config_type struct {
	Drive Drive__Carry_type

	Points_Config_Update_type
	Creation_Time time.Time // 创建时间

}

// 同步
type Points_Config_Synchronization_type struct {
	Points_Config_Add_type
	Creation_Time time.Time // 创建时间
}

// 点位-》查询数量
// 传递：driveid 设备id，page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Points_Config__Count(driveid []uint, page uint, pageSize uint) (Count uint, err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			COUNT(Points_Config.Id) 
		FROM Points_Config
		INNER JOIN Drive_Config ON Points_Config.Drive_Id = Drive_Config.Id
	`
	var args []interface{} // 存储SQL参数，防止SQL注入
	var whereConditions []string

	// 2. 构建WHERE条件（支持多个 driveid）
	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Points_Config`.`Drive_Id` IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}
	// 拼接 WHERE
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// ⚠️ COUNT 查询 不要 LIMIT，已删除
	// 3. 执行查询
	err = DB.QueryRow(baseQuery, args...).Scan(&Count)

	// 区分无数据和查询错误，日志补充上下文便于排查
	if err == sql.ErrNoRows {
		log.Printf("查询点位配置无数据，driveid=%v", driveid)
		Count = 0
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误：%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	log.Printf("查询成功 %d", Count)
	return
}

// 点位-》查询配置（回调）
// 传递：driveid 设备id，page 页码，pageSize 每页数量，callback 回调函数
// 返回：err 错误
func Points_Config__Query_Callback(driveid []uint, page uint, pageSize uint, callback func(Points_Config_type)) (err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT 
			Points_Config.Id, 
			Points_Config.Tag,
			Points_Config.Description,
			Points_Config.Config,
			Points_Config.RW_Cancel,
			Points_Config.Value_Type,
			Points_Config.Creation_Time,
			Drive_Config.Id AS Drive_Id,
			Drive_Config.Type AS Drive_Type,
			Drive_Config.Name AS Drive_Name
		FROM Points_Config
		INNER JOIN Drive_Config ON Points_Config.Drive_Id = Drive_Config.Id
	`
	var whereConditions []string
	var args []interface{} // 存储SQL参数，防止SQL注入

	// 2. 构建WHERE条件（支持多个 driveid：IN 查询）
	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Points_Config`.`Drive_Id` IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}
	// 拼接 WHERE 条件
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 3. 构建分页条件
	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var (
			config      Points_Config_type
			Description sql.NullString
		)
		err = rows.Scan(
			&config.Id,
			&config.Tag,
			&Description,
			&config.Config,
			&config.RW_Cancel,
			&config.Value_Type,
			&config.Creation_Time,
			&config.Drive.Id,
			&config.Drive.Type,
			&config.Drive.Name,
		)
		if err != nil {
			log.Print(err.Error())
			return err
		}

		config.Description = Description.String
		callback(config)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

// 点位-》查询配置
// 传递：driveid 设备 id, page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func Points_Config__Query(driveid []uint, page uint, pageSize uint) (configs []Points_Config_type, err error) {
	err = Points_Config__Query_Callback(driveid, page, pageSize, func(config Points_Config_type) {
		configs = append(configs, config)
	})
	return
}

// 点位-》查询设备id
// 传递：Id 点位id
// 返回：drive_id 设备id err 错误
func Points_Config__DriveId(id uint) (drive_id uint, err error) {
	query := `
			SELECT
				Drive_Id
			FROM
				Points_Config
			WHERE
				Id = ?
		`
	err = DB.QueryRow(query, id).Scan(&drive_id)
	if err != nil {
		err = fmt.Errorf("ERROR 查询设备id错误 %s", err)
		log.Print(err.Error())
	}
	return
}

// 点位-》增加配置
// 传递：config 配置数组形式
// 返回：err 错误
func Points_Config__Add(configs ...Points_Config_Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}

	// 2. 遍历校验每个配置
	for i, cfg := range configs {
		if cfg.Drive_Id == 0 {
			return fmt.Errorf("批量新增失败：第%d条数据 Drive_Id 等于0", i)
		}
		if cfg.Tag == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Tag 不能为空", i)
		}
		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Config 不能为空", i)
		}
	}

	// 3. SQL 插入（包含 Id 字段）
	baseQuery := `
		INSERT INTO Points_Config (
			Id,
			Drive_Id,
			Tag,
			Description,
			RW_Cancel,
			Value_Type,
			Config,
			Creation_Time
		) VALUES
	`

	var (
		args              []interface{}
		valuePlaceholders []string
		ids               []uint
	)
	now := time.Now()

	// 4. 构建批量参数
	for _, cfg := range configs {
		ids = append(ids, cfg.Drive_Id)
		// 占位符 8 个值
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?, ?)")

		args = append(args,
			cfg.Id, // 👈 关键：Id=0自增，非0用传入
			cfg.Drive_Id,
			cfg.Tag,
			cfg.Description,
			cfg.RW_Cancel,
			cfg.Value_Type,
			cfg.Config,
			now, // 创建时间统一用当前时间
		)
	}

	// 5. 拼接 SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 6. 执行插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 Points_Config 失败: %w", err)
	}

	// 7. 更新点位长度统计
	err = Drive_Config__Points_Length(ids...)
	if err != nil {
		return
	}

	return nil
}

// 点位-》修改配置
// 传递：config 配置
// 返回：conid 获取自增的Id，err 错误
func Points_Config__Update(configs ...Points_Config_Update_type) (err error) {
	// 1. 空列表校验
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	// 2. 遍历逐个更新
	for i, cfg := range configs {
		// 2.1 单条配置参数校验

		// 可选：校验必填字段（Type/Name/Config非空，根据业务需求加）

		if cfg.Id == 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置不能为空", i)
			return
		}

		// 2.2 动态拼接SET子句
		var setClauses []string
		var args []interface{}
		if cfg.Tag != "" {
			setClauses = append(setClauses, "`Tag` = ?")
			args = append(args, cfg.Tag)
		}
		if cfg.Description != "" {
			setClauses = append(setClauses, "`Description` = ?")
			args = append(args, sql.NullString{
				String: cfg.Description,
				Valid:  cfg.Description != "null",
			})
		}
		if cfg.RW_Cancel != "" {
			setClauses = append(setClauses, "`RW_Cancel` = ?")
			args = append(args, cfg.RW_Cancel)
		}
		if cfg.Value_Type != "" {
			setClauses = append(setClauses, "`Value_Type` = ?")
			args = append(args, cfg.Value_Type)
		}
		if cfg.Config != "" {
			setClauses = append(setClauses, "`Config` = ?")
			args = append(args, cfg.Config)
		}

		setClauses = append(setClauses, "`Creation_Time` = ?")
		args = append(args, time.Now())

		// 2.3 校验更新字段i
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段至少传一个", i)
			return
		}

		// 2.4 拼接SQL并执行
		query := fmt.Sprintf("UPDATE `Points_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		args = append(args, cfg.Id)

		_, err = DB.Exec(query, args...)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s, 参数:%v", i, cfg.Id, err, query, args)
			return
		}
	}

	return
}

func Points_Config__Synchronization(configs ...Points_Config_Synchronization_type) (err error) {
	// 1. 空列表校验
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	// 2. 遍历逐个更新
	for i, cfg := range configs {
		// 2.1 单条配置参数校验

		// 可选：校验必填字段（Type/Name/Config非空，根据业务需求加）

		if cfg.Id == 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置不能为空", i)
			return
		}

		// 2.2 动态拼接SET子句
		var setClauses []string
		var args []interface{}

		if cfg.Drive_Id != 0 {
			setClauses = append(setClauses, "`Drive_Id` = ?")
			args = append(args, cfg.Drive_Id)
		}
		if cfg.Tag != "" {
			setClauses = append(setClauses, "`Tag` = ?")
			args = append(args, cfg.Tag)
		}
		if cfg.Description != "" {
			setClauses = append(setClauses, "`Description` = ?")
			args = append(args, sql.NullString{
				String: cfg.Description,
				Valid:  cfg.Description != "null",
			})
		}
		if cfg.Config != "" {
			setClauses = append(setClauses, "`Config` = ?")
			args = append(args, cfg.Config)
		}
		if cfg.RW_Cancel != "" {
			setClauses = append(setClauses, "`RW_Cancel` = ?")
			args = append(args, cfg.RW_Cancel)
		}
		if cfg.Value_Type != "" {
			setClauses = append(setClauses, "`Value_Type` = ?")
			args = append(args, cfg.Value_Type)
		}
		if cfg.Creation_Time.IsZero() {
			setClauses = append(setClauses, "`Creation_Time` = ?")
			args = append(args, time.Now())
		}

		// 2.3 校验更新字段i
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段至少传一个", i)
			return
		}

		// 2.4 拼接SQL并执行
		query := fmt.Sprintf("UPDATE `Points_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		args = append(args, cfg.Id)

		_, err = DB.Exec(query, args...)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s, 参数:%v", i, cfg.Id, err, query, args)
			return
		}
	}

	return
}

// 点位-》删除配置
// 传递：ids 删除的id数组
// 返回：err 错误
func Points_Config__Del(ids ...uint) (err error) {
	// 1. 遍历逐个
	for idx, id := range ids {

		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx)
			return
		}

		_, err = Points_Config__DriveId(id)
		if err != nil {
			return
		}

		query := "DELETE FROM `Points_Config` WHERE `Id` = ? "
		// 修改数据库
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s", idx, id, err, query)
			return
		}

	}

	err = Drive_Config__Points_Length(ids...)
	if err != nil {
		return
	}
	return
}
