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

// 采集配置增加结构体
type Collector_Config_Add_type struct {
	Label   string // 标识
	Name    string // 设备名称
	User_Id uint   // 用户id
	Passwd  string // 密码
}

type Collector_Config_Update_type struct {
	Id      uint   // 采集 Id
	Label   string // 标识
	Name    string // 设备名称
	User_Id uint   // 用户id
	Passwd  string // 密码
}

// 采集配置结构体
type Collector_Config_type struct {
	Id                 uint      // 采集 Id
	Label              string    // 标识
	User_Id            uint      // 创建用户 id
	Creation_Time      time.Time // 创建时间
	Last_Modified_Time time.Time // 最后修改时间
	Name               string    // 设备名称
	Mqtt_Topic         string    // 命令
	Passwd             string    // 密码
}

// 采集-》查询数量
// 传递：page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Collector_Config__Count(page uint, pageSize uint) (count uint, err error) {
	// 1. 初始化 SQL（COUNT 查询不需要 LIMIT，否则统计的是当前页数量而非总数）
	baseQuery := "SELECT COUNT(`Id`) FROM `Collector_Config`"
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
		err = fmt.Errorf("[Collector_Config__Count] 查询失败 | SQL=%s | args=%v | err=%w",
			baseQuery, args, err)
		log.Print(err)
		return 0, err
	}

	return count, nil
}

// 采集-》查询配置（回调）
// 传递：page 页码，pageSize 每页数量，callback 回调函数
// 返回：err 错误
func Collector_Config__Query_Callback(page uint, pageSize uint, callback func(Collector_Config_type)) error {

	// 1. 初始化 SQL
	baseQuery := "SELECT `Id`, `Label`, `User_Id`, `Creation_Time`, `Last_Modified_Time`, `Name`, `Mqtt_Topic`, `Passwd` FROM `Collector_Config`"

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
		return err
	}
	// 修复：仅在 err == nil 时 defer close，避免 panic
	defer rows.Close()

	for rows.Next() {
		var cfg Collector_Config_type
		err = rows.Scan(
			&cfg.Id,                 // 采集 Id
			&cfg.Label,              // 标识
			&cfg.User_Id,            // 创建用户 id
			&cfg.Creation_Time,      // 创建时间
			&cfg.Last_Modified_Time, // 最后修改时间
			&cfg.Name,               // 设备名称
			&cfg.Mqtt_Topic,         // 命令
			&cfg.Passwd,             // 密码
		)
		if err != nil {
			log.Print(err.Error())
			return err
		}

		callback(cfg)
	}

	// 检查遍历过程中的错误
	if err = rows.Err(); err != nil {
		return err
	}

	return nil
}

// 采集-》查询配置
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func Collector_Config__Query(page uint, pageSize uint) (configs []Collector_Config_type, err error) {
	err = Collector_Config__Query_Callback(page, pageSize, func(cfg Collector_Config_type) {
		configs = append(configs, cfg)
	})
	return
}

// 采集-》搜索
// 传递：field quantity 数量，vague 搜索字段
// 返回：configs 配置，err 错误
func Collector_Config__Search_Name(quantity uint, value string) (configs []Collector_Config_type, err error) {
	// 1. 初始化 SQL
	baseQuery := "SELECT `Id`, `Label`, `User_Id`, `Creation_Time`, `Last_Modified_Time`, `Name`, `Mqtt_Topic`, `Passwd` FROM `Collector_Config` WHERE `Name` = ? LIMIT ?"

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, value, quantity)
	if err != nil {
		err = fmt.Errorf("ERROR 查询采集配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, []interface{}{value, quantity})
		log.Print(err)
		return
	}
	// 修复：仅在 err == nil 时 defer close，避免 panic
	defer rows.Close()

	for rows.Next() {
		var cfg Collector_Config_type
		err = rows.Scan(
			&cfg.Id,                 // 采集 Id
			&cfg.Label,              // 标识
			&cfg.User_Id,            // 创建用户 id
			&cfg.Creation_Time,      // 创建时间
			&cfg.Last_Modified_Time, // 最后修改时间
			&cfg.Name,               // 设备名称
			&cfg.Mqtt_Topic,         // 命令
			&cfg.Passwd,             // 密码
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		configs = append(configs, cfg)
	}

	// 检查遍历过程中的错误
	err = rows.Err()
	if err != nil {
		return
	}

	return
}

// 采集-》增加配置
// 传递：config 配置数组形式
// 返回：err 错误
func Collector_Config__Add(configs ...Collector_Config_Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		err = fmt.Errorf("批量新增失败：待新增配置列表为空")
		return
	}

	// 2. 遍历校验每个配置的参数合法性
	for i, cfg := range configs {
		// 可选：校验必填字段（Type/Name/Config非空，根据业务需求加）
		if cfg.Label == "" || cfg.User_Id == 0 || cfg.Name == "" {
			err = fmt.Errorf("批量新增失败：第%d条配置Label/User_Id/Name不能为空", i)
			return
		}
	}

	// 3. 拼接批量INSERT的SQL和参数
	baseQuery := "INSERT INTO `Collector_Config`() VALUES "
	var args []interface{}         // 存储所有参数
	var valuePlaceholders []string // 存储每个值组的占位符 (?, ?, ?, ?)

	// 遍历配置列表，拼接占位符和参数
	for _, cfg := range configs {
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?)")
		args = append(args, cfg.Label, cfg.User_Id, time.Now(), cfg.Name, "?")
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
func Collector_Config__Update(configs ...Collector_Config_Update_type) (err error) {
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

		if config.Label != "" {
			setClauses = append(setClauses, "`Label` = ?")
			args = append(args, config.Label)
		}

		if config.Name != "" {
			setClauses = append(setClauses, "`Name` = ?")
			args = append(args, config.Name)
		}

		if config.User_Id != 0 {
			setClauses = append(setClauses, "`User_Id` = ?")
			args = append(args, config.User_Id)
		}

		if config.Passwd != "" {
			setClauses = append(setClauses, "`Passwd` = ?")
			args = append(args, config.Passwd)
		}

		setClauses = append(setClauses, "`Passwd` = ?")
		args = append(args, config.Passwd)

		// 2.3 校验：至少有一个更新字段（Name/Config二选一）
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段 Name至少传一个非空值", idx+1)
			return
		}

		// 2.4 拼接SQL：WHERE条件指定ID
		query := fmt.Sprintf("UPDATE `Collector_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
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
func Collector_Config__Del(ids ...uint) (err error) {
	// 1. 遍历逐个
	for idx, id := range ids {
		// 1.1 单条配置参数校验
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}

		query := "DELETE FROM `Collector_Config` WHERE `Id` = ? "
		// 修改数据库
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s", idx, id, err, query)
			return
		}
	}
	return
}

