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
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	// tag 格式正则：支持两种格式
	// 1. //设备id//标识符（包含设备id）
	// 2. /标识符（不包含设备id）
	// - 设备id不能为空，不含/
	// - 标识符不能为空，中间可有/，但不能以/结尾，不能有//
	Regex_Tag_Format = `^(//[^/]+//[^/]+(/[^/]+)*|/[^/]+(/[^/]+)*)$`

	// Config 格式正则：xx;xx;xx 格式
	// - 前后不能有;
	// - 中间可以有多个;，甚至可以连续;;
	Regex_Config_Format = `^[^;]+(;+[^;]+)*$`

	// Conversion 格式正则：空字符串、"0" 或一元一次方程表达式
	// - 空字符串或 "0" 表示无换算
	// - 非空非"0"时必须包含变量 x，只允许数字、运算符(+,-,*,/)、括号、小数点、空白
	// - 示例：x*2+10、(x-32)*5/9、x/100.0*3600
	Regex_Conversion_Format = `^(|0|[0-9\s\+\-\*\/\.\(\)]*x[0-9x\s\+\-\*\/\.\(\)]*)$`
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
type Collector_Config_Add_type struct {
	Label   string // 标识
	Uuid    string // Uuid
	Name    string // 设备名称
	User_Id uint   // 用户id
}

type Collector_Config_Update_type struct {
	Id   uint   // 采集 Id
	Name string // 设备名称
}

// 采集配置结构体
type Collector_Config_type struct {
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

type Drive_Config_Query_type struct {
	Id            uint      // 递增id
	Type          string    // 驱动类型
	Name          string    // 驱动名称
	Config        string    // 配置参数
	Points_Length uint      // 点位数量
	Creation_Time time.Time // 创建时间
}

type Drive_Config_type struct {
	Id            uint      // 递增id
	Type          string    // 驱动类型
	Name          string    // 驱动名称
	Config        string    //
	Points_Length uint      // 点位数量
	Creation_Time time.Time // 创建时间
}

// Drive_Config_Add_type 驱动配置增加结构体
type Drive_Config_Add_type struct {
	Id     uint   // 递增id
	Type   string // 驱动类型
	Name   string // 驱动名称
	Config string //
}

// Drive_Config_Update_type 驱动配置更新结构体
type Drive_Config_Update_type struct {
	Id     uint   // 驱动id
	Name   string // 驱动名称
	Config string //
}

// 驱动-》查询配置（回调）
func Drive_Config__Query_Callback(page uint, pageSize uint, callback func(Drive_Config_Query_type)) (err error) {
	baseQuery := `
		SELECT Id, Type, Name, Config, Points_Length, Creation_Time
		FROM Drive_Config
	`
	var args []interface{}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询驱动配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cfg Drive_Config_Query_type
		err = rows.Scan(&cfg.Id, &cfg.Type, &cfg.Name, &cfg.Config, &cfg.Points_Length, &cfg.Creation_Time)
		if err != nil {
			log.Print(err.Error())
			return
		}
		callback(cfg)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历驱动配置结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

// 驱动-》查询配置
func Drive_Config__Query(page uint, pageSize uint) (configs []Drive_Config_Query_type, err error) {
	err = Drive_Config__Query_Callback(page, pageSize, func(cfg Drive_Config_Query_type) {
		configs = append(configs, cfg)
	})
	return
}

// 驱动-》查询数量
func Drive_Config__Count(page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `Drive_Config`"
	var args []interface{}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[Drive_Config__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return
}

// 驱动-》增加配置
func Drive_Config__Add(configs ...Drive_Config_Add_type) (err error) {
	if len(configs) == 0 {
		return
	}
	baseQuery := "INSERT INTO `Drive_Config`(`Id`, `Type`, `Name`, `Config`, `Points_Length`, `Creation_Time`) VALUES "
	var args []interface{}
	var valuePlaceholders []string
	createTime := time.Now()
	configRegex := regexp.MustCompile(Regex_Config_Format)

	for i, cfg := range configs {
		if cfg.Type == "" {
			return fmt.Errorf("批量新增失败：第%d条配置Type不能为空", i+1)
		}
		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：第%d条配置Config不能为空", i+1)
		}
		if !configRegex.MatchString(cfg.Config) {
			return fmt.Errorf("批量新增失败：第%d条配置Config格式不正确", i+1)
		}
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?)")
		args = append(args, cfg.Id, cfg.Type, cfg.Name, cfg.Config, 0, createTime)
	}

	query := baseQuery + strings.Join(valuePlaceholders, ", ")
	_, err = DB.Exec(query, args...)
	if err != nil {
		err = fmt.Errorf("批量新增驱动配置失败: %w", err)
	}
	return
}

// 驱动-》修改配置
func Drive_Config__Update(configs ...Drive_Config_Update_type) (err error) {
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	configRegex := regexp.MustCompile(Regex_Config_Format)

	for idx, config := range configs {
		if config.Id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
		var setClauses []string
		var args []interface{}

		if config.Name != "" {
			setClauses = append(setClauses, "`Name` = ?")
			args = append(args, config.Name)
		}
		if config.Config != "" {
			if !configRegex.MatchString(config.Config) {
				err = fmt.Errorf("ERROR 第%d条配置Config格式不正确", idx+1)
				return
			}
			setClauses = append(setClauses, "`Config` = ?")
			args = append(args, config.Config)
		}

		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段", idx+1)
			return
		}

		query := fmt.Sprintf("UPDATE `Drive_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		args = append(args, config.Id)
		result, errExec := DB.Exec(query, args...)
		if errExec != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v", idx+1, config.Id, errExec)
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("WARNING 第%d条配置更新无生效行, ID:%d", idx+1, config.Id)
		}
	}
	return
}

// 驱动-》删除配置
func Drive_Config__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	for idx, id := range ids {
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
		query := "DELETE FROM `Drive_Config` WHERE `Id` = ?"
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置删除失败, ID:%d, 错误:%v", idx+1, id, err)
			return
		}
	}
	return
}

