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
	Name          string    // 驱动名称
	Config        string    // json配置参数
	Type          string    // 驱动类型
	Creation_Time time.Time // 创建时间
	Collector_Id  uint      // 采集器标识
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
	Config      string // 配置信息
	History     string // 存储
	Alarm       string // 报警
	Alarm_Group int    // 报警组
}

// 点位配置更新结构体
type Points_Config_Update_type struct {
	Id          uint   // 点位id
	Tag         string // 点位标识
	Description string // 点位描述
	RW_Cancel   string // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
	Value_Type  string // 输出类型
	Config      string // 配置信息
	History     string // 存储
	Alarm       string // 报警
	Alarm_Group int    // 报警组
}

// 点位配置结构体
type Points_Config_type struct {
	Collector Collector__Carry_type
	Drive     Drive__Carry_type

	Points_Config_Update_type
	Creation_Time time.Time // 创建时间

}

/*
***************mqtt配置结构体***************
 */

// 结构体
type Mqtt__type struct {
	Id                 uint      // 点位id
	Type               string    // 类型 私有协议、繁易
	Example_IDentifier string    // mqtt实例标识符
	Topic_Push         string    // 主题
	Topic_Down         string    // 点下发值
	Topic_Alarm        string    // 点位报警
	Creation_Time      time.Time // 创建时间
	Creation_User      uint      // 创建的用户id
}

type Mqtt__Add_type struct {
	Type               string // 类型 私有协议、繁易
	Example_IDentifier string // mqtt实例标识符
	Topic_Push         string // 主题
	Topic_Down         string // 点下发值
	Topic_Alarm        string // 点位报警
	Creation_User      uint   // 创建的用户id
}

type Mqtt__Update_type struct {
	Id                 uint   // 点位id
	Type               string // 类型 私有协议、繁易
	Example_IDentifier string // mqtt实例标识符
	Topic_Push         string // 主题
	Topic_Down         string // 点下发值
	Topic_Alarm        string // 点位报警
	Creation_User      uint   // 创建的用户id
}

// Mqtt-》查询数量
// 传递：Types 设备类型，Example_IDentifiers 设备标识符，Topics 设备主题，page 页码，pageSize 每页数量，
// 返回：Count 数量，err 错误
func Mqtt__Count(Types []string, Example_IDentifiers []string, Topics []string, page uint, pageSize uint) (Count uint, err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			COUNT(Mqtt.Id) 
		FROM Mqtt 
	`
	var args []interface{} // 存储SQL参数，防止SQL注入
	var whereConditions []string

	if len(Types) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Types)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt`.`Type` IN (%s)", placeholders))
		for _, Type := range Types {
			args = append(args, Type)
		}
	}

	if len(Example_IDentifiers) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Example_IDentifiers)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt`.`Example_IDentifier` IN (%s)", placeholders))
		for _, Example_IDentifier := range Example_IDentifiers {
			args = append(args, Example_IDentifier)
		}
	}

	if len(Topics) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Topics)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt`.`Topic` IN (%s)", placeholders))
		for _, Topic := range Topics {
			args = append(args, Topic)
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
		log.Printf("查询点位配置无数据")
		Count = 0
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误：%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}

	return
}