// 采集-》更新 最后修改时间
// 传递：Uuid 采集器uuid heartbeat 心跳时间
// 返回：err 错误
func Collector_Config__Update__LastModifiedTime(Label string, t time.Time) (err error) {
	query := "UPDATE `Collector_Config` SET `Last_Modified_Time` = ? WHERE `Label` = ? "

	_, err = DB.Exec(query, t, Label)
	if err != nil {
		log.Printf("ERROR 最后修改时间 Label:%s %s", Label, err)
	}
	return
}

type Collector_Config__Query_MqttByLabel_type struct {
	Id         uint
	Label      string
	Mqtt_Topic string
	Passwd     string
}

// Collector_Config__QueryMqttByUuids 根据uuid查询 Mqtt_Topic、Passwd
func Collector_Config__Query_MqttByLabel(label string) (r Collector_Config__Query_MqttByLabel_type, err error) {
	if label == "" {
		err = fmt.Errorf("ERROR 查询采集配置失败，Label不能为空")
		return
	}

	baseQuery := `
		SELECT
			Id, Label, Mqtt_Topic, Passwd
		FROM
			Collector_Config
		WHERE
			Label = ?
	`

	// 执行查询
	err = DB.QueryRow(baseQuery, label).Scan(&r.Id, &r.Label, &r.Mqtt_Topic, &r.Passwd)
	if err == sql.ErrNoRows {
		err = fmt.Errorf("未找到Label=%s的记录", label)
	} else if err != nil {
		err = fmt.Errorf("查询失败: %w", err)
	}
	return
}

/*
***************驱动配置结构体***************
 */

/*
***************驱动配置结构体***************
 */

type CollectorGet_Drive_Config_type struct {
	Id            uint   // 递增id
	Collector_Id  uint   // 采集id
	Type          string // 驱动类型
	Config        string // json配置参数
	Points_Length uint   // 点位数量
}

type Drive_Config_Add_type struct {
	Id           uint   // 递增id
	Collector_Id uint   // 采集id
	Type         string // 驱动类型
	Name         string // 驱动名称
	Config       string // json配置参数
}

type Drive_Config_Update_type struct {
	Id     uint   // 驱动id
	Name   string // 驱动名称
	Config string // json配置参数
}

type Drive_Config_type struct {
	Id            uint      // 递增id
	Collector_Id  uint      // 采集id
	Type          string    // 驱动类型
	Name          string    // 驱动名称
	Config        string    // json配置参数
	Points_Length uint      // 点位数量
	Creation_Time time.Time // 创建时间
}

// 驱动 -》查询配置
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func CollectorGet_Drive_Config__Query(Label []string, Collector_Id []uint) (configs []CollectorGet_Drive_Config_type, err error) {
	if len(Collector_Id) == 0 {
		err = fmt.Errorf("ERROR 查询驱动配置失败，Collector_Id不能为空")
		return
	}

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Collector_Id,
			Drive_Config.Type,
			Drive_Config.Config,
			Drive_Config.Points_Length
		FROM
			Drive_Config
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`

	var whereConditions []string // 存储WHERE子句的条件片段
	var args []interface{}       // 存储SQL参数，防止注入

	if len(Label) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Label)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Config`.`Label` IN (%s)", placeholders))
		for _, id := range Label {
			args = append(args, id)
		}
	}

	if len(Collector_Id) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Collector_Id)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Drive_Config`.`Collector_Id` IN (%s)", placeholders))
		for _, id := range Collector_Id {
			args = append(args, id)
		}
	}

	// 4. 拼接WHERE子句
	finalQuery := baseQuery
	if len(whereConditions) > 0 {
		finalQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 5. 执行查询
	rows, err := DB.Query(finalQuery, args...)

	// 修复：先检查 err，若出错则 rows 为 nil，不能执行 defer close
	if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败，错误:%v, SQL:%s, 参数:%v", err, finalQuery, args)
		log.Print(err)
		return
	}

	// 只有在 rows 不为 nil 时才注册 defer 关闭
	defer rows.Close()

	for rows.Next() {
		var cfg CollectorGet_Drive_Config_type
		err = rows.Scan(
			&cfg.Id,            // 递增id
			&cfg.Collector_Id,  // 采集id
			&cfg.Type,          // 驱动类型
			&cfg.Config,        // json配置参数
			&cfg.Points_Length, // 点位数量
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		configs = append(configs, cfg)
	}

	// 检查迭代过程中是否发生错误
	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历驱动配置结果集失败，错误:%v", err)
		log.Print(err)
	}

	return
}

// 驱动-》查询数量
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Drive_Config__Count(Collector_Id []uint, page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `Drive_Config`"
	var whereConditions []string
	var args []interface{}

	// 处理多 Collector_Id IN 查询
	if len(Collector_Id) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Collector_Id)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Id` IN (%s)", placeholders))
		for _, id := range Collector_Id {
			args = append(args, id)
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
		log.Printf("[Drive_Config__Count] 无符合条件的数据 | Collector_Id=%v", Collector_Id)
	} else if err != nil {
		err = fmt.Errorf("[Drive_Config__Count] 查询失败 | Collector_Id=%v | SQL=%s | args=%v | err=%w", Collector_Id, baseQuery, args, err)
		log.Print(err)
	}

	return count, err
}

