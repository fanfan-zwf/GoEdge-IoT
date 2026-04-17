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

// 采集-》查询数量
// 传递：page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Collector_Info__Count(page uint, pageSize uint) (count uint, err error) {
	// 1. 初始化 SQL（COUNT 查询不需要 LIMIT，否则统计的是当前页数量而非总数）
	baseQuery := "SELECT COUNT(`Id`) FROM `Collector_Info`"
	var whereConditions []string
	var args []interface{}

	// 注意：COUNT 统计全量数据，不应受分页参数影响，故移除原有的 page != 0 添加 LIMIT 的逻辑

	// 2. 合并 WHERE 条件
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		// 分页计算：page从1开始的话，偏移量是 (page-1)*pageSize；page为0则不分页
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	// 3. 执行 COUNT 查询
	err = DB.QueryRow(baseQuery, args...).Scan(&count)

	// 4. 错误处理
	if err != nil {
		if err == sql.ErrNoRows {
			count = 0
			return count, nil
		}
		err = fmt.Errorf("[Collector_Info__Count] 查询失败 | SQL=%s | args=%v | err=%w",
			baseQuery, args, err)
		log.Print(err)
		return 0, err
	}

	return count, nil
}

// 采集-》查询配置
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func Collector_Info__Query(page uint, pageSize uint) (configs []Collector_Info_type, err error) {

	// 1. 初始化 SQL
	baseQuery := "SELECT `Id`, `Equipment_Id`, `Label`, `Creation_Time`, `Uuid`, `Sn`, `User_Id`, `Version`, `Last_Activity_Time`, `Name` FROM `Collector_Info`"

	var whereConditions []string
	var args []interface{}

	// 2. 合并 WHERE 条件
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 3. 添加分页
	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询采集配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return nil, err
	}
	// 修复：仅在 err == nil 时 defer close，避免 panic
	defer rows.Close()

	var (
		Sn                 sql.NullString
		Last_Activity_Time sql.NullTime
		Name               sql.NullString
	)
	for rows.Next() {
		var Config Collector_Info_type
		err = rows.Scan(
			&Config.Id,
			&Config.Equipment_Id,
			&Config.Label,
			&Config.Creation_Time,
			&Config.Uuid,
			&Sn,
			&Config.User_Id,
			&Config.Version,
			&Last_Activity_Time,
			&Name,
		)
		if err != nil {
			log.Print(err.Error())
			return nil, err
		}

		Config.Sn = Sn.String
		Config.Last_Activity_Time = Last_Activity_Time.Time
		Config.Name = Name.String

		configs = append(configs, Config)
	}

	// 检查遍历过程中的错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

// 采集-》查询采集主题
// 传递：采集服务的uuid
// 返回：listen_topic 采集服务监听主题，err 错误
func Collector_Info__Query_Uuid__ListenTopic(uuid string) (listen_topic string, err error) {
	if uuid == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	baseQuery := "SELECT `Listen_Topic` FROM `Collector_Info` WHERE `Uuid` = ?"
	err = DB.QueryRow(baseQuery, uuid).Scan(&listen_topic)
	if err != nil {
		err = fmt.Errorf("ERROR [Collector_Info__Query_Uuid__ListenTopic] 查询失败 | SQL=%s | args=%v | err=%w",
			baseQuery, uuid, err)
		log.Print(err)
	}

	return
}

// 采集-》查询采集主题
// 传递：采集服务的uuid
// 返回：listen_topic 采集服务监听主题，err 错误
func Collector_Info__Query_Uuid__DbServiceConfig(uuid string) (db_service_config string, err error) {
	if uuid == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	baseQuery := "SELECT `Db_Service_Config` FROM `Collector_Info` WHERE `Uuid` = ?"
	err = DB.QueryRow(baseQuery, uuid).Scan(&db_service_config)
	if err != nil {
		err = fmt.Errorf("ERROR [Collector_Info__Query_Uuid__DbServiceConfig] 查询失败 | SQL=%s | args=%v | err=%w",
			baseQuery, uuid, err)
		log.Print(err)
	}

	return
}