// Mqtt-》查询配置（回调）
// 传递：Types 设备类型，Example_IDentifiers 设备标识符，Topics 设备主题，page 页码，pageSize 每页数量，callback 回调函数
// 返回：err 错误
func Mqtt__Query_Callback(Types []string, Example_IDentifiers []string, Topics []string, page uint, pageSize uint, callback func(Mqtt__type)) (err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Mqtt.Id,
			Mqtt.Type,
			Mqtt.Example_IDentifier,
			Mqtt.Topic_Push,
			Mqtt.Topic_Down,
			Mqtt.Topic_Alarm,
			Mqtt.Creation_Time,
			Mqtt.Creation_User
		FROM Mqtt
	`
	var whereConditions []string
	var args []interface{} // 存储SQL参数，防止SQL注入

	if len(Types) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Types)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt`.`Type` IN (%s)", placeholders))
		for _, Type := range Types {
			args = append(args, Type)
		}
	}

	if len(Example_IDentifiers) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Example_IDentifiers)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt`.`Example_IDentifier` IN (%s)", placeholders))
		for _, Example_IDentifier := range Example_IDentifiers {
			args = append(args, Example_IDentifier)
		}
	}

	if len(Topics) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Topics)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt`.`Topic` IN (%s)", placeholders))
		for _, Topic := range Topics {
			args = append(args, Topic)
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
			config Mqtt__type

			Creation_User sql.NullInt64
			Topic_Push    sql.NullString
			Topic_Down    sql.NullString
			Topic_Alarm   sql.NullString
		)
		err = rows.Scan(
			&config.Id,
			&config.Type,
			&config.Example_IDentifier,
			&Topic_Push,
			&Topic_Down,
			&Topic_Alarm,
			&config.Creation_Time,
			&Creation_User,
		)
		if err != nil {
			log.Print(err.Error())
			return err
		}
		config.Topic_Push = Topic_Push.String
		config.Topic_Down = Topic_Down.String
		config.Topic_Alarm = Topic_Alarm.String
		config.Creation_User = uint(Creation_User.Int64)
		callback(config)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

// Mqtt-》查询配置
// 传递：driveid 设备 id, page 页码，pageSize 每页数量
// 返回：configs 配置，err 错误
func Mqtt__Query(Types []string, Example_IDentifiers []string, Topics []string, page uint, pageSize uint) (configs []Mqtt__type, err error) {
	err = Mqtt__Query_Callback(Types, Example_IDentifiers, Topics, page, pageSize, func(config Mqtt__type) {
		configs = append(configs, config)
	})
	return
}

// Mqtt-》增加配置
// 传递：config 配置数组形式
// 返回：err 错误
func Mqtt__Add(configs ...Mqtt__Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}

	// 3. SQL 插入（包含 Id 字段）
	baseQuery := `
		INSERT INTO Mqtt (
			Type,
			Example_IDentifier,
			Topic_Push,
			Topic_Down,
			Topic_Alarm,
			Creation_Time,
			Creation_User
		) VALUES
	`

	var (
		args              []interface{}
		valuePlaceholders []string
	)
	now := time.Now()

	// 4. 构建批量参数
	for _, cfg := range configs {
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			cfg.Type,
			cfg.Example_IDentifier,
			sql.NullString{
				String: cfg.Topic_Push,
				Valid:  cfg.Topic_Push != "",
			},
			sql.NullString{
				String: cfg.Topic_Down,
				Valid:  cfg.Topic_Down != "",
			},
			sql.NullString{
				String: cfg.Topic_Alarm,
				Valid:  cfg.Topic_Alarm != "",
			},
			now,
			sql.NullInt16{
				Int16: int16(cfg.Creation_User),
				Valid: cfg.Creation_User != 0,
			},
		)
	}

	// 5. 拼接 SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 6. 执行插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 Points_Config 失败: %w", err)
	}

	return nil
}