// 驱动 -》查询配置（回调）
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量，callback 回调函数
// 返回：err 错误
func Drive_Config__Query_Callback(Collector_Id []uint, page uint, pageSize uint, callback func(Drive_Config_type)) (err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Collector_Id,
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
	if len(Collector_Id) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Collector_Id)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Drive_Config`.`Collector_Id` IN (%s)", placeholders))
		for _, id := range Collector_Id {
			args = append(args, id)
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
		var cfg Drive_Config_type
		err = rows.Scan(
			&cfg.Id,            // 递增id
			&cfg.Collector_Id,  // 采集id
			&cfg.Type,          // 驱动类型
			&cfg.Name,          // 驱动名称
			&cfg.Config,        // json配置参数
			&cfg.Points_Length, // 点位数量
			&cfg.Creation_Time, // 创建时间
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		callback(cfg)
	}

	// 检查迭代过程中是否发生错误
	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历驱动配置结果集失败，错误:%v", err)
		log.Print(err)
	}

	return
}

// 驱动 -》查询配置
// 传递：driveType 驱动类型，page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func Drive_Config__Query(Collector_Id []uint, page uint, pageSize uint) (configs []Drive_Config_type, err error) {
	err = Drive_Config__Query_Callback(Collector_Id, page, pageSize, func(config Drive_Config_type) {
		configs = append(configs, config)
	})
	return
}

// 驱动-》搜索
// 传递： quantity 数量，vague 搜索字段
// 返回：configs 配置，err 错误
func Drive_Config__Search__Name(quantity uint, value string) (configs []Drive_Config_type, err error) {
	if quantity > 30 {
		quantity = 30
	}

	// 1. 初始化 SQL
	baseQuery := `
		SELECT
			Drive_Config.Id,
			Drive_Config.Collector_Id,
			Drive_Config.Type,
			Drive_Config.Name,
			Drive_Config.Config,
			Drive_Config.Points_Length,
			Drive_Config.Creation_Time
		FROM
			Drive_Config
	`

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
		var cfg Drive_Config_type
		err = rows.Scan(
			&cfg.Id,            // 递增id
			&cfg.Collector_Id,  // 采集id
			&cfg.Type,          // 驱动类型
			&cfg.Name,          // 驱动名称
			&cfg.Config,        // json配置参数
			&cfg.Points_Length, // 点位数量
			&cfg.Creation_Time, // 创建时间
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		configs = append(configs, cfg)
	}

	// 检查迭代过程中是否发生错误
	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历驱动配置结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

// 驱动-》增加配置
// 传递：config 配置数组形式
// 返回：err 错误
func Drive_Config__Add(configs ...Drive_Config_Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		return
	}
	// 3. 拼接批量INSERT的SQL和参数
	baseQuery := "INSERT INTO `Drive_Config`(`Id`, `Collector_Id`, `Type`, `Name`, `Config`, `Points_Length`, `Creation_Time`, `Update_Time`) VALUES "
	var args []interface{}
	var valuePlaceholders []string
	createTime := time.Now()

	// 遍历配置列表
	for _, cfg := range configs {

		if cfg.Id == 0 || cfg.Collector_Id == 0 || cfg.Type == "" || cfg.Name == "" || cfg.Config == "" {
			err = fmt.Errorf("批量新增失败：第%d条配置Type/Name/Config/Collector_Id不能为空 %v", cfg.Id, cfg)
			return
		}

		// 拼接占位符和参数
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			cfg.Id,           // 递增id
			cfg.Collector_Id, // 采集id
			cfg.Type,         // 驱动类型
			cfg.Name,         // 驱动名称
			cfg.Config,       // json配置参数
			createTime,
			createTime,
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

	t := time.Now()

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
		args = append(args, t)

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
	if len(ids) == 0 {
		return
	}
	// 去重
	slices.Sort(ids)
	ids = slices.Compact(ids)

	// 参数校验：不能有id=0
	for idx, id := range ids {
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 拼接 IN 的占位符
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, v := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	sql := fmt.Sprintf("DELETE FROM `Drive_Config` WHERE `Id` IN (%s)", strings.Join(placeholders, ","))

	_, err = tx.Exec(sql, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 批量删除失败，sql:%s, err:%w", sql, err)
		return err
	}

	return
}

// 驱动-》更新点位数量
// 传递：id 驱动ID, quantity 点位数量
// 返回：err 错误
func Drive_Config__Update__PointsLength(ids ...uint) (err error) {
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
		quantity, err = Point_Config__Count([]uint{}, []uint{id}, 0, 0)
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

const (
	Value_Type__Bool    = 1 // 常用
	Value_Type__Int8    = 2
	Value_Type__Uint8   = 3
	Value_Type__Int16   = 4
	Value_Type__Uint16  = 5
	Value_Type__Int32   = 6
	Value_Type__Uint32  = 7
	Value_Type__Int64   = 8
	Value_Type__Uint64  = 9
	Value_Type__Int     = 10 // 常用
	Value_Type__Uint    = 11 // 常用
	Value_Type__Float32 = 12
	Value_Type__Float64 = 13
	Value_Type__Float   = 14 // 常用
	Value_Type__String  = 15 // 常用
)

var (
	RW_Cancel_map = map[int]string{
		1: "N",
		2: "R",
		3: "W",
		4: "R/W",
	}

	Value_Type_map = map[int]string{
		Value_Type__Bool:    "bool", // 常用
		Value_Type__Int8:    "int8",
		Value_Type__Uint8:   "uint8",
		Value_Type__Int16:   "int16",
		Value_Type__Uint16:  "uint16",
		Value_Type__Int32:   "int32",
		Value_Type__Uint32:  "uint32",
		Value_Type__Int64:   "int64",
		Value_Type__Uint64:  "uint64",
		Value_Type__Int:     "int", // 常用
		Value_Type__Uint:    "uint",
		Value_Type__Float32: "float32",
		Value_Type__Float64: "float64",
		Value_Type__Float:   "float",  // 常用
		Value_Type__String:  "string", // 常用
	}
)

/*
***************点位配置结构体***************
 */

type CollectorGet_Point_Config_type struct {
	Id         uint   // 点位id
	Drive_Id   uint   // 驱动id
	Config     string // 配置信息
	RW_Cancel  int    // 读写方式
	Value_Type int    // 输出类型
}

// 点位配置增加结构体
type Point_Config_Add_type struct {
	Drive_Id    uint   // 驱动id
	Name        string // 点位名称
	Description string // 说明
	Config      string // 配置信息
	RW_Cancel   int    // 读写方式
	Value_Type  int    // 输出类型
}

// 点位配置更新结构体
type Point_Config_Update_type struct {
	Id          uint   // 点位id
	Drive_Id    uint   // 驱动id
	Name        string // 点位名称
	Description string // 说明
	Config      string // 配置信息
	RW_Cancel   int    // 读写方式
	Value_Type  int    // 输出类型
}

// 点位配置结构体
type Point_Config_type struct {
	Id            uint      // 点位id
	Drive_Id      uint      // 驱动id
	Name          string    // 点位名称
	Description   string    // 说明
	Config        string    // 配置信息
	RW_Cancel     int       // 读写方式
	Value_Type    int       // 输出类型
	Creation_Time time.Time // 创建时间
}

func CollectorGet_Point_Config__Query(Label []string, collectorId []uint, driveid []uint) (configs []CollectorGet_Point_Config_type, err error) {
	if len(collectorId) == 0 && len(driveid) == 0 {
		err = fmt.Errorf("ERROR 查询点位配置失败，参数为空")
		log.Print(err)
		return
	}

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT 
			Point_Config.Id,
			Point_Config.Drive_Id,
			Point_Config.Config,
			Point_Config.RW_Cancel,
			Point_Config.Value_Type
		FROM Point_Config
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var whereConditions []string
	var args []interface{} // 存储SQL参数，防止SQL注入

	if len(Label) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Label)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Config`.`Label` IN (%s)", placeholders))
		for _, id := range Label {
			args = append(args, id)
		}
	}

	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Point_Config.Drive_Id IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	// 新增：支持多个 collectorId
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Collector_Config.Id IN (%s)", placeholders))
		for _, id := range collectorId {
			args = append(args, id)
		}
	}

	// 拼接 WHERE 条件
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cfg CollectorGet_Point_Config_type
		err = rows.Scan(
			&cfg.Id,
			&cfg.Drive_Id,
			&cfg.Config,     // 配置信息
			&cfg.RW_Cancel,  // 读写方式,
			&cfg.Value_Type, // 输出类型
		)
		if err != nil {
			log.Print(err.Error())
			return
		}
		configs = append(configs, cfg)
	}

	err = rows.Err()
	return
}

