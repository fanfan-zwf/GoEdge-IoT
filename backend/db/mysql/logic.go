/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: mysql 用户逻辑
 */

package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

/*
***************驱动配置结构体***************
 */

type Drive_Config_type struct {
	Id            uint   // 驱动id
	Type          string // 驱动类型
	Name          string // 驱动名称
	Points_Length uint   // 点位数量
	Config        string // json配置参数
}

// 点位-》查询配置
// 传递: driveid 设备id, page 页码, pageSize 每页数量
// 返回: configs 配置, err 错误
func Drive_Config__Query_DriveType(drive_type string) (configs []Drive_Config_type, err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := "SELECT `Id`, `Type`, `Name`, `Config`, `Points_Length` FROM `Drive_Config` WHERE `Type` = ?"
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
// 传递: driveid 驱动id
// 返回: configs 配置, err 错误
func Drive_Config__Query_DriveId(driveid uint) (config Drive_Config_type, err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := "SELECT `Id`, `Type`, `Name`, `Config`, `Points_Length` FROM `Drive_Config` WHERE `Id` = ?"

	// 2. 执行查询（统一处理，减少重复代码）
	err = DB.QueryRow(baseQuery, driveid).Scan(
		&config.Id,
		&config.Type,
		&config.Name,
		&config.Config,
		&config.Points_Length,
	)

	return
}

// 驱动-》查询数量
// 传递: driveType 驱动类型, page 页码, pageSize 每页数量
// 返回: Count 数量, err 错误
func Drive_Config__Count(driveType string, page uint, pageSize uint) (Count uint, err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := "SELECT COUNT(`Id`) FROM `Drive_Config`"
	var args []interface{} // 存储SQL参数，防止SQL注入

	// 2. 构建WHERE条件（统一处理，避免多个else if分支）
	if driveType != "" {
		baseQuery += " WHERE `Type` = ?"
		args = append(args, driveType)
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
		// log.Printf("查询驱动配置无数据，驱动类型：%s, 分页%d/%d", driveType, page, pageSize)
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败，错误：%v, SQL:%s, 参数:%v", err, baseQuery, args)
		return
	}
	return
}

// 驱动-》查询配置
// 传递: driveType 驱动类型, page 页码, pageSize 每页数量
// 返回: configs 配置, err 错误
func Drive_Config__Query(driveType string, page uint, pageSize uint) (configs []Drive_Config_type, err error) {

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := "SELECT `Id`, `Type`, `Name`, `Config`, `Points_Length` FROM `Drive_Config`"
	var args []interface{} // 存储SQL参数，防止SQL注入

	// 2. 构建WHERE条件（统一处理，避免多个else if分支）
	if driveType != "" {
		baseQuery += " WHERE `Type` = ?"
		args = append(args, driveType)
	}

	// 3. 构建分页条件（统一处理，避免重复逻辑）
	if page != 0 {
		// 分页计算：page从1开始的话，偏移量是 (page-1)*pageSize；page为0则不分页
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	// 4. 执行查询（统一处理，减少重复代码）
	rows, err := DB.Query(baseQuery, args...)

	// 区分无数据和查询错误，日志补充上下文便于排查
	if err == sql.ErrNoRows {
		// log.Printf("查询驱动配置无数据，驱动类型：%s, 分页%d/%d", driveType, page, pageSize)
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败, 错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}

	if err == sql.ErrNoRows {
		return
	} else if err != nil {
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
		var Config Drive_Config_type
		err = rows.Scan(
			&Config.Id,
			&Config.Type,
			&Config.Name,
			&Config.Config,
			&Config.Points_Length,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		configs = append(configs, Config)
	}
	return
}

// 驱动-》增加配置
// 传递: config 配置数组形式
// 返回: err 错误
func Drive_Config__Add(configs ...Drive_Config_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		err = fmt.Errorf("批量新增失败：待新增配置列表为空")
		return
	}

	// 2. 遍历校验每个配置的参数合法性
	for i, cfg := range configs {
		if cfg.Id != 0 || cfg.Points_Length != 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置参数错误 Id=%d/Points_Length=%d, 必须为0 ",
				i, cfg.Id, cfg.Points_Length)
			return
		}
		// 可选：校验必填字段（Type/Name/Config非空，根据业务需求加）
		if cfg.Type == "" || cfg.Name == "" {
			err = fmt.Errorf("批量新增失败：第%d条配置Type/Name不能为空", i)
			return
		}
	}

	// 3. 拼接批量INSERT的SQL和参数
	baseQuery := "INSERT INTO `Drive_Config`(`Type`, `Name`, `Config`) VALUES "
	var args []interface{}         // 存储所有参数
	var valuePlaceholders []string // 存储每个值组的占位符 (?, ?, ?)

	// 遍历配置列表，拼接占位符和参数
	for _, cfg := range configs {
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?)")
		args = append(args, cfg.Type, cfg.Name, cfg.Config)
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
// 传递: config 配置
// 返回: conid 获取自增的Id, err 错误
func Drive_Config__Update(configs ...Drive_Config_type) (err error) {
	// 1. 空列表校验
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	// 2. 遍历逐个更新
	for idx, config := range configs {
		// 2.1 单条配置参数校验
		if config.Id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
		if config.Points_Length != 0 {
			err = fmt.Errorf("ERROR 第%d条配置Points_Length不允许手动赋值 值:%d", idx, config.Points_Length)
			return
		}

		// 2.2 动态拼接SET子句
		var setClauses []string
		var args []interface{}
		if config.Config != "" {
			if !json.Valid([]byte(config.Config)) {
				err = fmt.Errorf("ERROR 第%d条配置非法JSON:%s", idx+1, config.Config)
				return
			}
			setClauses = append(setClauses, "`Config` = ?")
			args = append(args, config.Config)
		}

		// 2.3 校验更新字段
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段 Type/Name/Config至少传一个", idx+1)
			return
		}

		// 2.4 拼接SQL并执行
		query := fmt.Sprintf("UPDATE `Drive_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		args = append(args, config.Id)

		_, err = DB.Exec(query, args...)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s, 参数:%v", idx, config.Id, err, query, args)
			return
		}
	}

	log.Printf("批量更新成功，共更新%d条配置", len(configs))
	return
}