// 驱动-》同步配置（根据Id比对：新增/更新/删除）
func Drive_Config__Sync(configs []Drive_Config_Query_type) (err error) {
	// 1. 读取 MySQL 全部配置
	existing, err := Drive_Config__Query(0, 0)
	if err != nil {
		return fmt.Errorf("同步驱动配置：读取现有配置失败: %w", err)
	}

	// 2. 构建现有配置映射
	existingMap := make(map[uint]Drive_Config_Query_type, len(existing))
	for _, cfg := range existing {
		existingMap[cfg.Id] = cfg
	}

	// 3. 遍历传入配置，分为新增和更新
	incomingIds := make(map[uint]bool, len(configs))
	var toAdd []Drive_Config_Add_type
	var toUpdate []Drive_Config_Update_type

	for _, cfg := range configs {
		incomingIds[cfg.Id] = true
		if _, ok := existingMap[cfg.Id]; !ok {
			toAdd = append(toAdd, Drive_Config_Add_type{
				Id:     cfg.Id,
				Type:   cfg.Type,
				Config: cfg.Config,
			})
		} else {
			toUpdate = append(toUpdate, Drive_Config_Update_type{
				Id:     cfg.Id,
				Config: cfg.Config,
			})
		}
	}

	// 4. 找出现有但传入中没有的，需要删除
	var toDel []uint
	for _, cfg := range existing {
		if !incomingIds[cfg.Id] {
			toDel = append(toDel, cfg.Id)
		}
	}

	// 5. 执行增删改
	if len(toAdd) > 0 {
		if err = Drive_Config__Add(toAdd...); err != nil {
			return fmt.Errorf("同步驱动配置：新增失败: %w", err)
		}
		log.Printf("同步驱动配置：新增 %d 条", len(toAdd))
	}
	if len(toUpdate) > 0 {
		if err = Drive_Config__Update(toUpdate...); err != nil {
			return fmt.Errorf("同步驱动配置：更新失败: %w", err)
		}
		log.Printf("同步驱动配置：更新 %d 条", len(toUpdate))
	}
	if len(toDel) > 0 {
		if err = Drive_Config__Del(toDel...); err != nil {
			return fmt.Errorf("同步驱动配置：删除失败: %w", err)
		}
		log.Printf("同步驱动配置：删除 %d 条", len(toDel))
	}

	return nil
}