// 点位-》查询数量
// 传递：driveid 设备id，page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Point_Config__Count(collectorId []uint, driveid []uint, page uint, pageSize uint) (Count uint, err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			COUNT(Point_Config.Id) 
		FROM Point_Config
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var args []interface{} // 存储SQL参数，防止SQL注入
	var whereConditions []string

	// 2. 构建WHERE条件（支持多个 driveid）
	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Config`.`Drive_Id` IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	// 新增：支持多个 collectorId
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Config`.`Id` IN (%s)", placeholders))
		for _, id := range collectorId {
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
func Point_Config__Query_Callback(collectorId []uint, driveid []uint, page uint, pageSize uint, callback func(Point_Config_type)) (err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT 
			Point_Config.Id,
			Point_Config.Drive_Id,
			Point_Config.Name,
			Point_Config.Description,
			Point_Config.Config,
			Point_Config.RW_Cancel,
			Point_Config.Value_Type,
			Point_Config.Creation_Time
		FROM Point_Config
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var whereConditions []string
	var args []interface{} // 存储SQL参数，防止SQL注入

	// 2. 构建WHERE条件（支持多个 driveid：IN 查询）
	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Point_Config.Drive_Id IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	// 新增：支持多个 collectorId
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Collector_Config.Id IN (%s)", placeholders))
		for _, id := range collectorId {
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
			cfg         Point_Config_type
			Description sql.NullString
		)
		err = rows.Scan(
			&cfg.Id,
			&cfg.Drive_Id,
			&cfg.Name,
			&cfg.Description,   // 说明
			&cfg.Config,        // 配置信息
			&cfg.RW_Cancel,     // 读写方式,
			&cfg.Value_Type,    // 输出类型
			&cfg.Creation_Time, // 创建时间
		)
		if err != nil {
			log.Print(err.Error())
			return err
		}
		cfg.Description = Description.String
		callback(cfg)
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
func Point_Config__Query(collectorId []uint, driveid []uint, page uint, pageSize uint) (configs []Point_Config_type, err error) {
	err = Point_Config__Query_Callback(collectorId, driveid, page, pageSize, func(config Point_Config_type) {
		configs = append(configs, config)
	})
	return
}

// 点位-》查询设备id
// 传递：Id 点位id
// 返回：drive_id 设备id err 错误

// Point_Config__DriveIds 批量查询：传入多个Id，返回 []{Id,Drive_Id}
type Point_Config__Query__DriveIds_type struct {
	Id       uint
	Drive_Id uint
}

func Point_Config__Query__DriveIds(ids ...uint) (list []Point_Config__Query__DriveIds_type, err error) {
	if len(ids) == 0 {
		return
	}

	// 去重
	slices.Sort(ids)
	ids = slices.Compact(ids)

	// 生成 ?,?,?
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))

	for _, v := range ids {
		if v == 0 {
			err = fmt.Errorf("ERROR Id 不能等于0")
			return
		}
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	query := fmt.Sprintf(`
			SELECT
				Id,
				Drive_Id
			FROM
				Point_Config
			WHERE
				Id IN (%s)
		`, strings.Join(placeholders, ","))

	rows, err := DB.Query(query, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 批量查询Drive_Id失败: %w", err)
		log.Println(err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item Point_Config__Query__DriveIds_type
		if err = rows.Scan(&item.Id, &item.Drive_Id); err != nil {
			err = fmt.Errorf("ERROR scan行失败: %w", err)
			log.Println(err.Error())
			return nil, err
		}
		list = append(list, item)
	}

	// 检查rows迭代错误
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("ERROR rows迭代错误: %w", err)
		log.Println(err.Error())
		return nil, err
	}

	return list, nil
}

// 点位-》增加配置
// 传递：config 配置数组形式
// 返回：err 错误
func Point_Config__Add(configs ...Point_Config_Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}

	// 3. SQL 插入（包含 Id 字段）
	baseQuery := `
		INSERT INTO Point_Config (
			Drive_Id,
			Name,
			Description,
			Config,
			RW_Cancel,
			Value_Type,
			Creation_Time
		) VALUES
	`

	var (
		args              []interface{}
		valuePlaceholders []string
	)
	t := time.Now()
	exist_Drive_Id := make(map[uint]bool)

	// 4. 构建批量参数
	for i, cfg := range configs {

		if cfg.Drive_Id == 0 {
			return fmt.Errorf("批量新增失败：第%d条数据 Drive_Id 等于0", i)
		}
		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Config 不能为空", i)
		}

		if exist_Drive_Id[cfg.Drive_Id] {
			return fmt.Errorf("批量新增失败：第%d条数据 Drive_Id 重复", i)
		}

		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			cfg.Drive_Id,
			cfg.Name,
			sql.NullString{
				String: cfg.Description,
				Valid:  cfg.Description != "" && cfg.Description != "null",
			},
			cfg.Config,
			cfg.RW_Cancel,
			cfg.Value_Type,
			t,
		)

		exist_Drive_Id[cfg.Drive_Id] = true
	}

	// 5. 拼接 SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 6. 执行插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 Point_Config 失败: %w", err)
	}

	// 7. 更新点位长度统计
	ids := make([]uint, 0, len(exist_Drive_Id))
	for id := range exist_Drive_Id {
		ids = append(ids, id)
	}
	err = Drive_Config__Update__PointsLength(ids...)
	if err != nil {
		return
	}

	return nil
}