// 采集-》搜索
// 传递：field quantity 数量，vague 模糊搜索字符串
// 返回：configs 配置，err 错误
func Collector_Info__Search_Name(field string, quantity uint, vague string) (configs []Collector_Info_type, err error) {
	if field != "Name" {
		err = fmt.Errorf("ERROR field参数错误 field:%s", field)
		return
	} else if field == "" {
		field = "Name"
	}

	if vague == "" {
		return nil, fmt.Errorf("参数错误")
	}

	// 1. 初始化 SQL
	baseQuery := "SELECT `Id`, `Equipment_Id`, `Label`, `Creation_Time`, `Uuid`, `Sn`, `User_Id`, `Version`, `Last_Activity_Time`, `Name` FROM `Collector_Info` WHERE ? LIKE ? LIMIT ?"

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, field, vague, quantity)
	if err != nil {
		err = fmt.Errorf("ERROR 查询采集配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, []interface{}{vague, quantity})
		log.Print(err)
		return nil, err
	}
	// 修复：仅在 err == nil 时 defer close，避免 panic
	defer rows.Close()

	var (
		Sn                 sql.NullString
		Last_Activity_Time sql.NullTime
		Name               sql.NullString
	)
	for rows.Next() {
		var Config Collector_Info_type
		err = rows.Scan(
			&Config.Id,
			&Config.Equipment_Id,
			&Config.Label,
			&Config.Creation_Time,
			&Config.Uuid,
			&Sn,
			&Config.User_Id,
			&Config.Version,
			&Last_Activity_Time,
			&Name,
		)
		if err != nil {
			log.Print(err.Error())
			return nil, err
		}

		Config.Sn = Sn.String
		Config.Last_Activity_Time = Last_Activity_Time.Time
		Config.Name = Name.String

		configs = append(configs, Config)
	}

	// 检查遍历过程中的错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

// 采集-》增加配置
// 传递：config 配置数组形式
// 返回：err 错误
func Collector_Info__Add(configs ...Collector_Info_Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		err = fmt.Errorf("批量新增失败：待新增配置列表为空")
		return
	}

	// 2. 遍历校验每个配置的参数合法性
	for i, cfg := range configs {
		// 可选：校验必填字段（Type/Name/Config非空，根据业务需求加）
		if cfg.Label == "" || cfg.Uuid == "" || cfg.User_Id == 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置Label/Uuid/User_Id不能为空", i)
			return
		}
	}

	// 3. 拼接批量INSERT的SQL和参数
	baseQuery := "INSERT INTO `Collector_Info`(`Label`, `Uuid`, `User_Id`, `Creation_Time`, `Name` ,`Version`) VALUES "
	var args []interface{}         // 存储所有参数
	var valuePlaceholders []string // 存储每个值组的占位符 (?, ?, ?, ?)

	// 遍历配置列表，拼接占位符和参数
	for _, cfg := range configs {
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?)")
		args = append(args, cfg.Label, cfg.Uuid, cfg.User_Id, time.Now(), sql.NullString{
			String: cfg.Name,
			Valid:  cfg.Name != "",
		}, "V1.0")
	}

	// 拼接完整SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 4. 执行批量插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		err = fmt.Errorf("批量新增驱动配置失败, SQL:%s, 参数数:%d, 错误:%v", query, len(args), err)
	}
	return
}

// 采集-》更新配置
// 传递：config 配置数组形式
// 返回：err 错误
func Collector_Info__Update(configs ...Collector_Info_Update_type) (err error) {
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

		// 2.3 校验：至少有一个更新字段（Name/Config二选一）
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段 Name至少传一个非空值", idx+1)
			return
		}

		// 2.4 拼接SQL：WHERE条件指定ID
		query := fmt.Sprintf("UPDATE `Collector_Info` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
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

// 采集-》删除配置
// 传递：ids 删除的id数组
// 返回：err 错误
func Collector_Info__Del(ids ...uint) (err error) {
	// 1. 遍历逐个
	for idx, id := range ids {
		// 1.1 单条配置参数校验
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}

		query := "DELETE FROM `Collector_Info` WHERE `Id` = ? "
		// 修改数据库
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s", idx, id, err, query)
			return
		}
	}
	return
}