// Drive_Config__Search 搜索驱动配置
// 功能：按 Name 模糊搜索 + Type 精确过滤，返回匹配结果
// 传入：name — 驱动名称（LIKE 匹配，调用方需自带 % 通配符，空串不过滤）；types — 驱动类型列表（IN 精确匹配，空切片不过滤）；quantity — 返回数量上限（最大 30）
// 返回：匹配的驱动配置列表，错误
func Drive_Config__Search(name string, types []string, quantity uint) (configs []Drive_Config_Query_type, err error) {
	if quantity > 30 {
		quantity = 30
	}

	baseQuery := `
		SELECT Id, Type, Name, Config, Points_Length, Creation_Time
		FROM Drive_Config
	`
	var whereConditions []string
	var args []interface{}

	if name != "" {
		whereConditions = append(whereConditions, "`Name` LIKE ?")
		args = append(args, name)
	}
	if len(types) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(types)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Type` IN (%s)", placeholders))
		for _, t := range types {
			args = append(args, t)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	baseQuery += " LIMIT ?"
	args = append(args, quantity)

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 搜索驱动配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cfg Drive_Config_Query_type
		err = rows.Scan(&cfg.Id, &cfg.Type, &cfg.Name, &cfg.Config, &cfg.Points_Length, &cfg.Creation_Time)
		if err != nil {
			log.Print(err.Error())
			return
		}
		configs = append(configs, cfg)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历驱动配置结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

// 驱动-》更新点位数量
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
		quantity, err = Point_Config__Count([]uint{id}, 0, 0)
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
***************点位配置查询***************
 */
/*
***************点位配置结构体***************
 */
type Point_Config_Query_type struct {
	Id            uint      // 点位id
	Drive_Id      uint      // 驱动id
	Tag           string    // 点位标签
	Description   string    // 说明
	Config        string    // 配置信息
	Conversion    string    // 换算
	RW_Cancel     int       // 读写方式
	Value_Type    int       // 输出类型
	Creation_Time time.Time // 创建时间
}

type Point_Config_type struct {
	Id            uint      // 点位id
	Drive_Id      uint      // 驱动id
	Tag           string    // 点位标签
	Description   string    // 说明
	Config        string    // 配置信息
	Conversion    string    // 换算
	RW_Cancel     int       // 读写方式读写方式 1：禁止； 2：只读； 3：只写； 4：读写；
	Value_Type    int       // 输出类型
	Creation_Time time.Time // 创建时间
}

// 点位配置增加结构体
type Point_Config_Add_type struct {
	Id          uint   // 点位id
	Drive_Id    uint   // 驱动id
	Tag         string // 点位标签
	Description string // 说明
	Config      string // 配置信息
	Conversion  string // 换算
	RW_Cancel   int    // 读写方式
	Value_Type  int    // 输出类型
}

// 点位配置更新结构体
type Point_Config_Update_type struct {
	Id          uint   // 点位id
	Drive_Id    uint   // 驱动id
	Tag         string // 点位标签
	Description string // 说明
	Config      string // 配置信息
	Conversion  string // 换算
	RW_Cancel   int    // 读写方式
	Value_Type  int    // 输出类型
}

var (
	RW_Cancel_map = map[int]string{
		1: "N",
		2: "R",
		3: "W",
		4: "R/W",
	}
)

// 点位-》查询配置（回调）
// 传递：driveid 设备id，page 页码，pageSize 每页数量，callback 回调函数
func Point_Config__Query_Callback(driveid []uint, page uint, pageSize uint, callback func(Point_Config_Query_type)) (err error) {
	baseQuery := `
		SELECT Id, Drive_Id, Tag, Description, Config, Conversion, RW_Cancel, Value_Type, Creation_Time
		FROM Point_Config
	`
	var whereConditions []string
	var args []interface{}

	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Drive_Id` IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cfg Point_Config_Query_type
		var desc sql.NullString
		err = rows.Scan(&cfg.Id, &cfg.Drive_Id, &cfg.Tag, &desc, &cfg.Config, &cfg.Conversion, &cfg.RW_Cancel, &cfg.Value_Type, &cfg.Creation_Time)
		if err != nil {
			log.Print(err.Error())
			return
		}
		cfg.Description = desc.String // NULL 转为空字符串
		callback(cfg)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历点位配置结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

// 点位-》查询配置
// 传递：driveid 设备id, page 页码，pageSize 每页数量
func Point_Config__Query(driveid []uint, page uint, pageSize uint) (configs []Point_Config_Query_type, err error) {
	err = Point_Config__Query_Callback(driveid, page, pageSize, func(cfg Point_Config_Query_type) {
		configs = append(configs, cfg)
	})
	return
}

// 点位-》查询数量
func Point_Config__Count(driveid []uint, page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `Point_Config`"
	var whereConditions []string
	var args []interface{}

	if len(driveid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Drive_Id` IN (%s)", placeholders))
		for _, id := range driveid {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[Point_Config__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return
}

// 点位-》增加配置
func Point_Config__Add(configs ...Point_Config_Add_type) (err error) {
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}
	baseQuery := `INSERT INTO Point_Config (Id, Drive_Id, Tag, Description, Config, Conversion, RW_Cancel, Value_Type, Creation_Time) VALUES `
	var args []interface{}
	var valuePlaceholders []string
	t := time.Now()
	tagRegex := regexp.MustCompile(Regex_Tag_Format)
	configRegex := regexp.MustCompile(Regex_Config_Format)
	conversionRegex := regexp.MustCompile(Regex_Conversion_Format)

	for i, cfg := range configs {
		if cfg.Drive_Id == 0 {
			return fmt.Errorf("批量新增失败：第%d条数据 Drive_Id 等于0", i)
		}
		if cfg.Tag == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Tag 不能为空", i)
		}
		if !tagRegex.MatchString(cfg.Tag) {
			return fmt.Errorf("批量新增失败：第%d条数据 Tag 格式错误，必须符合 //设备id//标识符 格式", i)
		}
		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Config 不能为空", i)
		}
		if !configRegex.MatchString(cfg.Config) {
			return fmt.Errorf("批量新增失败：第%d条数据 Config 格式错误，必须符合 xx;xx;xx 格式", i)
		}
		// Conversion 可选，但非空时必须符合一元一次方程格式
		if cfg.Conversion != "" && cfg.Conversion != "0" && !conversionRegex.MatchString(cfg.Conversion) {
			return fmt.Errorf("批量新增失败：第%d条数据 Conversion 格式错误，必须是一元一次方程（如 x*2+10）", i)
		}
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			cfg.Id,
			cfg.Drive_Id,
			cfg.Tag,
			sql.NullString{String: cfg.Description, Valid: cfg.Description != "" && cfg.Description != "null"},
			cfg.Config,
			cfg.Conversion,
			cfg.RW_Cancel,
			cfg.Value_Type,
			t,
		)
	}

	query := baseQuery + strings.Join(valuePlaceholders, ", ")
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 Point_Config 失败: %w", err)
	}
	return nil
}

// 点位-》修改配置
func Point_Config__Update(configs ...Point_Config_Update_type) (err error) {
	if len(configs) == 0 {
		return
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

	var sqlPieces []string
	var allArgs []interface{}

	tagRegex := regexp.MustCompile(Regex_Tag_Format)
	configRegex := regexp.MustCompile(Regex_Config_Format)
	conversionRegex := regexp.MustCompile(Regex_Conversion_Format)

	for i, cfg := range configs {
		if cfg.Id == 0 {
			err = fmt.Errorf("批量更新失败：第%d条配置Id不能为空", i)
			return
		}
		var setClauses []string

		if cfg.Drive_Id != 0 {
			setClauses = append(setClauses, "`Drive_Id`=?")
			allArgs = append(allArgs, cfg.Drive_Id)
		}
		if cfg.Tag != "" {
			if !tagRegex.MatchString(cfg.Tag) {
				err = fmt.Errorf("批量更新失败：第%d条配置Tag格式错误，必须符合 //设备id//标识符 格式", i)
				return
			}
			setClauses = append(setClauses, "`Tag`=?")
			allArgs = append(allArgs, cfg.Tag)
		}
		if cfg.Description != "" {
			setClauses = append(setClauses, "`Description` = ?")
			allArgs = append(allArgs, sql.NullString{String: cfg.Description, Valid: cfg.Description != "null"})
		}
		if cfg.Config != "" {
			if !configRegex.MatchString(cfg.Config) {
				err = fmt.Errorf("批量更新失败：第%d条配置Config格式错误，必须符合 xx;xx;xx 格式", i)
				return
			}
			setClauses = append(setClauses, "`Config`=?")
			allArgs = append(allArgs, cfg.Config)
		}
		if cfg.Conversion != "" {
			// Conversion 非空时必须符合一元一次方程格式（"0" 表示无换算）
			if cfg.Conversion != "0" && !conversionRegex.MatchString(cfg.Conversion) {
				err = fmt.Errorf("批量更新失败：第%d条配置Conversion格式错误，必须是一元一次方程（如 x*2+10）", i)
				return
			}
			setClauses = append(setClauses, "`Conversion`=?")
			allArgs = append(allArgs, cfg.Conversion)
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
			err = fmt.Errorf("第%d条配置未指定任何更新字段", i)
			return
		}

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
func Point_Config__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	for idx, id := range ids {
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
		query := "DELETE FROM `Point_Config` WHERE `Id` = ?"
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条点位删除失败, ID:%d, 错误:%v", idx+1, id, err)
			return
		}
	}
	return
}

// 点位-》同步配置（根据Id比对：新增/更新/删除）
func Point_Config__Sync(configs []Point_Config_Query_type) (err error) {
	// 1. 读取 MySQL 全部配置
	existing, err := Point_Config__Query(nil, 0, 0)
	if err != nil {
		return fmt.Errorf("同步点位配置：读取现有配置失败: %w", err)
	}

	// 2. 构建现有配置映射
	existingMap := make(map[uint]Point_Config_Query_type, len(existing))
	for _, cfg := range existing {
		existingMap[cfg.Id] = cfg
	}

	// 3. 遍历传入配置，分为新增和更新
	incomingIds := make(map[uint]bool, len(configs))
	var toAdd []Point_Config_Add_type
	var toUpdate []Point_Config_Update_type

	for _, cfg := range configs {
		incomingIds[cfg.Id] = true
		if ex, ok := existingMap[cfg.Id]; !ok {
			toAdd = append(toAdd, Point_Config_Add_type{
				Id:         cfg.Id,
				Drive_Id:   cfg.Drive_Id,
				Config:     cfg.Config,
				Conversion: cfg.Conversion,
				RW_Cancel:  cfg.RW_Cancel,
				Value_Type: cfg.Value_Type,
			})
		} else {
			// 比对是否有变化
			if cfg.Drive_Id == ex.Drive_Id && cfg.Config == ex.Config && cfg.Conversion == ex.Conversion && cfg.RW_Cancel == ex.RW_Cancel && cfg.Value_Type == ex.Value_Type {
				continue
			}
			toUpdate = append(toUpdate, Point_Config_Update_type{
				Id:         cfg.Id,
				Drive_Id:   cfg.Drive_Id,
				Config:     cfg.Config,
				Conversion: cfg.Conversion,
				RW_Cancel:  cfg.RW_Cancel,
				Value_Type: cfg.Value_Type,
			})
		}
	}

	// 4. 找出现有但传入中没有的，需要删除
	var toDel []uint
	for _, cfg := range existing {
		if !incomingIds[cfg.Id] {
			toDel = append(toDel, cfg.Id)
		}
	}

	// 5. 执行增删改
	if len(toAdd) > 0 {
		if err = Point_Config__Add(toAdd...); err != nil {
			return fmt.Errorf("同步点位配置：新增失败: %w", err)
		}
		log.Printf("同步点位配置：新增 %d 条", len(toAdd))
	}
	if len(toUpdate) > 0 {
		if err = Point_Config__Update(toUpdate...); err != nil {
			return fmt.Errorf("同步点位配置：更新失败: %w", err)
		}
		log.Printf("同步点位配置：更新 %d 条", len(toUpdate))
	}
	if len(toDel) > 0 {
		if err = Point_Config__Del(toDel...); err != nil {
			return fmt.Errorf("同步点位配置：删除失败: %w", err)
		}
		log.Printf("同步点位配置：删除 %d 条", len(toDel))
	}

	return nil
}

// 点位-》查询设备id
// Point_Config__Query__DriveIds 批量查询：传入多个Id，返回 []{Id,Drive_Id}
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

// Point_Config__Search 搜索点位配置
// 功能：按 Tag 模糊搜索 + Drive_Id 精确过滤，返回匹配结果
// 传入：tag — 点位标签（LIKE 匹配，调用方需自带 % 通配符，空串不过滤）；driveIds — 驱动id列表（IN 精确匹配，空切片不过滤）；quantity — 返回数量上限（最大 30）
// 返回：匹配的点位配置列表，错误
func Point_Config__Search(tag string, driveIds []uint, quantity uint) (configs []Point_Config_Query_type, err error) {
	if quantity > 30 {
		quantity = 30
	}

	baseQuery := `
		SELECT Id, Drive_Id, Tag, Description, Config, Conversion, RW_Cancel, Value_Type, Creation_Time
		FROM Point_Config
	`
	var whereConditions []string
	var args []interface{}

	if tag != "" {
		whereConditions = append(whereConditions, "`Tag` LIKE ?")
		args = append(args, tag)
	}
	if len(driveIds) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(driveIds)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Drive_Id` IN (%s)", placeholders))
		for _, id := range driveIds {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	baseQuery += " LIMIT ?"
	args = append(args, quantity)

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 搜索点位配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cfg Point_Config_Query_type
		var desc sql.NullString
		err = rows.Scan(&cfg.Id, &cfg.Drive_Id, &cfg.Tag, &desc, &cfg.Config, &cfg.Conversion, &cfg.RW_Cancel, &cfg.Value_Type, &cfg.Creation_Time)
		if err != nil {
			log.Print(err.Error())
			return
		}
		cfg.Description = desc.String
		configs = append(configs, cfg)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历点位配置结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

/*
***************报警配置查询***************
 */

type Alarm_Config_Query_type struct {
	Id            uint      // 报警id
	Point_Id      uint      // 点位id
	Tag           string    // 报警标识符
	Config        string    // 报警配置
	Group         int       // 报警组
	Creation_Time time.Time // 创建时间
}

type Alarm_Config_type struct {
	Id            uint      // 报警id
	Point_Id      uint      // 点位id
	Tag           string    // 报警标识符
	Config        string    // 报警配置
	Group         int       // 报警组
	Creation_Time time.Time // 创建时间
}

// 报警配置增加结构体
type Alarm_Config_Add_type struct {
	Id       uint   // 报警id
	Point_Id uint   // 点位id
	Tag      string // 报警标识符
	Config   string // 报警配置
	Group    int    // 报警组
}

// 报警配置更新结构体
type Alarm_Config_Update_type struct {
	Id       uint   // 报警id
	Point_Id uint   // 点位id
	Tag      string // 报警标识符
	Config   string // 报警配置
	Group    int    // 报警组
}

// 报警-》查询配置（回调）
func Alarm_Config__Query_Callback(pointid []uint, page uint, pageSize uint, callback func(Alarm_Config_Query_type)) (err error) {
	baseQuery := `
		SELECT Id, Point_Id, Tag, Config, ` + "`Group`" + `, Creation_Time
		FROM Alarm_Config
	`
	var whereConditions []string
	var args []interface{}

	if len(pointid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Id` IN (%s)", placeholders))
		for _, id := range pointid {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询报警配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cfg Alarm_Config_Query_type
		err = rows.Scan(&cfg.Id, &cfg.Point_Id, &cfg.Tag, &cfg.Config, &cfg.Group, &cfg.Creation_Time)
		if err != nil {
			log.Print(err.Error())
			return
		}
		callback(cfg)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历报警配置结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

// 报警-》查询配置
func Alarm_Config__Query(pointid []uint, page uint, pageSize uint) (configs []Alarm_Config_Query_type, err error) {
	err = Alarm_Config__Query_Callback(pointid, page, pageSize, func(cfg Alarm_Config_Query_type) {
		configs = append(configs, cfg)
	})
	return
}

// 报警-》查询数量
func Alarm_Config__Count(pointid []uint, page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `Alarm_Config`"
	var whereConditions []string
	var args []interface{}

	if len(pointid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Id` IN (%s)", placeholders))
		for _, id := range pointid {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[Alarm_Config__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return
}

// 报警-》增加配置
func Alarm_Config__Add(configs ...Alarm_Config_Add_type) (err error) {
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}
	baseQuery := `INSERT INTO Alarm_Config (Id, Point_Id, Tag, Config, ` + "`Group`" + `, Creation_Time) VALUES `
	var args []interface{}
	var valuePlaceholders []string
	t := time.Now()
	tagRegex := regexp.MustCompile(Regex_Tag_Format)

	for i, cfg := range configs {
		if cfg.Point_Id == 0 {
			return fmt.Errorf("批量新增失败：第%d条数据 Point_Id 等于0", i)
		}
		if cfg.Tag == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Tag 不能为空", i)
		}
		if !tagRegex.MatchString(cfg.Tag) {
			return fmt.Errorf("批量新增失败：第%d条数据 Tag 格式不正确", i)
		}
		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：第%d条数据 Config 不能为空", i)
		}
		if cfg.Group == 0 {
			return fmt.Errorf("批量新增失败：第%d条数据 Group 不能等于0", i)
		}
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?)")
		args = append(args, cfg.Id, cfg.Point_Id, cfg.Tag, cfg.Config, cfg.Group, t)
	}

	query := baseQuery + strings.Join(valuePlaceholders, ", ")
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 Alarm_Config 失败: %w", err)
	}
	return nil
}

// 报警-》修改配置
func Alarm_Config__Update(configs ...Alarm_Config_Update_type) (err error) {
	if len(configs) == 0 {
		return
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

	var sqlPieces []string
	var allArgs []interface{}
	tagRegex := regexp.MustCompile(Regex_Tag_Format)

	for i, cfg := range configs {
		if cfg.Id == 0 {
			err = fmt.Errorf("批量更新失败：第[%d]条配置Id不能为空", i)
			return
		}
		var setClauses []string

		if cfg.Point_Id != 0 {
			setClauses = append(setClauses, "`Point_Id`=?")
			allArgs = append(allArgs, cfg.Point_Id)
		}
		if cfg.Tag != "" {
			if !tagRegex.MatchString(cfg.Tag) {
				err = fmt.Errorf("批量更新失败：第[%d]条配置Tag格式不正确", i)
				return
			}
			setClauses = append(setClauses, "`Tag`=?")
			allArgs = append(allArgs, cfg.Tag)
		}
		if cfg.Config != "" {
			setClauses = append(setClauses, "`Config`=?")
			allArgs = append(allArgs, cfg.Config)
		}
		if cfg.Group != 0 {
			setClauses = append(setClauses, "`Group`=?")
			allArgs = append(allArgs, cfg.Group)
		}

		if len(setClauses) == 0 {
			err = fmt.Errorf("第[%d]条配置未指定任何更新字段", i)
			return
		}

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
	log.Printf("批量更新成功，共更新%d条报警配置", len(configs))
	return
}

// 报警-》删除配置
func Alarm_Config__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	for idx, id := range ids {
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
		query := "DELETE FROM `Alarm_Config` WHERE `Id` = ?"
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条报警删除失败, ID:%d, 错误:%v", idx+1, id, err)
			return
		}
	}
	return
}

// 报警-》同步配置（根据Id比对：新增/更新/删除）
func Alarm_Config__Sync(configs []Alarm_Config_Add_type) (err error) {
	// 1. 读取 MySQL 全部配置
	existing, err := Alarm_Config__Query(nil, 0, 0)
	if err != nil {
		return fmt.Errorf("同步报警配置：读取现有配置失败: %w", err)
	}

	// 2. 构建现有配置映射
	existingMap := make(map[uint]Alarm_Config_Query_type, len(existing))
	for _, cfg := range existing {
		existingMap[cfg.Id] = cfg
	}

	// 3. 遍历传入配置，分为新增和更新
	incomingIds := make(map[uint]bool, len(configs))
	var toAdd []Alarm_Config_Add_type
	var toUpdate []Alarm_Config_Update_type

	for _, cfg := range configs {
		incomingIds[cfg.Id] = true
		if ex, ok := existingMap[cfg.Id]; !ok {
			toAdd = append(toAdd, cfg)
		} else {
			// 比对是否有变化
			if cfg.Point_Id == ex.Point_Id && cfg.Config == ex.Config && cfg.Group == ex.Group {
				continue
			}
			toUpdate = append(toUpdate, Alarm_Config_Update_type{
				Id:       cfg.Id,
				Point_Id: cfg.Point_Id,
				Tag:      cfg.Tag,
				Config:   cfg.Config,
				Group:    cfg.Group,
			})
		}
	}

	// 4. 找出现有但传入中没有的，需要删除
	var toDel []uint
	for _, cfg := range existing {
		if !incomingIds[cfg.Id] {
			toDel = append(toDel, cfg.Id)
		}
	}

	// 5. 执行增删改
	if len(toAdd) > 0 {
		if err = Alarm_Config__Add(toAdd...); err != nil {
			return fmt.Errorf("同步报警配置：新增失败: %w", err)
		}
		log.Printf("同步报警配置：新增 %d 条", len(toAdd))
	}
	if len(toUpdate) > 0 {
		if err = Alarm_Config__Update(toUpdate...); err != nil {
			return fmt.Errorf("同步报警配置：更新失败: %w", err)
		}
		log.Printf("同步报警配置：更新 %d 条", len(toUpdate))
	}
	if len(toDel) > 0 {
		if err = Alarm_Config__Del(toDel...); err != nil {
			return fmt.Errorf("同步报警配置：删除失败: %w", err)
		}
		log.Printf("同步报警配置：删除 %d 条", len(toDel))
	}

	return nil
}

// Alarm_Config__Search 搜索报警配置
// 功能：按 Tag 模糊搜索 + Point_Id 精确过滤 + Group 精确过滤，返回匹配结果
// 传入：tag — 报警标签（LIKE 匹配，调用方需自带 % 通配符，空串不过滤）；pointIds — 点位id列表（IN 精确匹配，空切片不过滤）；groups — 报警组列表（IN 精确匹配，空切片不过滤）；quantity — 返回数量上限（最大 30）
// 返回：匹配的报警配置列表，错误
func Alarm_Config__Search(tag string, pointIds []uint, groups []int, quantity uint) (configs []Alarm_Config_Query_type, err error) {
	if quantity > 30 {
		quantity = 30
	}

	baseQuery := `
		SELECT Id, Point_Id, Tag, Config, ` + "`Group`" + `, Creation_Time
		FROM Alarm_Config
	`
	var whereConditions []string
	var args []interface{}

	if tag != "" {
		whereConditions = append(whereConditions, "`Tag` LIKE ?")
		args = append(args, tag)
	}
	if len(pointIds) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointIds)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Id` IN (%s)", placeholders))
		for _, id := range pointIds {
			args = append(args, id)
		}
	}
	if len(groups) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(groups)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Group` IN (%s)", placeholders))
		for _, g := range groups {
			args = append(args, g)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	baseQuery += " LIMIT ?"
	args = append(args, quantity)

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 搜索报警配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cfg Alarm_Config_Query_type
		err = rows.Scan(&cfg.Id, &cfg.Point_Id, &cfg.Tag, &cfg.Config, &cfg.Group, &cfg.Creation_Time)
		if err != nil {
			log.Print(err.Error())
			return
		}
		configs = append(configs, cfg)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历报警配置结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

/*
***************报警数据查询***************
 */

type Alarm_Data_Query_type struct {
	Id       uint      // 报警数据id
	Alarm_Id uint      // 报警配置id
	Status   string    // 报警状态
	Time     time.Time // 报警时间
}

type Alarm_Data_type struct {
	Id       uint      // 报警数据id
	Alarm_Id uint      // 报警配置id
	Status   string    // 报警状态
	Time     time.Time // 报警时间
}

// 报警数据增加结构体
type Alarm_Data_Add_type struct {
	Id       uint      // 报警数据id（0=自增）
	Alarm_Id uint      // 报警配置id
	Status   string    // 报警状态
	Time     time.Time // 报警时间
}

// 报警数据更新结构体
type Alarm_Data_Update_type struct {
	Id       uint   // 报警数据id
	Alarm_Id uint   // 报警配置id
	Status   string // 报警状态
}

// 报警数据-》查询（回调）
func Alarm_Data__Query_Callback(alarmIds []uint, page uint, pageSize uint, callback func(Alarm_Data_Query_type)) (err error) {
	baseQuery := `
		SELECT Id, Alarm_Id, Status, ` + "`Time`" + `
		FROM Alarm_Data
	`
	var whereConditions []string
	var args []interface{}

	if len(alarmIds) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(alarmIds)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Alarm_Id` IN (%s)", placeholders))
		for _, id := range alarmIds {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询报警数据失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var d Alarm_Data_Query_type
		err = rows.Scan(&d.Id, &d.Alarm_Id, &d.Status, &d.Time)
		if err != nil {
			log.Print(err.Error())
			return
		}
		callback(d)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历报警数据结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

// 报警数据-》查询
func Alarm_Data__Query(alarmIds []uint, page uint, pageSize uint) (data []Alarm_Data_Query_type, err error) {
	err = Alarm_Data__Query_Callback(alarmIds, page, pageSize, func(d Alarm_Data_Query_type) {
		data = append(data, d)
	})
	return
}

// 报警数据-》查询数量
func Alarm_Data__Count(alarmIds []uint, page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `Alarm_Data`"
	var whereConditions []string
	var args []interface{}

	if len(alarmIds) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(alarmIds)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Alarm_Id` IN (%s)", placeholders))
		for _, id := range alarmIds {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[Alarm_Data__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return
}

// 报警数据-》增加
func Alarm_Data__Add(data ...Alarm_Data_Add_type) (err error) {
	if len(data) == 0 {
		return fmt.Errorf("批量新增失败：待新增数据列表为空")
	}
	baseQuery := `INSERT INTO Alarm_Data (Id, Alarm_Id, Status, ` + "`Time`" + `) VALUES `
	var args []interface{}
	var valuePlaceholders []string

	for i, d := range data {
		if d.Alarm_Id == 0 {
			return fmt.Errorf("批量新增失败：第[%d]条数据 Alarm_Id 等于0", i)
		}
		if d.Status == "" {
			return fmt.Errorf("批量新增失败：第[%d]条数据 Status 不能为空", i)
		}
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?)")
		args = append(args, d.Id, d.Alarm_Id, d.Status, d.Time)
	}

	query := baseQuery + strings.Join(valuePlaceholders, ", ")
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 Alarm_Data 失败: %w", err)
	}
	return nil
}

// 报警数据-》修改
func Alarm_Data__Update(data ...Alarm_Data_Update_type) (err error) {
	if len(data) == 0 {
		return
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

	var sqlPieces []string
	var allArgs []interface{}

	for i, d := range data {
		if d.Id == 0 {
			err = fmt.Errorf("批量更新失败：第[%d]条配置Id不能为空", i)
			return
		}
		var setClauses []string

		if d.Alarm_Id != 0 {
			setClauses = append(setClauses, "`Alarm_Id`=?")
			allArgs = append(allArgs, d.Alarm_Id)
		}
		if d.Status != "" {
			setClauses = append(setClauses, "`Status`=?")
			allArgs = append(allArgs, d.Status)
		}

		if len(setClauses) == 0 {
			err = fmt.Errorf("第[%d]条配置未指定任何更新字段", i)
			return
		}

		sql := fmt.Sprintf("UPDATE `Alarm_Data` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		allArgs = append(allArgs, d.Id)
		sqlPieces = append(sqlPieces, sql)
	}

	fullSql := strings.Join(sqlPieces, ";")
	_, err = tx.Exec(fullSql, allArgs...)
	if err != nil {
		err = fmt.Errorf("批量更新执行失败, SQL:%s, args:%v, err:%w", fullSql, allArgs, err)
		return err
	}
	log.Printf("批量更新成功，共更新%d条报警数据", len(data))
	return
}

// 报警数据-》删除
func Alarm_Data__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	for idx, id := range ids {
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
		query := "DELETE FROM `Alarm_Data` WHERE `Id` = ?"
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条报警数据删除失败, ID:%d, 错误:%v", idx+1, id, err)
			return
		}
	}
	return
}

// 报警数据-》同步（根据Id比对：新增/更新/删除）
func Alarm_Data__Sync(data []Alarm_Data_Add_type) (err error) {
	// 1. 读取 MySQL 全部数据
	existing, err := Alarm_Data__Query(nil, 0, 0)
	if err != nil {
		return fmt.Errorf("同步报警数据：读取现有数据失败: %w", err)
	}

	// 2. 构建现有数据映射
	existingMap := make(map[uint]Alarm_Data_Query_type, len(existing))
	for _, d := range existing {
		existingMap[d.Id] = d
	}

	// 3. 遍历传入数据，分为新增和更新
	incomingIds := make(map[uint]bool, len(data))
	var toAdd []Alarm_Data_Add_type
	var toUpdate []Alarm_Data_Update_type

	for _, d := range data {
		incomingIds[d.Id] = true
		if ex, ok := existingMap[d.Id]; !ok {
			toAdd = append(toAdd, d)
		} else {
			// 比对是否有变化
			if d.Alarm_Id == ex.Alarm_Id && d.Status == ex.Status {
				continue
			}
			toUpdate = append(toUpdate, Alarm_Data_Update_type{
				Id:       d.Id,
				Alarm_Id: d.Alarm_Id,
				Status:   d.Status,
			})
		}
	}

	// 4. 找出现有但传入中没有的，需要删除
	var toDel []uint
	for _, d := range existing {
		if !incomingIds[d.Id] {
			toDel = append(toDel, d.Id)
		}
	}

	// 5. 执行增删改
	if len(toAdd) > 0 {
		if err = Alarm_Data__Add(toAdd...); err != nil {
			return fmt.Errorf("同步报警数据：新增失败: %w", err)
		}
		log.Printf("同步报警数据：新增 %d 条", len(toAdd))
	}
	if len(toUpdate) > 0 {
		if err = Alarm_Data__Update(toUpdate...); err != nil {
			return fmt.Errorf("同步报警数据：更新失败: %w", err)
		}
		log.Printf("同步报警数据：更新 %d 条", len(toUpdate))
	}
	if len(toDel) > 0 {
		if err = Alarm_Data__Del(toDel...); err != nil {
			return fmt.Errorf("同步报警数据：删除失败: %w", err)
		}
		log.Printf("同步报警数据：删除 %d 条", len(toDel))
	}

	return nil
}

/*
***************历史配置查询***************
 */

type History_Config_Query_type struct {
	Id            uint      // 历史id
	Point_Id      uint      // 点位id
	Config        string    // 历史配置
	Creation_Time time.Time // 创建时间
}

type History_Config_type struct {
	Id            uint      // 历史id
	Point_Id      uint      // 点位id
	Config        string    // 历史配置
	Creation_Time time.Time // 创建时间
}

// 历史配置增加结构体
type History_Config_Add_type struct {
	Id       uint   // 历史id
	Point_Id uint   // 点位id
	Config   string // 历史配置
}

// 历史配置更新结构体
type History_Config_Update_type struct {
	Id       uint   // 历史id
	Point_Id uint   // 点位id
	Config   string // 历史配置
}

// 历史-》查询配置（回调）
func History_Config__Query_Callback(pointid []uint, page uint, pageSize uint, callback func(History_Config_Query_type)) (err error) {
	baseQuery := `
		SELECT Id, Point_Id, Config, Creation_Time
		FROM History_Config
	`
	var whereConditions []string
	var args []interface{}

	if len(pointid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Id` IN (%s)", placeholders))
		for _, id := range pointid {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询历史配置失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cfg History_Config_Query_type
		err = rows.Scan(&cfg.Id, &cfg.Point_Id, &cfg.Config, &cfg.Creation_Time)
		if err != nil {
			log.Print(err.Error())
			return
		}
		callback(cfg)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历历史配置结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

// 历史-》查询配置
func History_Config__Query(pointid []uint, page uint, pageSize uint) (configs []History_Config_Query_type, err error) {
	err = History_Config__Query_Callback(pointid, page, pageSize, func(cfg History_Config_Query_type) {
		configs = append(configs, cfg)
	})
	return
}

// 历史-》查询数量
func History_Config__Count(pointid []uint, page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `History_Config`"
	var whereConditions []string
	var args []interface{}

	if len(pointid) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pointid)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Point_Id` IN (%s)", placeholders))
		for _, id := range pointid {
			args = append(args, id)
		}
	}

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[History_Config__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return
}

// 历史-》增加配置
func History_Config__Add(configs ...History_Config_Add_type) (err error) {
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}
	baseQuery := `INSERT INTO History_Config (Id, Point_Id, Config, Creation_Time) VALUES `
	var args []interface{}
	var valuePlaceholders []string
	t := time.Now()

	for i, cfg := range configs {
		if cfg.Point_Id == 0 {
			return fmt.Errorf("批量新增失败：第[%d]条数据 Point_Id 等于0", i)
		}
		if cfg.Config == "" {
			return fmt.Errorf("批量新增失败：第[%d]条数据 Config 不能为空", i)
		}
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?)")
		args = append(args, cfg.Id, cfg.Point_Id, cfg.Config, t)
	}

	query := baseQuery + strings.Join(valuePlaceholders, ", ")
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 History_Config 失败: %w", err)
	}
	return nil
}

// 历史-》修改配置
func History_Config__Update(configs ...History_Config_Update_type) (err error) {
	if len(configs) == 0 {
		return
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

	var sqlPieces []string
	var allArgs []interface{}

	for i, cfg := range configs {
		if cfg.Id == 0 {
			err = fmt.Errorf("批量更新失败：第[%d]条配置Id不能为空", i)
			return
		}
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
			err = fmt.Errorf("第[%d]条配置未指定任何更新字段", i)
			return
		}

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
	log.Printf("批量更新成功，共更新%d条历史配置", len(configs))
	return
}

// 历史-》删除配置
func History_Config__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	for idx, id := range ids {
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
		query := "DELETE FROM `History_Config` WHERE `Id` = ?"
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条历史删除失败, ID:%d, 错误:%v", idx+1, id, err)
			return
		}
	}
	return
}