// Mqtt-》修改配置
// 传递：config 配置
// 返回：conid 获取自增的Id，err 错误
func Mqtt__Update(configs ...Mqtt__Update_type) (err error) {
	// 1. 空列表校验
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	now := time.Now()

	// 2. 遍历逐个更新
	for i, cfg := range configs {
		// 2.1 单条配置参数校验
		if cfg.Id == 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置不能为空", i)
			return
		}

		// 2.2 动态拼接SET子句
		var setClauses []string
		var args []interface{}
		if cfg.Type != "" {
			setClauses = append(setClauses, "`Type` = ?")
			args = append(args, cfg.Type)
		}
		if cfg.Example_IDentifier != "" {
			setClauses = append(setClauses, "`Example_IDentifier` = ?")
			args = append(args, cfg.Example_IDentifier)
		}
		if cfg.Topic_Push != "null" {
			setClauses = append(setClauses, "`Topic_Push` = ?")
			args = append(args, sql.NullString{
				String: cfg.Topic_Push,
				Valid:  cfg.Topic_Push != "null" || cfg.Topic_Push == "undefined",
			})
		}
		if cfg.Topic_Down != "null" {
			setClauses = append(setClauses, "`Topic_Down` = ?")
			args = append(args, sql.NullString{
				String: cfg.Topic_Down,
				Valid:  cfg.Topic_Down != "null" || cfg.Topic_Down == "undefined",
			})
		}
		if cfg.Topic_Alarm != "null" {
			setClauses = append(setClauses, "`Topic_Alarm` = ?")
			args = append(args, sql.NullString{
				String: cfg.Topic_Alarm,
				Valid:  cfg.Topic_Alarm != "null" || cfg.Topic_Alarm == "undefined",
			})
		}
		if cfg.Creation_User != 0 {
			setClauses = append(setClauses, "`Creation_User` = ?")
			args = append(args, cfg.Creation_User)
		}

		setClauses = append(setClauses, "`Creation_Time` = ?")
		args = append(args, now)

		// 2.3 校验更新字段i
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段至少传一个", i)
			return
		}

		// 2.4 拼接SQL并执行
		query := fmt.Sprintf("UPDATE `Mqtt` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		args = append(args, cfg.Id)

		_, err = DB.Exec(query, args...)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s, 参数:%v", i, cfg.Id, err, query, args)
			return
		}
	}

	return
}

// Mqtt-》删除配置
// 传递：ids 删除的id数组
// 返回：err 错误
func Mqtt__Del(ids ...uint) (err error) {
	// 1. 遍历逐个
	for idx, id := range ids {

		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx)
			return
		}

		query := "DELETE FROM `Mqtt` WHERE `Id` = ? "
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
***************mqtt配置结构体***************
 */

// 结构体
type Mqtt_Points__type struct {
	Id          uint   // 点位id
	Mqtt_Id     uint   // MQTT id
	Tag         string // 点位标识符
	RW_Cancel   string // 读写方式
	Value_Type  string // 值类型
	History     string // 存储
	Alarm       string // 报警
	Alarm_Group int    // 报警组
	Format_Path string // 格式路径

	Creation_Time time.Time // 创建时间
	Creation_User uint      // 创建的用户id
}

type Mqtt_Points__Add_type struct {
	Mqtt_Id     uint   // MQTT id
	Tag         string // 点位标识符
	RW_Cancel   string // 读写方式
	Value_Type  string // 值类型
	History     string // 存储
	Alarm       string // 报警
	Alarm_Group int    // 报警组
	Format_Path string // 格式路径

	Creation_User uint // 创建的用户id
}

type Mqtt_Points__Update_type struct {
	Mqtt_Points__type
}

func Mqtt_Points__Count(Mqtt_Ids []uint, Tags []string, RW_Cancels []string, page uint, pageSize uint) (Count uint, err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			COUNT(Mqtt_Points.Id) 
		FROM Mqtt 
	`
	var args []interface{} // 存储SQL参数，防止SQL注入
	var whereConditions []string

	if len(Mqtt_Ids) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Mqtt_Ids)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt_Points`.`Mqtt_Id` IN (%s)", placeholders))
		for _, Mqtt_Id := range Mqtt_Ids {
			args = append(args, Mqtt_Id)
		}
	}

	if len(Tags) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Tags)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt_Points`.`Tag` IN (%s)", placeholders))
		for _, Tag := range Tags {
			args = append(args, Tag)
		}
	}

	if len(RW_Cancels) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(RW_Cancels)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt_Points`.`RW_Cancel` IN (%s)", placeholders))
		for _, RW_Cancel := range RW_Cancels {
			args = append(args, RW_Cancel)
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
		log.Printf("查询点位配置无数据")
		Count = 0
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR 查询点位配置失败，错误：%v, SQL:%s, 参数:%v", err, baseQuery, args)
		log.Print(err)
		return
	}

	return
}