// 点位-》修改配置
// 传递：config 配置
// 返回：conid 获取自增的Id，err 错误
func Point_Config__Update(configs ...Point_Config_Update_type) (err error) {
	if len(configs) == 0 {
		return
	}

	// 开启事务
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	// defer处理回滚/提交
	defer func() {
		r := recover()
		if r != nil {
			_ = tx.Rollback()
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var sqlPieces []string
	var allArgs []interface{}

	exist_Drive_Id := make(map[uint]bool)
	now := time.Now()

	for i, cfg := range configs {
		if cfg.Id == 0 {
			err = fmt.Errorf("批量更新失败：第%d条配置Id不能为空", i)
			return
		}

		if cfg.Drive_Id == 0 {
			err = fmt.Errorf("批量更新失败：第%d条配置Drive_Id等于0", i)
			return
		}

		if exist_Drive_Id[cfg.Drive_Id] {
			err = fmt.Errorf("第%d条配置Drive_Id重复", i)
			return
		}

		exist_Drive_Id[cfg.Drive_Id] = true

		var setClauses []string

		if cfg.Drive_Id != 0 {
			setClauses = append(setClauses, "`Drive_Id`=?")
			allArgs = append(allArgs, cfg.Drive_Id)
		}

		// Name：""不修改；字符串"null" 设置数据库NULL
		if cfg.Name != "" {
			setClauses = append(setClauses, "`Name`=?")
			allArgs = append(allArgs, cfg.Name)
		}

		if cfg.Description != "" {
			setClauses = append(setClauses, "`Description` = ?")
			allArgs = append(allArgs, sql.NullString{
				String: cfg.Description,
				Valid:  cfg.Description != "null",
			})
		}

		if cfg.Config != "" {
			setClauses = append(setClauses, "`Config`=?")
			allArgs = append(allArgs, cfg.Config)
		}

		if cfg.RW_Cancel != 0 {
			setClauses = append(setClauses, "`RW_Cancel`=?")
			allArgs = append(allArgs, cfg.RW_Cancel)
		}

		if cfg.Value_Type != 0 {
			setClauses = append(setClauses, "`Value_Type`=?")
			allArgs = append(allArgs, cfg.Value_Type)
		}

		if len(setClauses) == 0 {
			err = fmt.Errorf("第%d条配置未指定任何更新字段，至少传一个", i)
			return
		}

		setClauses = append(setClauses, "`Update_Time`=?")
		allArgs = append(allArgs, now)

		sql := fmt.Sprintf("UPDATE `Point_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		allArgs = append(allArgs, cfg.Id)
		sqlPieces = append(sqlPieces, sql)
	}

	fullSql := strings.Join(sqlPieces, ";")

	_, err = tx.Exec(fullSql, allArgs...)
	if err != nil {
		err = fmt.Errorf("批量更新执行失败, SQL:%s, args:%v, err:%w", fullSql, allArgs, err)
		return err
	}

	log.Printf("批量更新成功，共更新%d条配置", len(configs))
	return
}

// 点位-》删除配置
// 传递：ids 删除的id数组
// 返回：err 错误
func Point_Config__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	// 去重
	slices.Sort(ids)
	ids = slices.Compact(ids)

	// 参数校验：不能有id=0
	for idx, id := range ids {
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
	}

	// 查询设备id更新数量
	var driveId_list []Point_Config__Query__DriveIds_type
	driveId_list, err = Point_Config__Query__DriveIds(ids...)
	if err != nil {
		return
	}
	if len(driveId_list) == 0 {
		err = fmt.Errorf("ERROR 删除点位前 查询设备id失败")
		return
	}

	// 删除配置
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 拼接 IN 的占位符
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, v := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	sql := fmt.Sprintf("DELETE FROM `Point_Config` WHERE `Id` IN (%s)", strings.Join(placeholders, ","))

	_, err = tx.Exec(sql, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 批量删除失败，sql:%s, err:%w", sql, err)
		return err
	}

	// 更新点位数量
	var driveIds []uint
	for _, v := range driveId_list {
		driveIds = append(driveIds, v.Drive_Id)
	}
	err = Drive_Config__Update__PointsLength(driveIds...)
	if err != nil {
		return
	}
	return
}

/*
***************报警***************
 */

type CollectorGet_Alarm_Config_type struct {
	Id       uint   // 报警id
	Point_Id uint   // 点位id
	Config   string // 报警配置
	Group    int    // 报警组
}

// 配置增加结构体
type Alarm_Config_Add_type struct {
	Point_Id uint   // 点位id
	Config   string // 报警配置
	Name     string // 报警名称
	Group    int    // 报警组
}

// 配置更新结构体
type Alarm_Config_Update_type struct {
	Id       uint   // 报警id
	Point_Id uint   // 点位id
	Config   string // 报警配置
	Name     string // 报警名称
	Group    int    // 报警组
}

// 配置结构体
type Alarm_Config_type struct {
	Id            uint      // 报警id
	Point_Id      uint      // 点位id
	Config        string    // 报警配置
	Name          string    // 报警名称
	Group         int       // 报警组
	Creation_Time time.Time // 报警创建时间
}

func CollectorGet_Alarm_Config__Query(Label []string, collectorId []uint) (configs []CollectorGet_Alarm_Config_type, err error) {
	if len(collectorId) == 0 || len(Label) == 0 {
		err = fmt.Errorf("ERROR 查询点位配置失败，参数为空")
		log.Print(err)
		return
	}

	baseQuery := `
		SELECT
			Alarm_Config.Id,
			Alarm_Config.Point_Id,
			Alarm_Config.Config,
			Alarm_Config.Group
		FROM Alarm_Config
		INNER JOIN Point_Config ON Alarm_Config.Point_Id = Point_Config.Id
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var whereConditions []string
	var args []interface{}

	if len(Label) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Label)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Config`.`Label` IN (%s)", placeholders))
		for _, id := range Label {
			args = append(args, id)
		}
	}

	// 新增：支持多个 collectorId
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Collector_Config.Id IN (%s)", placeholders))
		for _, id := range collectorId {
			args = append(args, id)
		}
	}

	// 拼接 WHERE 条件
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 4. 执行查询
	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cfg CollectorGet_Alarm_Config_type
		err = rows.Scan(
			&cfg.Id,
			&cfg.Point_Id,
			&cfg.Config,
			&cfg.Group,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}
		configs = append(configs, cfg)
	}

	err = rows.Err()
	return
}