// 驱动-》删除配置
// 传递: ids 删除的id数组
// 返回: err 错误
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

/*
***************点位配置结构体***************
 */
type Points_Config_type struct {
	Id          uint   // 点位id
	Drive_Id    uint   // 驱动id唯一标识符
	Tag         string // 点位标识
	Drive_Type  string // 驱动类型
	Description string // 点位描述
	RW_Cancel   string // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
	Value_Type  string // 输出类型
	Config      string
}

// 点位-》查询数量
// 传递: driveid 设备id, page 页码, pageSize 每页数量
// 返回: Count 数量, err 错误
func Points_Config__Count(driveid uint, page uint, pageSize uint) (Count uint, err error) {
	if driveid == 0 {
		err = fmt.Errorf("ERROR driveid传递参数错误")
		return
	}

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
		// log.Printf("查询驱动配置无数据，驱动类型：%s, 分页%d/%d", driveType, page, pageSize)
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败，错误：%v, SQL:%s, 参数:%v", err, baseQuery, args)
		return
	}
	return
}

// 点位-》查询配置
// 传递: driveid 设备id, page 页码, pageSize 每页数量
// 返回: configs 配置, err 错误
func Points_Config__Query(driveid uint, page uint, pageSize uint) (configs []Points_Config_type, err error) {
	if driveid == 0 {
		err = fmt.Errorf("ERROR 配置driveid(Id)不能为空")
		return
	}

	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT 
			Points_Config.Id,
			Points_Config.Drive_Id,
			Drive_Config.Type,
			Points_Config.Tag,
			Points_Config.Description,
			Points_Config.Config,
			Points_Config.RW_Cancel,
			Points_Config.Value_Type
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
	rows, err := DB.Query(baseQuery, args...)

	// 区分无数据和查询错误，日志补充上下文便于排查
	if err == sql.ErrNoRows {
		// log.Printf("查询驱动配置无数据，驱动类型：%s, 分页%d/%d", driveType, page, pageSize)
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败, 错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}

	if err == sql.ErrNoRows {
		return
	} else if err != nil {
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
		var Config Points_Config_type
		err = rows.Scan(
			&Config.Id,
			&Config.Drive_Id,
			&Config.Drive_Type,
			&Config.Tag,
			&Config.Description,
			&Config.Config,
			&Config.RW_Cancel,
			&Config.Value_Type,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		configs = append(configs, Config)
	}
	return
}

// 点位-》增加配置
// 传递: config 配置数组形式
// 返回: err 错误
func Points_Config__Add(configs ...Points_Config_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		err = fmt.Errorf("批量新增失败：待新增配置列表为空")
		return
	}

	// 2. 遍历校验每个配置的参数合法性
	for i, cfg := range configs {
		if cfg.Id != 0 || cfg.Drive_Id != 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置参数错误 Id=%d/Drive_Id=%d, 必须为0 ",
				i, cfg.Id, cfg.Drive_Id)
			return
		}
		// 可选：校验必填字段（Type/Name/Config非空，根据业务需求加）
		if cfg.Tag == "" || cfg.Config == "" || cfg.RW_Cancel == "" || cfg.Value_Type == "" {
			err = fmt.Errorf("批量新增失败：第%d条配置Tag/Config/RW_Cancel/Value_Type不能为空", i)
			return
		}
	}

	// 3. 拼接批量INSERT的SQL和参数
	// baseQuery := "INSERT INTO `Drive_Config`(`Type`, `Name`, `Config`) VALUES "
	baseQuery := `
		INSERT
			INTO
				Points_Config
			(
				Drive_Id,
				Tag,
				Description,
				Config,
				RW_Cancel,
				Value_Type
			)
		VALUES
	`

	var args []interface{}         // 存储所有参数
	var valuePlaceholders []string // 存储每个值组的占位符 (?, ?, ?)

	// 遍历配置列表，拼接占位符和参数
	for _, cfg := range configs {
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?)")
		args = append(
			args,
			cfg.Drive_Id,
			cfg.Tag,
			sql.NullString{
				String: cfg.Description,
				Valid:  cfg.Description != "",
			},
			cfg.Config,
			cfg.RW_Cancel,
			cfg.Value_Type)
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