func Mqtt_Points__Query_Callback(Mqtt_Ids []uint, Tags []string, RW_Cancels []string, page uint, pageSize uint, callback func(Mqtt_Points__type)) (err error) {
	// 1. 初始化SQL和参数切片，避免多次拼接字符串，提升可读性和安全性
	baseQuery := `
		SELECT
			Mqtt_Points.Id,
			Mqtt_Points.Mqtt_Id,
			Mqtt_Points.Tag,
			Mqtt_Points.RW_Cancel,
			Mqtt_Points.Value_Type,
			Mqtt_Points.History,
			Mqtt_Points.Alarm,
			Mqtt_Points.Alarm_Group,
			Mqtt_Points.Format_Path,
			Mqtt_Points.Creation_Time,
			Mqtt_Points.Creation_User
		FROM Mqtt_Points
	`
	var whereConditions []string
	var args []interface{} // 存储SQL参数，防止SQL注入

	if len(Mqtt_Ids) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Mqtt_Ids)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt_Points`.`Mqtt_Id` IN (%s)", placeholders))
		for _, Mqtt_Id := range Mqtt_Ids {
			args = append(args, Mqtt_Id)
		}
	}

	if len(Tags) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(Tags)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt_Points`.`Tag` IN (%s)", placeholders))
		for _, Tag := range Tags {
			args = append(args, Tag)
		}
	}

	if len(RW_Cancels) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(RW_Cancels)), ",")
		whereConditions = append(whereConditions, fmt.Sprintf("`Mqtt_Points`.`RW_Cancel` IN (%s)", placeholders))
		for _, RW_Cancel := range RW_Cancels {
			args = append(args, RW_Cancel)
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
			config Mqtt_Points__type

			Creation_User sql.NullInt64
			History       sql.NullString
			Alarm         sql.NullString
			Alarm_Group   sql.NullInt64
			Format_Path   sql.NullString
		)
		err = rows.Scan(
			&config.Id,
			&config.Mqtt_Id,
			&config.Tag,
			&config.RW_Cancel,
			&config.Value_Type,
			&History,
			&Alarm,
			&Alarm_Group,
			&Format_Path,
			&config.Creation_Time,
			&Creation_User,
		)
		if err != nil {
			log.Print(err.Error())
			return err
		}
		config.History = History.String
		config.Alarm = Alarm.String
		config.Alarm_Group = int(Alarm_Group.Int64)
		config.Format_Path = Format_Path.String
		config.Creation_User = uint(Creation_User.Int64)
		callback(config)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

func Mqtt_Points__Query(Mqtt_Ids []uint, Tags []string, RW_Cancels []string, page uint, pageSize uint) (r []Mqtt_Points__type, err error) {
	err = Mqtt_Points__Query_Callback(Mqtt_Ids, Tags, RW_Cancels, page, pageSize, func(config Mqtt_Points__type) {
		r = append(r, config)
	})
	return
}

func Mqtt_Points__Add(configs ...Mqtt_Points__Add_type) (err error) {
	// 1. 基础校验：空列表直接返回
	if len(configs) == 0 {
		return fmt.Errorf("批量新增失败：待新增配置列表为空")
	}

	// 3. SQL 插入（包含 Id 字段）
	baseQuery := `
		INSERT INTO Mqtt_Points (
			Mqtt_Id,
			Tag,
			RW_Cancel,
			Value_Type,
			History,
			Alarm,
			Alarm_Group,
			Format_Path,
			Creation_Time,
			Creation_User
		) VALUES
	`

	var (
		args              []interface{}
		valuePlaceholders []string
	)
	now := time.Now()

	// 4. 构建批量参数
	for _, cfg := range configs {
		valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			cfg.Mqtt_Id,
			cfg.Tag,
			cfg.RW_Cancel,
			cfg.Value_Type,
			sql.NullString{
				String: cfg.History,
				Valid:  cfg.History != "",
			},
			sql.NullString{
				String: cfg.Alarm,
				Valid:  cfg.Alarm != "",
			},
			sql.NullInt64{
				Int64: int64(cfg.Alarm_Group),
				Valid: cfg.Alarm_Group != 0,
			},
			sql.NullString{
				String: cfg.Format_Path,
				Valid:  cfg.Format_Path != "",
			},
			now,
			sql.NullInt64{
				Int64: int64(cfg.Creation_User),
				Valid: cfg.Creation_User != 0,
			},
		)
	}

	// 5. 拼接 SQL
	query := baseQuery + strings.Join(valuePlaceholders, ", ")

	// 6. 执行插入
	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("批量插入 Points_Config 失败: %w", err)
	}

	return nil
}