// 采集-》心跳
// 传递：Uuid 采集器uuid heartbeat 心跳时间
// 返回：err 错误
func Collector_Info__Last_Activity_Time(Uuid string, heartbeat time.Time) (err error) {
	query := "UPDATE `Collector_Info` SET `Last_Activity_Time` = ? WHERE `Uuid` = ? "

	_, err = DB.Exec(query, heartbeat, Uuid)
	if err != nil {
		log.Printf("ERROR 心跳写入错误 Uuid:%s %s", Uuid, err)
	}
	return
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
	Name         string // 驱动名称
	Config       string // json配置参数
	Type         string // 驱动类型
	Collector_Id uint   // 采集器标识
}
type Drive_Config_Update_type struct {
	Id     uint   // 驱动id
	Name   string // 驱动名称
	Config string // json配置参数
}
type Drive_Config_type struct {
	Collector Collector__Carry_type

	Drive_Config_Update_type
	Type          string    // 驱动类型
	Points_Length uint      // 点位数量
	Creation_Time time.Time // 创建时间
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
			IFNULL(Drive_Config.Collector_Id, 0) AS Collector_Id,
			Drive_Config.Creation_Time,
			IFNULL(Collector_Info.Name, '') AS Creation_Name,
			IFNULL(Collector_Info.Uuid, '') AS Creation_Uuid
		FROM
			Drive_Config
		INNER JOIN Collector_Info ON
			Drive_Config.Collector_Id = Collector_Info.Id
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
			&config.Collector.Id,
			&config.Creation_Time,
			&config.Collector.Name,
			&config.Collector.Uuid,
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
			Drive_Config.Creation_Time,
			IFNULL(Drive_Config.Collector_Id, 0) AS Collector_Id,
			IFNULL(Collector_Info.Name, '') AS Creation_Name,
			IFNULL(Collector_Info.Uuid, '') AS Creation_Uuid
		FROM
			Drive_Config
		INNER JOIN Collector_Info ON
			Drive_Config.Collector_Id = Collector_Info.Id
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
		&config.Collector.Id,
		&config.Creation_Time,
		&config.Collector.Name,
		&config.Collector.Uuid,
	)

	return
}