// 历史-》同步配置（根据Id比对：新增/更新/删除）
func History_Config__Sync(configs []History_Config_Add_type) (err error) {
	// 1. 读取 MySQL 全部配置
	existing, err := History_Config__Query(nil, 0, 0)
	if err != nil {
		return fmt.Errorf("同步历史配置：读取现有配置失败: %w", err)
	}

	// 2. 构建现有配置映射
	existingMap := make(map[uint]History_Config_Query_type, len(existing))
	for _, cfg := range existing {
		existingMap[cfg.Id] = cfg
	}

	// 3. 遍历传入配置，分为新增和更新
	incomingIds := make(map[uint]bool, len(configs))
	var toAdd []History_Config_Add_type
	var toUpdate []History_Config_Update_type

	for _, cfg := range configs {
		incomingIds[cfg.Id] = true
		if ex, ok := existingMap[cfg.Id]; !ok {
			toAdd = append(toAdd, cfg)
		} else {
			// 比对是否有变化
			if cfg.Point_Id == ex.Point_Id && cfg.Config == ex.Config {
				continue
			}
			toUpdate = append(toUpdate, History_Config_Update_type{
				Id:       cfg.Id,
				Point_Id: cfg.Point_Id,
				Config:   cfg.Config,
			})
		}
	}

	// 4. 找出现有但传入中没有的，需要删除
	var toDel []uint
	for _, cfg := range existing {
		if !incomingIds[cfg.Id] {
			toDel = append(toDel, cfg.Id)
		}
	}

	// 5. 执行增删改
	if len(toAdd) > 0 {
		if err = History_Config__Add(toAdd...); err != nil {
			return fmt.Errorf("同步历史配置：新增失败: %w", err)
		}
		log.Printf("同步历史配置：新增 %d 条", len(toAdd))
	}
	if len(toUpdate) > 0 {
		if err = History_Config__Update(toUpdate...); err != nil {
			return fmt.Errorf("同步历史配置：更新失败: %w", err)
		}
		log.Printf("同步历史配置：更新 %d 条", len(toUpdate))
	}
	if len(toDel) > 0 {
		if err = History_Config__Del(toDel...); err != nil {
			return fmt.Errorf("同步历史配置：删除失败: %w", err)
		}
		log.Printf("同步历史配置：删除 %d 条", len(toDel))
	}

	return nil
}