// 报警-》查询数量
// 传递：driveid 设备id，page 页码，pageSize 每页数量
// 返回：Count 数量，err 错误
func Alarm_Config__Count(collectorId []uint, driveid []uint, pointid []uint) (Count uint, err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			COUNT(Alarm_Config.Id)
		FROM Alarm_Config
		INNER JOIN Point_Config ON Alarm_Config.Point_Id = Point_Config.Id
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var args []interface{} // 存储SQL参数，防止SQL注入
	var whereConditions []string

	// 新增：支持多个 collectorId
	if len(pointid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Config`.`Id` IN (%s)", placeholders))
		for _, id := range pointid {
			args = append(args, id)
		}
	}

	// 2. 构建WHERE条件（支持多个 driveid）
	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Config`.`Drive_Id` IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	// 新增：支持多个 collectorId
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Config`.`Id` IN (%s)", placeholders))
		for _, id := range collectorId {
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

// 报警-》查询配置（回调）
// 传递：driveid 设备id，page 页码，pageSize 每页数量，callback 回调函数
// 返回：err 错误
func Alarm_Config__Query_Callback(collectorId []uint, driveid []uint, pointid []uint, page uint, pageSize uint, callback func(Alarm_Config_type)) (err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Alarm_Config.Id,
			Alarm_Config.Point_Id,
			Alarm_Config.Config,
			Alarm_Config.Name,
			Alarm_Config.Group,
			Alarm_Config.Creation_Time
		FROM Alarm_Config
		INNER JOIN Point_Config ON Alarm_Config.Point_Id = Point_Config.Id
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var whereConditions []string
	var args []interface{} // 存储SQL参数，防止SQL注入

	// 新增：支持多个 collectorId
	if len(pointid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Config`.`Id` IN (%s)", placeholders))
		for _, id := range pointid {
			args = append(args, id)
		}
	}

	// 2. 构建WHERE条件（支持多个 driveid：IN 查询）
	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Point_Config.Drive_Id IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	// 新增：支持多个 collectorId
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Collector_Config.Id IN (%s)", placeholders))
		for _, id := range collectorId {
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
		var cfg Alarm_Config_type
		err = rows.Scan(
			&cfg.Id,            // 报警id
			&cfg.Point_Id,      // 点位id
			&cfg.Config,        // 报警配置
			&cfg.Name,          // 报警名称
			&cfg.Group,         // 报警组
			&cfg.Creation_Time, // 报警创建时间
		)
		if err != nil {
			log.Print(err.Error())
			return err
		}
		callback(cfg)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

func Alarm_Config__Query(collectorId []uint, driveid []uint, pointid []uint, page uint, pageSize uint) (r []Alarm_Config_type, err error) {
	err = Alarm_Config__Query_Callback(collectorId, driveid, pointid, page, pageSize, func(cfg Alarm_Config_type) {
		r = append(r, cfg)
	})
	return
}

// 报警-》增加配置
// 传递：config 配置数组形式
// 返回：err 错误
func Alarm_Config__Add(configs ...Alarm_Config_Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}

	// 3. SQL 插入（包含 Id 字段）
	baseQuery := `
		INSERT INTO Alarm_Config (
			Point_Id,
			Config,
			Name,
			Group,
			Creation_Time
		) VALUES
	`

	var (
		args              []interface{}
		valuePlaceholders []string
	)
	t := time.Now()
	exist_Alarm_Id := make(map[uint]bool)

	// 4. 构建批量参数
	for i, cfg := range configs {
		if cfg.Point_Id == 0 {
			return fmt.Errorf("批量新增失败：第[%d]条数据 Point_Id 等于0", i)
		}

		if cfg.Group == 0 {
			return fmt.Errorf("批量新增失败：点位[%d]数据 Group 不能等于0", cfg.Point_Id)
		}

		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：点位[%d]数据 Config 不能为空", cfg.Point_Id)
		}

		if exist_Alarm_Id[cfg.Point_Id] {
			return fmt.Errorf("批量新增失败：点位[%d]数据重复", cfg.Point_Id)
		}

		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?)")
		args = append(args,
			cfg.Point_Id,
			cfg.Config,
			cfg.Name,
			cfg.Group,
			t,
		)

		exist_Alarm_Id[cfg.Point_Id] = true
	}

	// 5. 拼接 SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 6. 执行插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 报警配置 失败: %w", err)
	}

	return nil
}