// 驱动-》查询数量
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Drive_Config__Count(collectorId uint, driveType string, page uint, pageSize uint) (count uint, err error) {
	// 1. 初始化SQL和条件切片（规范WHERE条件拼接）
	baseQuery := "SELECT COUNT(`Id`) FROM `Drive_Config`"
	var whereConditions []string // 存储WHERE子句的条件片段
	var args []interface{}       // 存储SQL参数，防止注入

	// 2. 拼接WHERE条件（统一收集条件，最后合并）
	if collectorId != 0 {
		whereConditions = append(whereConditions, "`Collector_Id` = ?")
		args = append(args, collectorId)
	}
	if driveType != "" {
		whereConditions = append(whereConditions, "`Type` = ?")
		args = append(args, driveType)
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

	// 4. 执行COUNT查询（COUNT统计绝对不能加LIMIT！）
	// 注意：COUNT结果用uint64更安全，避免数值溢出
	err = DB.QueryRow(baseQuery, args...).Scan(&count)

	// 5. 精细化错误处理（补充上下文，便于排查）
	if err == sql.ErrNoRows {
		// 无数据时返回0，符合COUNT的语义（COUNT本身不会返回NoRows，此处兜底）
		count = 0
		log.Printf("[Drive_Config__Count] 无符合条件的数据 | collectorId=%d | driveType=%s", collectorId, driveType)
	} else if err != nil {
		err = fmt.Errorf("[Drive_Config__Count] 查询失败 | collectorId=%d | driveType=%s | SQL=%s | args=%v | err=%w",
			collectorId, driveType, baseQuery, args, err)
		log.Print(err) // 建议用结构化日志，此处简化为log.Error
	}

	return count, err
}

// 驱动 -》查询配置
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func Drive_Config__Query(collectorId uint, driveType string, page uint, pageSize uint) (configs []Drive_Config_type, err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Type,
			Drive_Config.Name,
			Drive_Config.Config,
			Drive_Config.Points_Length,
			IFNULL(Drive_Config.Collector_Id, 0) AS Collector_Id,
			Drive_Config.Creation_Time,
			IFNULL(Collector_Info.Name, '') AS Creation_Name,
			IFNULL(Collector_Info.Uuid, '') AS Creation_Uuid
		FROM
			Drive_Config
		LEFT JOIN Collector_Info ON
			Drive_Config.Collector_Id = Collector_Info.Id
	`

	var whereConditions []string // 存储WHERE子句的条件片段
	var args []interface{}       // 存储SQL参数，防止注入

	// 2. 拼接WHERE条件（统一收集条件，最后合并）
	if collectorId != 0 {
		whereConditions = append(whereConditions, "`Collector_Id` = ?")
		args = append(args, collectorId)
	}
	if driveType != "" {
		whereConditions = append(whereConditions, "`Type` = ?")
		args = append(args, driveType)
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
			&config.Collector.Id,
			&config.Creation_Time,
			&config.Collector.Name,
			&config.Collector.Uuid,
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
func Drive_Config__Search_Name(field string, quantity uint, vague string) (configs []Drive_Config_type, err error) {
	if field != "Name" {
		err = fmt.Errorf("ERROR field参数错误 field:%s", field)
		return
	} else if field == "" {
		field = "Name"
	}

	if vague == "" {
		return nil, fmt.Errorf("参数错误")
	}

	// 1. 初始化 SQL
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Type,
			Drive_Config.Name,
			Drive_Config.Config,
			Drive_Config.Points_Length,
			IFNULL(Drive_Config.Collector_Id, 0) AS Collector_Id,
			Drive_Config.Creation_Time,
			IFNULL(Collector_Info.Name, '') AS Creation_Name,
			IFNULL(Collector_Info.Uuid, '') AS Creation_Uuid
		FROM
			Drive_Config
		LEFT JOIN Collector_Info ON
			Drive_Config.Collector_Id = Collector_Info.Id
		WHERE
			? LIKE ?
		LIMIT ?
	`

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, field, vague, quantity)
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
			&config.Collector.Id,
			&config.Creation_Time,
			&config.Collector.Name,
			&config.Collector.Uuid,
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
		// 可选：校验必填字段（Type/Name/Config非空，根据业务需求加）
		if cfg.Type == "" || cfg.Name == "" || cfg.Config == "" || cfg.Collector_Id == 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置Type/Name/Config/Collector_Id不能为空 %v", i, cfg)
			return
		}
	}

	// 3. 拼接批量INSERT的SQL和参数
	baseQuery := "INSERT INTO `Drive_Config`(`Type`, `Name`, `Config`, `Collector_Id`, `Creation_Time`) VALUES "
	var args []interface{}         // 存储所有参数
	var valuePlaceholders []string // 存储每个值组的占位符 (?, ?, ?)

	// 遍历配置列表，拼接占位符和参数
	for _, cfg := range configs {
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?)")
		args = append(args, cfg.Type, cfg.Name, cfg.Config, cfg.Collector_Id, time.Now())
	}

	// 拼接完整SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 4. 执行批量插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		err = fmt.Errorf("批量新增驱动配置失败, SQL:%s, 参数数:%d, 错误:%v", query, len(args), err)
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
func Drive_Config__Points_Length(id uint, quantity int) (err error) {
	query := `
		UPDATE
			Drive_Config
		SET
			Points_Length = Points_Length + CAST( ? AS SIGNED )
		WHERE
			Id = ?
	`
	_, err = DB.Exec(query, quantity, id)
	if err != nil {
		err = fmt.Errorf("ERROR 修改点位数量错误 %s", err)
		log.Print(err)
	}
	return
}

/*
***************点位配置结构体***************
 */
// 点位配置增加结构体
type Points_Config_Add_type struct {
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
	Collector Collector__Carry_type
	Drive     Drive__Carry_type

	Points_Config_Update_type
	Creation_Time time.Time // 创建时间

}