func Mqtt_Points__Update(configs ...Mqtt_Points__Update_type) (err error) {
	// 1. 空列表校验
	if len(configs) == 0 {
		err = fmt.Errorf("ERROR 待更新配置列表为空")
		return
	}

	now := time.Now()

	// 2. 遍历逐个更新
	for i, cfg := range configs {
		// 2.1 单条配置参数校验
		if cfg.Id == 0 {
			err = fmt.Errorf("批量新增失败：第%d条配置不能为空", i)
			return
		}

		// 2.2 动态拼接SET子句
		var setClauses []string
		var args []interface{}
		if cfg.Mqtt_Id != 0 {
			setClauses = append(setClauses, "`Mqtt_Id` = ?")
			args = append(args, cfg.Mqtt_Id)
		}
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
		if cfg.Creation_User != 0 {
			setClauses = append(setClauses, "`Creation_User` = ?")
			args = append(args, cfg.Creation_User)
		}
		if cfg.History != "" {
			setClauses = append(setClauses, "`History` = ?")
			args = append(args, sql.NullString{
				String: cfg.History,
				Valid:  cfg.History != "",
			})
		}
		if cfg.Alarm != "" {
			setClauses = append(setClauses, "`Alarm` = ?")
			args = append(args, sql.NullString{
				String: cfg.Alarm,
				Valid:  cfg.Alarm != "",
			})
		}
		if cfg.Alarm_Group != 0 {
			setClauses = append(setClauses, "`Alarm_Group` = ?")
			args = append(args, sql.NullInt64{
				Int64: int64(cfg.Alarm_Group),
				Valid: cfg.Alarm_Group != 0,
			})
		}
		if cfg.Format_Path != "" {
			setClauses = append(setClauses, "`Format_Path` = ?")
			args = append(args, sql.NullString{
				String: cfg.Format_Path,
				Valid:  cfg.Format_Path != "",
			})
		}

		setClauses = append(setClauses, "`Creation_Time` = ?")
		args = append(args, now)

		// 2.3 校验更新字段i
		if len(setClauses) == 0 {
			err = fmt.Errorf("ERROR 第%d条配置未指定任何更新字段至少传一个", i)
			return
		}

		// 2.4 拼接SQL并执行
		query := fmt.Sprintf("UPDATE `Mqtt_Points` SET %s WHERE `Id` = ?", strings.Join(setClauses, ", "))
		args = append(args, cfg.Id)

		_, err = DB.Exec(query, args...)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s, 参数:%v", i, cfg.Id, err, query, args)
			return
		}
	}

	return
}

func Mqtt_Points__Del(ids ...uint) (err error) {
	// 1. 遍历逐个
	for idx, id := range ids {

		if id == 0 {
			err = fmt.Errorf("ERROR 第%d条配置ID(Id)不能为空", idx)
			return
		}

		query := "DELETE FROM `Mqtt_Points` WHERE `Id` = ? "
		// 修改数据库
		_, err = DB.Exec(query, id)
		if err != nil {
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%v, SQL:%s", idx, id, err, query)
			return
		}

	}
	return
}