// 点位-》修改配置
// 传递：config 配置
// 返回：conid 获取自增的Id，err 错误
func Alarm_Config__Update(configs ...Alarm_Config_Update_type) (err error) {
	if len(configs) == 0 {
		return
	}

	// 开启事务
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	// defer处理回滚/提交
	defer func() {
		r := recover()
		if r != nil {
			_ = tx.Rollback()
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var sqlPieces []string
	var allArgs []interface{}

	exist_Point_Id := make(map[uint]bool)
	t := time.Now()

	for i, cfg := range configs {
		if cfg.Id == 0 {
			err = fmt.Errorf("批量更新失败：第[%d]条配置Id不能为空", i)
			return
		}
		if exist_Point_Id[cfg.Id] {
			err = fmt.Errorf("重复配置 Id[%d]", cfg.Id)
			return
		}

		exist_Point_Id[cfg.Id] = true

		var setClauses []string

		if cfg.Point_Id != 0 {
			setClauses = append(setClauses, "`Point_Id`=?")
			allArgs = append(allArgs, cfg.Point_Id)
		}

		if cfg.Config != "" {
			setClauses = append(setClauses, "`Config`=?")
			allArgs = append(allArgs, cfg.Config)
		}

		if cfg.Name != "" {
			setClauses = append(setClauses, "`Name`=?")
			allArgs = append(allArgs, cfg.Name)
		}

		if cfg.Group != 0 {
			setClauses = append(setClauses, "`Config`=?")
			allArgs = append(allArgs, cfg.Config)
		}

		if len(setClauses) == 0 {
			err = fmt.Errorf("点位[%d]未指定任何更新字段，至少传一个", cfg.Id)
			return
		}

		setClauses = append(setClauses, "`Creation_Time`=?")
		allArgs = append(allArgs, t)

		sql := fmt.Sprintf("UPDATE `Alarm_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		allArgs = append(allArgs, cfg.Id)
		sqlPieces = append(sqlPieces, sql)
	}

	fullSql := strings.Join(sqlPieces, ";")

	_, err = tx.Exec(fullSql, allArgs...)
	if err != nil {
		err = fmt.Errorf("批量更新执行失败, SQL:%s, args:%v, err:%w", fullSql, allArgs, err)
		return err
	}

	log.Printf("批量更新成功，共更新%d条配置", len(configs))
	return
}

// 点位-》删除配置
// 传递：ids 删除的id数组
// 返回：err 错误
func Alarm_Config__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	// 去重
	slices.Sort(ids)
	ids = slices.Compact(ids)

	// 删除配置
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 拼接 IN 的占位符
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, v := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	sql := fmt.Sprintf("DELETE FROM `Alarm_Config` WHERE `Id` IN (%s)", strings.Join(placeholders, ","))

	_, err = tx.Exec(sql, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 批量删除失败，sql:%s, err:%w", sql, err)
		return err
	}
	return
}

/*
***************历史配置***************
 */

type CollectorGet_History_Config_type struct {
	Id       uint   // 历史id
	Point_Id uint   // 点位id
	Config   string // 历史配置
}

// 配置增加结构体
type History_Config_Add_type struct {
	Point_Id uint   // 点位id
	Config   string // 历史配置
}

// 配置更新结构体
type History_Config_Update_type struct {
	Id       uint   // 历史id
	Point_Id uint   // 点位id
	Config   string // 历史配置
}

// 配置结构体
type History_Config_type struct {
	Id            uint      // 历史id
	Point_Id      uint      // 点位id
	Config        string    // 历史配置
	Creation_Time time.Time // 创建时间
}

func CollectorGet_History_Config(Label []string, collectorId []uint) (r []CollectorGet_History_Config_type, err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			History_Config.Id,
			History_Config.Point_Id,
			History_Config.Config,
			History_Config.Creation_Time
		FROM History_Config
		INNER JOIN Point_Config ON History_Config.Point_Id = Point_Config.Id
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var whereConditions []string
	var args []interface{} // 存储SQL参数，防止SQL注入

	if len(Label) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Label)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Config`.`Label` IN (%s)", placeholders))
		for _, id := range Label {
			args = append(args, id)
		}
	}
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Collector_Config.Id IN (%s)", placeholders))
		for _, id := range collectorId {
			args = append(args, id)
		}
	}

	// 拼接 WHERE 条件
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	// 4. 执行查询
	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cfg CollectorGet_History_Config_type
		err = rows.Scan(
			&cfg.Id,       // 历史id
			&cfg.Point_Id, // 点位id
			&cfg.Config,   // 历史配置
		)
		if err != nil {
			log.Print(err.Error())
			return
		}
		r = append(r, cfg)
	}

	err = rows.Err()
	return
}

func History_Config__Count(collectorId []uint, driveid []uint, pointid []uint) (Count uint, err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			COUNT(History_Config.Id)
		FROM History_Config
		INNER JOIN Point_Config ON History_Config.Point_Id = Point_Config.Id
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var args []interface{} // 存储SQL参数，防止SQL注入
	var whereConditions []string

	// 新增：支持多个 collectorId
	if len(pointid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Config`.`Id` IN (%s)", placeholders))
		for _, id := range pointid {
			args = append(args, id)
		}
	}

	// 2. 构建WHERE条件（支持多个 driveid）
	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Config`.`Drive_Id` IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	// 新增：支持多个 collectorId
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Collector_Config`.`Id` IN (%s)", placeholders))
		for _, id := range collectorId {
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

func History_Config__Query_Callback(collectorId []uint, driveid []uint, pointid []uint, page uint, pageSize uint, callback func(History_Config_type)) (err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			History_Config.Id,
			History_Config.Point_Id,
			History_Config.Config,
			History_Config.Creation_Time
		FROM History_Config
		INNER JOIN Point_Config ON History_Config.Point_Id = Point_Config.Id
		INNER JOIN Drive_Config ON Point_Config.Drive_Id = Drive_Config.Id
		INNER JOIN Collector_Config ON Drive_Config.Collector_Id = Collector_Config.Id
	`
	var whereConditions []string
	var args []interface{} // 存储SQL参数，防止SQL注入

	// 新增：支持多个 collectorId
	if len(pointid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Config`.`Id` IN (%s)", placeholders))
		for _, id := range pointid {
			args = append(args, id)
		}
	}

	// 2. 构建WHERE条件（支持多个 driveid：IN 查询）
	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Point_Config.Drive_Id IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	// 新增：支持多个 collectorId
	if len(collectorId) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(collectorId)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("Collector_Config.Id IN (%s)", placeholders))
		for _, id := range collectorId {
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
		var cfg History_Config_type
		err = rows.Scan(
			&cfg.Id,            // 历史id
			&cfg.Point_Id,      // 点位id
			&cfg.Config,        // 历史配置
			&cfg.Creation_Time, // 创建时间
		)
		if err != nil {
			log.Print(err.Error())
			return err
		}
		callback(cfg)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

func History_Config__Query(collectorId []uint, driveid []uint, pointid []uint, page uint, pageSize uint) (r History_Config_type, err error) {
	err = History_Config__Query_Callback(collectorId, driveid, pointid, page, pageSize, func(cfg History_Config_type) {
		r = cfg
	})
	return
}

func History_Config__Add(configs ...History_Config_Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}

	// 3. SQL 插入（包含 Id 字段）
	baseQuery := `
		INSERT INTO History_Config (
			Point_Id,
			Config,
			Creation_Time
		) VALUES
	`

	var (
		args              []interface{}
		valuePlaceholders []string
	)
	t := time.Now()
	exist_History_Id := make(map[uint]bool)

	// 4. 构建批量参数
	for i, cfg := range configs {
		if cfg.Point_Id == 0 {
			return fmt.Errorf("批量新增失败：第[%d]条数据 Point_Id 等于0", i)
		}

		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：点位[%d]数据 Config 不能为空", cfg.Point_Id)
		}
		if exist_History_Id[cfg.Point_Id] {
			return fmt.Errorf("批量新增失败：点位[%d]数据重复", cfg.Point_Id)
		}

		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?)")
		args = append(args,
			cfg.Point_Id,
			cfg.Config,
			t,
		)

		exist_History_Id[cfg.Point_Id] = true
	}

	// 5. 拼接 SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 6. 执行插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 历史配置 失败: %w", err)
	}

	return nil
}