// 点位-》查询数量
// 传递：driveid 设备id，page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Points_Config__Count(driveid uint, page uint, pageSize uint) (Count uint, err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			COUNT(Points_Config.Id) 
		FROM Points_Config
		INNER JOIN Drive_Config ON Points_Config.Drive_Id = Drive_Config.Id
	`
	var args []interface{} // 存储SQL参数，防止SQL注入

	// 2. 构建WHERE条件（统一处理，避免多个else if分支）
	if driveid != 0 {
		baseQuery += " WHERE `Points_Config`.`Drive_Id` = ?"
		args = append(args, driveid)
	}

	// 3. 构建分页条件（统一处理，避免重复逻辑）
	if page != 0 {
		// 分页计算：page从1开始的话，偏移量是 (page-1)*pageSize；page为0则不分页
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}
	// 4. 执行查询（统一处理，减少重复代码）
	err = DB.QueryRow(baseQuery, args...).Scan(&Count)

	// 区分无数据和查询错误，日志补充上下文便于排查
	if err == sql.ErrNoRows {
		log.Printf("查询驱动配置无数据，分页%d/%d", page, pageSize)
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败，错误：%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	log.Printf("查询成功 %d", Count)
	return
}

// 点位-》查询配置
// 传递：driveid 设备 id, page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func Points_Config__Query(driveid uint, page uint, pageSize uint) (configs []Points_Config_type, err error) {
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
			Drive_Config.Name AS Drive_Name,
			IFNULL(Drive_Config.Collector_Id, 0) AS Collector_Id,
			IFNULL(Collector_Info.Name, '') AS Creation_Name,
	        IFNULL(Collector_Info.Uuid, '') AS Creation_Uuid
		FROM Points_Config
		INNER JOIN Drive_Config ON Points_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Info ON Drive_Config.Collector_Id = Collector_Info.Id	
	`
	var args []interface{} // 存储SQL参数，防止SQL注入

	// 2. 构建WHERE条件（统一处理，避免多个else if分支）
	if driveid != 0 {
		baseQuery += " WHERE `Points_Config`.`Drive_Id` = ?"
		args = append(args, driveid)
	}

	// 3. 构建分页条件（统一处理，避免重复逻辑）
	if page != 0 {
		// 分页计算：page从1开始的话，偏移量是 (page-1)*pageSize；page为0则不分页
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return nil, err
	}
	// 修复：移除多余的 ErrNoRows 判断（Query 不会返回 ErrNoRows，只会返回空结果集），并正确 defer
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
			&config.Collector.Id,
			&config.Collector.Name,
			&config.Collector.Uuid,
		)
		if err != nil {
			log.Print(err.Error())
			return nil, err
		}

		config.Description = Description.String
		configs = append(configs, config)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return configs, nil
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

	// 2. 批量插入 SQL（字段顺序严格对齐）
	baseQuery := `
		INSERT INTO Points_Config (
			Drive_Id,
			Tag,
			Description,
			RW_Cancel,
			Value_Type,
			Config,
			Creation_Time
		) VALUES
	`

	var args []interface{}
	var valuePlaceholders []string

	// 3. 遍历构建参数
	for i, cfg := range configs {
		// ========== 必传字段校验（修复版）==========
		if cfg.Drive_Id == 0 {
			return fmt.Errorf("批量新增失败：第%d条数据 Drive_Id 等于0", i)
		}
		if cfg.Tag == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Tag 不能为空", i+1)
		}
		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Config 不能为空", i+1)
		}
		// RW_Cancel 是 int 类型，不能判断 == ""
		// Value_Type 按需校验

		// 占位符
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?)")

		// ========== 字段顺序 必须和 INSERT 一致 ==========
		args = append(args,
			cfg.Drive_Id, // 1
			cfg.Tag,      // 2
			sql.NullString{String: cfg.Description, Valid: cfg.Description != ""}, // 3
			cfg.RW_Cancel,  // 4 int
			cfg.Value_Type, // 5 int
			cfg.Config,     // 6
			time.Now(),     // 7
		)
	}

	// 4. 拼接最终 SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 5. 执行
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 Points_Config 失败: %w", err)
	}

	for _, cfg := range configs {
		err = Drive_Config__Points_Length(cfg.Drive_Id, 1)
		if err != nil {
			return
		}
	}
	return
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

	log.Printf("批量更新成功，共更新%d条配置", len(configs))
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

		var drive_id uint
		drive_id, err = Points_Config__DriveId(id)
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

		err = Drive_Config__Points_Length(drive_id, -1)
		if err != nil {
			return
		}
	}
	return
}