/*
***************APP_Config 采集配置信息***************
 */

// APP_Config 类型定义
type APP_Config_type struct {
	Id    uint   // 递增id
	Ket   string // 键名
	Value string // 键值
}

// APP_Config_Add_type 增加结构体
type APP_Config_Add_type struct {
	Ket   string // 键名
	Value string // 键值
}

// APP_Config_Update_type 更新结构体
type APP_Config_Update_type struct {
	Id    uint   // id
	Ket   string // 键名
	Value string // 键值
}

// APP_Config-》查询配置（回调）
func APP_Config__Query_Callback(page uint, pageSize uint, callback func(APP_Config_type)) (err error) {
	baseQuery := `
		SELECT Id, ` + "`Ket`" + `, ` + "`Value`" + `
		FROM APP_Config
	`
	var args []interface{}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询APP_Config失败，错误:%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cfg APP_Config_type
		err = rows.Scan(&cfg.Id, &cfg.Ket, &cfg.Value)
		if err != nil {
			log.Print(err.Error())
			return
		}
		callback(cfg)
	}

	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("ERROR 遍历APP_Config结果集失败，错误:%v", err)
		log.Print(err)
	}
	return
}

// APP_Config-》查询配置
func APP_Config__Query(page uint, pageSize uint) (configs []APP_Config_type, err error) {
	err = APP_Config__Query_Callback(page, pageSize, func(cfg APP_Config_type) {
		configs = append(configs, cfg)
	})
	return
}