func History_Config__Update(configs ...History_Config_Update_type) (err error) {
	if len(configs) == 0 {
		return
	}

	// 开启事务
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	// defer处理回滚/提交
	defer func() {
		r := recover()
		if r != nil {
			_ = tx.Rollback()
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var sqlPieces []string
	var allArgs []interface{}

	exist_History_Id := make(map[uint]bool)
	t := time.Now()

	for i, cfg := range configs {
		if cfg.Id == 0 {
			err = fmt.Errorf("批量更新失败：第[%d]条配置Id不能为空", i)
			return
		}
		if exist_History_Id[cfg.Id] {
			err = fmt.Errorf("重复配置 Id[%d]", cfg.Id)
			return
		}

		exist_History_Id[cfg.Id] = true

		var setClauses []string

		if cfg.Point_Id != 0 {
			setClauses = append(setClauses, "`Point_Id`=?")
			allArgs = append(allArgs, cfg.Point_Id)
		}

		if cfg.Config != "" {
			setClauses = append(setClauses, "`Config`=?")
			allArgs = append(allArgs, cfg.Config)
		}

		if len(setClauses) == 0 {
			err = fmt.Errorf("点位[%d]未指定任何更新字段，至少传一个", cfg.Id)
			return
		}

		setClauses = append(setClauses, "`Creation_Time`=?")
		allArgs = append(allArgs, t)

		sql := fmt.Sprintf("UPDATE `History_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		allArgs = append(allArgs, cfg.Id)
		sqlPieces = append(sqlPieces, sql)
	}

	fullSql := strings.Join(sqlPieces, ";")

	_, err = tx.Exec(fullSql, allArgs...)
	if err != nil {
		err = fmt.Errorf("批量更新执行失败, SQL:%s, args:%v, err:%w", fullSql, allArgs, err)
		return err
	}

	log.Printf("批量更新成功，共更新%d条配置", len(configs))
	return
}

func History_Config__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	// 去重
	slices.Sort(ids)
	ids = slices.Compact(ids)

	// 删除配置
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 拼接 IN 的占位符
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, v := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	sql := fmt.Sprintf("DELETE FROM `History_Config` WHERE `Id` IN (%s)", strings.Join(placeholders, ","))

	_, err = tx.Exec(sql, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 批量删除失败，sql:%s, err:%w", sql, err)
		return err
	}
	return
}