// 点位-》修改配置
// 传递: config 配置
// 返回: conid 获取自增的Id, err 错误
func Points_Config__Update(configs ...Points_Config_type) (err error) {
	// 1. 空列表校验
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	// 2. 遍历逐个更新
	for i, cfg := range configs {
		// 2.1 单条配置参数校验
		if cfg.Drive_Id == 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置参数错误 Drive_Id=%d, 必须为0 ",
				i, cfg.Drive_Id)
			return
		}

		// 可选：校验必填字段（Type/Name/Config非空，根据业务需求加）
		if cfg.Tag == "" || cfg.Config == "" || cfg.RW_Cancel == "" || cfg.Value_Type == "" {
			err = fmt.Errorf("批量新增失败：第%d条配置Tag/Config/RW_Cancel/Value_Type不能为空", i)
			return
		}

		// 2.2 动态拼接SET子句
		var setClauses []string
		var args []interface{}

		if cfg.Tag != "" {
			setClauses = append(setClauses, "`Tag` = ?")
			args = append(args, cfg.Tag)
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
			if !json.Valid([]byte(cfg.Config)) {
				err = fmt.Errorf("ERROR 第%d条配置非法JSON:%s", i, cfg.Config)
				return
			}
			setClauses = append(setClauses, "`Config` = ?")
			args = append(args, cfg.Config)
		}

		// 2.3 校验更新字段i
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段 Tag/Config/RW_Cancel/Value_TypeS至少传一个", i)
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
// 传递: ids 删除的id数组
// 返回: err 错误
func Points_Config__Del(ids ...uint) (err error) {
	// 1. 遍历逐个
	for idx, id := range ids {
		// 1.1 单条配置参数校验
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx)
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
	return
}