// APP_Config-》查询数量
func APP_Config__Count(page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `APP_Config`"
	var args []interface{}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[APP_Config__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return
}

// APP_Config-》增加配置
func APP_Config__Add(configs ...APP_Config_Add_type) (err error) {
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}
	baseQuery := `INSERT INTO APP_Config (` + "`Ket`" + `, ` + "`Value`" + `) VALUES `
	var args []interface{}
	var valuePlaceholders []string

	for i, cfg := range configs {
		if cfg.Ket == "" {
			return fmt.Errorf("批量新增失败：第[%d]条数据 Ket 不能为空", i)
		}
		valuePlaceholders = append(valuePlaceholders, "(?, ?)")
		args = append(args, cfg.Ket, cfg.Value)
	}

	query := baseQuery + strings.Join(valuePlaceholders, ", ")
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 APP_Config 失败: %w", err)
	}
	return nil
}

// APP_Config-》修改配置
func APP_Config__Update(configs ...APP_Config_Update_type) (err error) {
	if len(configs) == 0 {
		return
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

	var sqlPieces []string
	var allArgs []interface{}

	for i, cfg := range configs {
		if cfg.Id == 0 {
			err = fmt.Errorf("批量更新失败：第[%d]条配置Id不能为空", i)
			return
		}
		var setClauses []string

		if cfg.Ket != "" {
			setClauses = append(setClauses, "`Ket`=?")
			allArgs = append(allArgs, cfg.Ket)
		}
		if cfg.Value != "" {
			setClauses = append(setClauses, "`Value`=?")
			allArgs = append(allArgs, cfg.Value)
		}

		if len(setClauses) == 0 {
			err = fmt.Errorf("第[%d]条配置未指定任何更新字段", i)
			return
		}

		sql := fmt.Sprintf("UPDATE `APP_Config` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		allArgs = append(allArgs, cfg.Id)
		sqlPieces = append(sqlPieces, sql)
	}

	fullSql := strings.Join(sqlPieces, ";")
	_, err = tx.Exec(fullSql, allArgs...)
	if err != nil {
		err = fmt.Errorf("批量更新执行失败, SQL:%s, args:%v, err:%w", fullSql, allArgs, err)
		return err
	}
	log.Printf("批量更新成功，共更新%d条APP_Config配置", len(configs))
	return
}

// APP_Config-》删除配置
func APP_Config__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return
	}
	for idx, id := range ids {
		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx+1)
			return
		}
		query := "DELETE FROM `APP_Config` WHERE `Id` = ?"
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条APP_Config删除失败, ID:%d, 错误:%v", idx+1, id, err)
			return
		}
	}
	return
}

// APP_Config__GetByKet 根据键名获取值
func APP_Config__GetByKet(ket string) (value string, err error) {
	query := "SELECT `Value` FROM `APP_Config` WHERE `Ket` = ? LIMIT 1"
	err = DB.QueryRow(query, ket).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		err = fmt.Errorf("[APP_Config__GetByKet] 查询失败 | Ket=%s | err=%w", ket, err)
		log.Print(err)
	}
	return
}

// APP_Config__SetByKet 根据键名设置值（不存在则新增，存在则更新）
func APP_Config__SetByKet(ket, value string) (err error) {
	if ket == "" {
		return fmt.Errorf("Ket 不能为空")
	}
	// 先查询是否存在
	existingValue, err := APP_Config__GetByKet(ket)
	if err != nil {
		return err
	}

	if existingValue == "" {
		// 不存在，新增
		return APP_Config__Add(APP_Config_Add_type{Ket: ket, Value: value})
	}
	// 存在，更新（需要先获取Id）
	query := "SELECT `Id` FROM `APP_Config` WHERE `Ket` = ? LIMIT 1"
	var id uint
	err = DB.QueryRow(query, ket).Scan(&id)
	if err != nil {
		return fmt.Errorf("查询APP_Config Id失败: %w", err)
	}
	return APP_Config__Update(APP_Config_Update_type{Id: id, Value: value})
}
