/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: mysql 用户逻辑
 */

package mysql

import (
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// tag 格式正则：支持两种格式
	// 1. //设备id//标识符（包含设备id）
	// 2. /标识符（不包含设备id）
	// - 设备id不能为空，不含/
	// - 标识符不能为空，中间可有/，但不能以/结尾，不能有//
	Regex_Tag_Format = `^(//[^/]+//[^/]+(/[^/]+)*|/[^/]+(/[^/]+)*)$`

	// Conversion 格式正则：空字符串、"0" 或含变量 x 的换算表达式
	// - 空字符串或 "0" 表示无换算
	// - 非空非"0"时必须包含变量 x，允许数字、运算符(+,-,*,/)、括号、逗号、小数点、空白，以及函数名（如 round2）
	// - 示例：x*2+10、(x-32)*5/9、x/100.0*3600、round2(x/3, 2)
	// - 注：只允许 round2 一个自定义函数 + 内置函数；非法标识符/函数会在运行时 expr 编译报错，此处仅作脏数据初筛
	// - 反向换算仍仅支持一元一次方程（非线性表达式无法反推）
	Regex_Conversion_Format = `^(|0|[0-9a-zA-Z\s\+\-\*\/\.\(\),]*x[0-9a-zA-Z\s\+\-\*\/\.\(\),]*)$`

	// Config 基础防注入正则（命中即拒绝）：单引号、反引号、反斜杠、SQL 注释标记(-- 与 /* */)、常见 SQL 关键字
	// 只作脏数据兜底：配置值一律走 ? 占位符进 SQL，注入本就不可达，这里防的是把 SQL 片段当配置存进来的误操作
	// 不拦双引号与分号：驱动配置是 JSON（必须带双引号），Modbus 点位配置是 ;/: 分隔串
	Regex_Config_Forbidden = "(?i)('|`|\\\\|--|/\\*|\\*/|\\b(select|insert|update|delete|drop|alter|create|truncate|union|exec|declare|sleep|benchmark|load_file|information_schema|into\\s+outfile)\\b)"

	// Config_Max_Length 配置字段长度上限，防超长报文（正常驱动 JSON 配置不足 300 字节）
	Config_Max_Length = 4096
)

// 配置格式校验的三条约定（改这里请同步前端 src/utils/configFormat.ts）：
//
//	Drive_Config.Config —— 必须是合法 JSON 对象：两个驱动的 Drive_Config_Switch 都直接 json.Unmarshal 到结构体，
//	                      非 JSON 会拖到服务重启/驱动初始化才炸，所以在入库这一层就挡住
//	Point_Config.Config —— 不做格式校验，只过 Config__Validate_Basic：Modbus 是 : 分隔串、S7 是地址串或 JSON、
//	                      历史数据里还有 ; 分隔串，具体格式由各驱动 Point_Config_Switch 负责解释
//	Alarm/History_Config.Config —— 装的是 expr 表达式/采集周期，格式由业务层解释，此处不校验

// configForbiddenRegex 预编译的基础防注入正则，供 Config__Validate_Basic 使用
var configForbiddenRegex = regexp.MustCompile(Regex_Config_Forbidden)

// Config__Validate_JSON 驱动配置校验：必须能解析成 JSON 对象
func Config__Validate_JSON(name, s string) error {
	t := strings.TrimSpace(s)
	if t == "" {
		return fmt.Errorf("%s 不能为空", name)
	}
	if len(t) > Config_Max_Length {
		return fmt.Errorf("%s 超长（%d 字符，上限 %d）", name, len(t), Config_Max_Length)
	}
	// 先验前缀再交给解析器：旧版 ; 分隔配置会报「以 { 开头」而不是费解的 JSON 错误
	if !strings.HasPrefix(t, "{") {
		return fmt.Errorf(`%s 必须是 JSON 对象（以 { 开头），例如 {"Address":"192.168.1.1:502"}`, name)
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(t), &probe); err != nil {
		return fmt.Errorf("%s 不是合法的 JSON，请检查括号与逗号: %w", name, err)
	}
	return nil
}

// Config__Validate_Basic 配置字段基础校验：非空、不超长、无控制字符、无 SQL 注入片段
// 允许 JSON、;/: 分隔串、S7 地址串、含空格等所有合法形态，格式正确性交给驱动层
func Config__Validate_Basic(name, s string) error {
	if s == "" {
		return fmt.Errorf("%s 不能为空", name)
	}
	if len(s) > Config_Max_Length {
		return fmt.Errorf("%s 超长（%d 字符，上限 %d）", name, len(s), Config_Max_Length)
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return r < 0x20 && r != '\t' && r != '\n' && r != '\r'
	}) >= 0 {
		return fmt.Errorf("%s 含控制字符，请重新输入", name)
	}
	if configForbiddenRegex.MatchString(s) {
		return fmt.Errorf("%s 含非法字符（单引号、反引号、反斜杠、-- 注释或 SQL 关键字）", name)
	}
	return nil
}

// Log_Error_Window 同类错误日志的打印窗口，窗口内只打印首条
const Log_Error_Window = 30 * time.Second

var (
	logErrMu       sync.Mutex
	logErrLast     = make(map[string]time.Time)
	logErrSuppress = make(map[string]int)
)

// logErrorThrottled 限流打印重复错误
// 传入：key — 错误类别（同类错误共用一个窗口）；msg — 完整日志文本
// 作用：数据库不可用等持续故障场景下，每类错误每个窗口最多打印一条，避免日志刷屏
func logErrorThrottled(key, msg string) {
	logErrMu.Lock()
	defer logErrMu.Unlock()

	now := time.Now()
	if last, ok := logErrLast[key]; ok && now.Sub(last) < Log_Error_Window {
		logErrSuppress[key]++
		return
	}
	logErrLast[key] = now

	if n := logErrSuppress[key]; n > 0 {
		logErrSuppress[key] = 0
		log.Printf("%s（上一窗口内同类错误已抑制 %d 条）", msg, n)
		return
	}
	log.Print(msg)
}

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

// applyIdScope 把 RBAC 实例级授权的 id 白名单追加为 `Id` IN (...) 过滤条件。
// 约定：idScope 为空/nil 表示"不加该过滤"（全量）；"授权集为空"由中间件提前判 403，不会走到这里，
// 故 len==0 恒等价为不过滤。必须在拼 LIMIT/OFFSET 之前调用，保证分页与 total 口径一致（同源）。
func applyIdScope(whereConditions *[]string, args *[]interface{}, idScope []uint) {
	if len(idScope) == 0 {
		return
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(idScope)), ",")
	*whereConditions = append(*whereConditions, fmt.Sprintf("`Id` IN (%s)", placeholders))
	for _, id := range idScope {
		*args = append(*args, id)
	}
}

// firstIdScope 从 variadic idScope 参数取首个白名单（未传=nil=不过滤）。
func firstIdScope(idScope [][]uint) []uint {
	if len(idScope) > 0 {
		return idScope[0]
	}
	return nil
}

// selectAllIds 查询指定表的全部行 Id（升序），供中间件枚举实例级 scope 的候选 id。
func selectAllIds(table string) (ids []uint, err error) {
	query := fmt.Sprintf("SELECT `Id` FROM `%s` ORDER BY `Id`", table)
	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ERROR 查询 %s 全部ID失败:%w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uint
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	return
}

// Drive_Config__AllIds 驱动表全部行 Id（实例级 scope 枚举候选）。
func Drive_Config__AllIds() (ids []uint, err error) { return selectAllIds("Drive_Config") }

// Point_Config__AllIds 点位表全部行 Id（实例级 scope 枚举候选）。
func Point_Config__AllIds() (ids []uint, err error) { return selectAllIds("Point_Config") }

// Alarm_Config__AllIds 报警表全部行 Id（实例级 scope 枚举候选）。
func Alarm_Config__AllIds() (ids []uint, err error) { return selectAllIds("Alarm_Config") }

// History_Config__AllIds 历史表全部行 Id（实例级 scope 枚举候选）。
func History_Config__AllIds() (ids []uint, err error) { return selectAllIds("History_Config") }

// 驱动-》查询配置（回调）
// idScope：实例级授权 id 白名单，nil/空=不过滤；须早于 LIMIT 生效。
func Drive_Config__Query_Callback(page uint, pageSize uint, idScope []uint, callback func(Drive_Config_Query_type)) (err error) {
	baseQuery := `
		SELECT Id, Type, Name, Config, Points_Length, Creation_Time
		FROM Drive_Config
	`
	var whereConditions []string
	var args []interface{}

	applyIdScope(&whereConditions, &args, idScope)
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
		err = fmt.Errorf("ERROR 查询驱动配置失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
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
		err = fmt.Errorf("ERROR 遍历驱动配置结果集失败，错误:%w", err)
		log.Print(err)
	}
	return
}

// 驱动-》查询配置
func Drive_Config__Query(page uint, pageSize uint, idScope ...[]uint) (configs []Drive_Config_Query_type, err error) {
	err = Drive_Config__Query_Callback(page, pageSize, firstIdScope(idScope), func(cfg Drive_Config_Query_type) {
		configs = append(configs, cfg)
	})
	return
}

// 驱动-》查询数量
func Drive_Config__Count(page uint, pageSize uint, idScope ...[]uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `Drive_Config`"
	var whereConditions []string
	var args []interface{}

	applyIdScope(&whereConditions, &args, firstIdScope(idScope))
	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
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

	for i, cfg := range configs {
		if cfg.Type == "" {
			return fmt.Errorf("批量新增失败：第%d条配置Type不能为空", i+1)
		}
		if err := Config__Validate_JSON("Config", cfg.Config); err != nil {
			return fmt.Errorf("批量新增失败：第%d条数据 %w", i+1, err)
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
			if err = Config__Validate_JSON("Config", config.Config); err != nil {
				err = fmt.Errorf("ERROR 第%d条数据 %w", idx+1, err)
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
			err = fmt.Errorf("ERROR 第%d条配置更新失败, ID:%d, 错误:%w", idx+1, config.Id, errExec)
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
			err = fmt.Errorf("ERROR 第%d条配置删除失败, ID:%d, 错误:%w", idx+1, id, err)
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
func Drive_Config__Search(name string, types []string, quantity uint, idScope ...[]uint) (configs []Drive_Config_Query_type, err error) {
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
	applyIdScope(&whereConditions, &args, firstIdScope(idScope))

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	baseQuery += " LIMIT ?"
	args = append(args, quantity)

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 搜索驱动配置失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
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
		err = fmt.Errorf("ERROR 遍历驱动配置结果集失败，错误:%w", err)
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
			err = fmt.Errorf("ERROR 修改点位数量错误 %w", err)
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
// 传递：driveid 设备id，page 页码，pageSize 每页数量，idScope 实例级授权 id 白名单(nil=不过滤)，callback 回调函数
func Point_Config__Query_Callback(driveid []uint, page uint, pageSize uint, idScope []uint, callback func(Point_Config_Query_type)) (err error) {
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
	applyIdScope(&whereConditions, &args, idScope)

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
		err = fmt.Errorf("ERROR 查询点位配置失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
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
		err = fmt.Errorf("ERROR 遍历点位配置结果集失败，错误:%w", err)
		log.Print(err)
	}
	return
}

// 点位-》查询配置
// 传递：driveid 设备id, page 页码，pageSize 每页数量
func Point_Config__Query(driveid []uint, page uint, pageSize uint, idScope ...[]uint) (configs []Point_Config_Query_type, err error) {
	err = Point_Config__Query_Callback(driveid, page, pageSize, firstIdScope(idScope), func(cfg Point_Config_Query_type) {
		configs = append(configs, cfg)
	})
	return
}

// 点位-》查询数量
func Point_Config__Count(driveid []uint, page uint, pageSize uint, idScope ...[]uint) (count uint, err error) {
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
	applyIdScope(&whereConditions, &args, firstIdScope(idScope))

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
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
		// Config 不做格式校验（驱动层 Point_Config_Switch 自己解释），只挡注入片段与控制字符
		if err := Config__Validate_Basic("Config", cfg.Config); err != nil {
			return fmt.Errorf("批量新增失败：第%d条数据 %w", i, err)
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
			if err = Config__Validate_Basic("Config", cfg.Config); err != nil {
				err = fmt.Errorf("批量更新失败：第%d条配置 %w", i, err)
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
			err = fmt.Errorf("ERROR 第%d条点位删除失败, ID:%d, 错误:%w", idx+1, id, err)
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

// Point_Config__Query__RW_type 点位读写属性（写值入口校验只读点位用）
type Point_Config__Query__RW_type struct {
	Id        uint
	Tag       string
	RW_Cancel int // 读写方式 1：禁止； 2：只读； 3：只写； 4：读写
}

// Point_Config__Query__RW 按点位 id 批量查询 Tag 与读写方式，供写值入口统一校验
func Point_Config__Query__RW(ids ...uint) (list []Point_Config__Query__RW_type, err error) {
	if len(ids) == 0 {
		return
	}

	// 去重
	slices.Sort(ids)
	ids = slices.Compact(ids)

	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, v := range ids {
		if v == 0 {
			return nil, fmt.Errorf("ERROR Id 不能等于0")
		}
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	query := fmt.Sprintf(`
			SELECT
				Id,
				Tag,
				RW_Cancel
			FROM
				Point_Config
			WHERE
				Id IN (%s)
		`, strings.Join(placeholders, ","))

	rows, err := DB.Query(query, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 批量查询RW_Cancel失败: %w", err)
		log.Println(err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item Point_Config__Query__RW_type
		if err = rows.Scan(&item.Id, &item.Tag, &item.RW_Cancel); err != nil {
			err = fmt.Errorf("ERROR scan行失败: %w", err)
			log.Println(err.Error())
			return nil, err
		}
		list = append(list, item)
	}

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("ERROR rows迭代错误: %w", err)
		log.Println(err.Error())
		return nil, err
	}

	return list, nil
}

// Point_Config__Query__IdTag_type 点位id 与点位标识 Tag 的成对结果（Tag <-> id 互查用）
type Point_Config__Query__IdTag_type struct {
	Id  uint
	Tag string
}

// pointConfig__Query__IdTag 通用查询：WHERE column IN (?,?...) 返回 []{Id, Tag}
// column 只由下面两个导出函数写死为 "Id"/"Tag"，不接收外部输入，避免拼接进 SQL
func pointConfig__Query__IdTag(column string, args []interface{}) (list []Point_Config__Query__IdTag_type, err error) {
	if len(args) == 0 {
		return
	}

	// 生成 ?,?,?
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf(`
			SELECT
				Id,
				Tag
			FROM
				Point_Config
			WHERE
				%s IN (%s)
		`, column, placeholders)

	rows, err := DB.Query(query, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 按%s批量查询Id/Tag失败: %w", column, err)
		log.Println(err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item Point_Config__Query__IdTag_type
		if err = rows.Scan(&item.Id, &item.Tag); err != nil {
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

// Point_Config__Query__IdTag_ByTags 按点位标识 Tag 批量反查点位 id（精确匹配）
// 传递：tags 点位标识，空串忽略；返回：[]{Id, Tag}，库里不存在的 Tag 不会出现在结果中
func Point_Config__Query__IdTag_ByTags(tags ...string) (list []Point_Config__Query__IdTag_type, err error) {
	if len(tags) == 0 {
		return
	}

	// 去重
	tmp_tags := append([]string(nil), tags...)
	slices.Sort(tmp_tags)
	tmp_tags = slices.Compact(tmp_tags)

	args := make([]interface{}, 0, len(tmp_tags))
	for _, v := range tmp_tags {
		if v == "" {
			continue
		}
		args = append(args, v)
	}

	return pointConfig__Query__IdTag("Tag", args)
}

// Point_Config__Query__IdTag_ByIds 按点位 id 批量查询点位标识 Tag
// 传递：ids 点位id，0 忽略；返回：[]{Id, Tag}，库里不存在的 id 不会出现在结果中
func Point_Config__Query__IdTag_ByIds(ids ...uint) (list []Point_Config__Query__IdTag_type, err error) {
	if len(ids) == 0 {
		return
	}

	// 去重
	tmp_ids := append([]uint(nil), ids...)
	slices.Sort(tmp_ids)
	tmp_ids = slices.Compact(tmp_ids)

	args := make([]interface{}, 0, len(tmp_ids))
	for _, v := range tmp_ids {
		if v == 0 {
			continue
		}
		args = append(args, v)
	}

	return pointConfig__Query__IdTag("Id", args)
}

// Point_Config__Query__Tag 按点位id查询点位标识 Tag
// 传递：id 点位id
// 返回：tag 点位标识；点位不存在时返回错误
func Point_Config__Query__Tag(id uint) (tag string, err error) {
	if id == 0 {
		err = fmt.Errorf("ERROR Id 不能等于0")
		return
	}

	err = DB.QueryRow("SELECT `Tag` FROM `Point_Config` WHERE `Id` = ?", id).Scan(&tag)
	if err != nil {
		err = fmt.Errorf("ERROR 查询点位Tag失败(id=%d): %w", id, err)
		log.Println(err.Error())
	}

	return
}

// Point_Config__Search 搜索点位配置
// 功能：按 Tag 模糊搜索 + Drive_Id 精确过滤，返回匹配结果
// 传入：tag — 点位标签（LIKE 匹配，调用方需自带 % 通配符，空串不过滤）；driveIds — 驱动id列表（IN 精确匹配，空切片不过滤）；quantity — 返回数量上限（最大 30）
// 返回：匹配的点位配置列表，错误
func Point_Config__Search(tag string, driveIds []uint, quantity uint, idScope ...[]uint) (configs []Point_Config_Query_type, err error) {
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
	applyIdScope(&whereConditions, &args, firstIdScope(idScope))

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	baseQuery += " LIMIT ?"
	args = append(args, quantity)

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 搜索点位配置失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
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
		err = fmt.Errorf("ERROR 遍历点位配置结果集失败，错误:%w", err)
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
func Alarm_Config__Query_Callback(pointid []uint, page uint, pageSize uint, idScope []uint, callback func(Alarm_Config_Query_type)) (err error) {
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
	applyIdScope(&whereConditions, &args, idScope)

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
		// 该函数位于报警判断热路径，数据库不可用时按窗口限流打印，避免刷屏
		err = fmt.Errorf("ERROR 查询报警配置失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
		logErrorThrottled("alarm_config_query", err.Error())
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
		err = fmt.Errorf("ERROR 遍历报警配置结果集失败，错误:%w", err)
		log.Print(err)
	}
	return
}

// 报警-》查询配置
func Alarm_Config__Query(pointid []uint, page uint, pageSize uint, idScope ...[]uint) (configs []Alarm_Config_Query_type, err error) {
	err = Alarm_Config__Query_Callback(pointid, page, pageSize, firstIdScope(idScope), func(cfg Alarm_Config_Query_type) {
		configs = append(configs, cfg)
	})
	return
}

// 报警-》查询数量
func Alarm_Config__Count(pointid []uint, page uint, pageSize uint, idScope ...[]uint) (count uint, err error) {
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
	applyIdScope(&whereConditions, &args, firstIdScope(idScope))

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
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
			err = fmt.Errorf("ERROR 第%d条报警删除失败, ID:%d, 错误:%w", idx+1, id, err)
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
func Alarm_Config__Search(tag string, pointIds []uint, groups []int, quantity uint, idScope ...[]uint) (configs []Alarm_Config_Query_type, err error) {
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
	applyIdScope(&whereConditions, &args, firstIdScope(idScope))

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	baseQuery += " LIMIT ?"
	args = append(args, quantity)

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 搜索报警配置失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
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
		err = fmt.Errorf("ERROR 遍历报警配置结果集失败，错误:%w", err)
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

	// 按时间倒序排列：最近发生的报警放在最前面（时间相同时用 Id 兜底）
	baseQuery += " ORDER BY `Time` DESC, `Id` DESC"

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		err = fmt.Errorf("ERROR 查询报警数据失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
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
		err = fmt.Errorf("ERROR 遍历报警数据结果集失败，错误:%w", err)
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
	if errors.Is(err, sql.ErrNoRows) {
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
			err = fmt.Errorf("ERROR 第%d条报警数据删除失败, ID:%d, 错误:%w", idx+1, id, err)
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
func History_Config__Query_Callback(pointid []uint, page uint, pageSize uint, idScope []uint, callback func(History_Config_Query_type)) (err error) {
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
	applyIdScope(&whereConditions, &args, idScope)

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
		err = fmt.Errorf("ERROR 查询历史配置失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
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
		err = fmt.Errorf("ERROR 遍历历史配置结果集失败，错误:%w", err)
		log.Print(err)
	}
	return
}

// 历史-》查询配置
func History_Config__Query(pointid []uint, page uint, pageSize uint, idScope ...[]uint) (configs []History_Config_Query_type, err error) {
	err = History_Config__Query_Callback(pointid, page, pageSize, firstIdScope(idScope), func(cfg History_Config_Query_type) {
		configs = append(configs, cfg)
	})
	return
}

// 历史-》查询数量
func History_Config__Count(pointid []uint, page uint, pageSize uint, idScope ...[]uint) (count uint, err error) {
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
	applyIdScope(&whereConditions, &args, firstIdScope(idScope))

	if len(whereConditions) > 0 {
		baseQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
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
			err = fmt.Errorf("ERROR 第%d条历史删除失败, ID:%d, 错误:%w", idx+1, id, err)
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
		err = fmt.Errorf("ERROR 查询APP_Config失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
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
		err = fmt.Errorf("ERROR 遍历APP_Config结果集失败，错误:%w", err)
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
	if errors.Is(err, sql.ErrNoRows) {
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
			err = fmt.Errorf("ERROR 第%d条APP_Config删除失败, ID:%d, 错误:%w", idx+1, id, err)
			return
		}
	}
	return
}

// APP_Config__GetByKet 根据键名获取值
func APP_Config__GetByKet(ket string) (value string, err error) {
	query := "SELECT `Value` FROM `APP_Config` WHERE `Ket` = ? LIMIT 1"
	err = DB.QueryRow(query, ket).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
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

/*
***************统一鉴权账号 account***************
 */

// Account_type 账号完整结构体
type Account_type struct {
	Id                uint      // 递增id
	Access_Key        string    // 唯一身份标识（同时作为服务调用 AK、前端登录用户名）
	Secret_Key        string    // HMAC 签名密钥（前端不可获取）
	Login_Pwd_Hash    string    // bcrypt 登录密码哈希（空则不能网页登录）
	Enable            int       // 1=启用 0=禁用
	Expire_At         time.Time // 账号到期时间（零值=永不过期）
	Access_Token_TTL  uint      // access_token 时效（秒）
	Refresh_Token_TTL uint      // refresh_token 时效（秒）
	Create_Time       time.Time // 创建时间
	Update_Time       time.Time // 更新时间
}

// Account_Add_type 账号新增/同步结构体（不含 Id/时间，由内部生成或写入）
type Account_Add_type struct {
	Access_Key        string    // 唯一身份标识（必填）
	Secret_Key        string    // HMAC 签名密钥（必填）
	Login_Pwd_Hash    string    // bcrypt 登录密码哈希（可空）
	Enable            int       // 1=启用 0=禁用（传入 0 视为禁用；默认值由调用方决定）
	Expire_At         time.Time // 到期时间（零值=永不过期，落库为 NULL）
	Access_Token_TTL  uint      // access_token 时效（秒），0 时回退默认 900
	Refresh_Token_TTL uint      // refresh_token 时效（秒），0 时回退默认 604800
}

const (
	Default_Access_Token_TTL  uint = 900    // access_token 默认时效 15 分钟
	Default_Refresh_Token_TTL uint = 604800 // refresh_token 默认时效 7 天
)

// normalizeTTL 时效为 0 时回退默认值
func normalizeTTL(ttl, def uint) uint {
	if ttl == 0 {
		return def
	}
	return ttl
}

// Account__Query_By_Access_Key 按 access_key 查询单条账号（走 uk_access_key 唯一索引）
// 返回：账号、是否存在、错误。未命中返回 found=false 且不报错。
func Account__Query_By_Access_Key(accessKey string) (acc Account_type, found bool, err error) {
	if accessKey == "" {
		return acc, false, fmt.Errorf("Account__Query_By_Access_Key: access_key 不能为空")
	}

	query := `
		SELECT Id, Access_Key, Secret_Key, Login_Pwd_Hash, Enable, Expire_At,
			   Access_Token_TTL, Refresh_Token_TTL, Create_Time, Update_Time
		FROM account
		WHERE Access_Key = ?
		LIMIT 1
	`

	var (
		pwdHash sql.NullString
		expire  sql.NullTime
	)
	err = DB.QueryRow(query, accessKey).Scan(
		&acc.Id, &acc.Access_Key, &acc.Secret_Key, &pwdHash, &acc.Enable, &expire,
		&acc.Access_Token_TTL, &acc.Refresh_Token_TTL, &acc.Create_Time, &acc.Update_Time,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return acc, false, nil
	}
	if err != nil {
		return acc, false, fmt.Errorf("[Account__Query_By_Access_Key] 查询失败 | access_key=%s | err=%w", accessKey, err)
	}

	acc.Login_Pwd_Hash = pwdHash.String
	if expire.Valid {
		acc.Expire_At = expire.Time
	}
	return acc, true, nil
}

// Account__Add 新增账号（Create_Time/Update_Time 取当前时间）
func Account__Add(cfg Account_Add_type) (err error) {
	if cfg.Access_Key == "" {
		return fmt.Errorf("Account__Add: Access_Key 不能为空")
	}
	if cfg.Secret_Key == "" {
		return fmt.Errorf("Account__Add: Secret_Key 不能为空")
	}

	now := time.Now()
	query := `
		INSERT INTO account
			(Access_Key, Secret_Key, Login_Pwd_Hash, Enable, Expire_At,
			 Access_Token_TTL, Refresh_Token_TTL, Create_Time, Update_Time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query,
		cfg.Access_Key,
		cfg.Secret_Key,
		sql.NullString{String: cfg.Login_Pwd_Hash, Valid: cfg.Login_Pwd_Hash != ""},
		cfg.Enable,
		sql.NullTime{Time: cfg.Expire_At, Valid: !cfg.Expire_At.IsZero()},
		normalizeTTL(cfg.Access_Token_TTL, Default_Access_Token_TTL),
		normalizeTTL(cfg.Refresh_Token_TTL, Default_Refresh_Token_TTL),
		now, now,
	)
	if err != nil {
		return fmt.Errorf("Account__Add 新增账号失败 | access_key=%s | err=%w", cfg.Access_Key, err)
	}
	return nil
}

// Account__Update_Mutable 按 access_key 更新可变字段（Secret_Key/Login_Pwd_Hash/Enable/Expire_At/时效），刷新 Update_Time
func Account__Update_Mutable(cfg Account_Add_type) (err error) {
	if cfg.Access_Key == "" {
		return fmt.Errorf("Account__Update_Mutable: Access_Key 不能为空")
	}

	query := `
		UPDATE account SET
			Secret_Key = ?, Login_Pwd_Hash = ?, Enable = ?, Expire_At = ?,
			Access_Token_TTL = ?, Refresh_Token_TTL = ?, Update_Time = ?
		WHERE Access_Key = ?
	`
	_, err = DB.Exec(query,
		cfg.Secret_Key,
		sql.NullString{String: cfg.Login_Pwd_Hash, Valid: cfg.Login_Pwd_Hash != ""},
		cfg.Enable,
		sql.NullTime{Time: cfg.Expire_At, Valid: !cfg.Expire_At.IsZero()},
		normalizeTTL(cfg.Access_Token_TTL, Default_Access_Token_TTL),
		normalizeTTL(cfg.Refresh_Token_TTL, Default_Refresh_Token_TTL),
		time.Now(),
		cfg.Access_Key,
	)
	if err != nil {
		return fmt.Errorf("Account__Update_Mutable 更新账号失败 | access_key=%s | err=%w", cfg.Access_Key, err)
	}
	return nil
}

// Account__Sync 按 Access_Key upsert 账号（存在则更新可变字段，不存在则新增），供 internal/sync-aksk 使用
// 返回：新增数量、更新数量、错误
func Account__Sync(cfgs []Account_Add_type) (added int, updated int, err error) {
	for i, cfg := range cfgs {
		if cfg.Access_Key == "" {
			return added, updated, fmt.Errorf("同步账号失败：第%d条 Access_Key 不能为空", i+1)
		}
		if cfg.Secret_Key == "" {
			return added, updated, fmt.Errorf("同步账号失败：第%d条 Secret_Key 不能为空", i+1)
		}

		_, found, qErr := Account__Query_By_Access_Key(cfg.Access_Key)
		if qErr != nil {
			return added, updated, qErr
		}

		if found {
			if uErr := Account__Update_Mutable(cfg); uErr != nil {
				return added, updated, uErr
			}
			updated++
		} else {
			if aErr := Account__Add(cfg); aErr != nil {
				return added, updated, aErr
			}
			added++
		}
	}
	log.Printf("同步账号：新增 %d 条，更新 %d 条", added, updated)
	return added, updated, nil
}

// Account_List_type 账号列表脱敏视图（不含 Secret_Key / Login_Pwd_Hash，仅暴露是否已设登录密码）
type Account_List_type struct {
	Id                uint      // 递增id
	Access_Key        string    // 唯一身份标识（前端登录用户名）
	Enable            int       // 1=启用 0=禁用
	Expire_At         time.Time // 到期时间（零值=永不过期）
	Access_Token_TTL  uint      // access_token 时效（秒）
	Refresh_Token_TTL uint      // refresh_token 时效（秒）
	Has_Login_Pwd     bool      // 是否已设置登录密码（不落库，查询时计算）
	Create_Time       time.Time // 创建时间
	Update_Time       time.Time // 更新时间
}

// Account__Query 分页查询账号列表（脱敏：绝不 SELECT Secret_Key / Login_Pwd_Hash）
// page=0 时返回全部，否则按 (page-1)*pageSize 偏移分页
func Account__Query(page uint, pageSize uint) (list []Account_List_type, err error) {
	baseQuery := `
		SELECT Id, Access_Key, Enable, Expire_At,
			   Access_Token_TTL, Refresh_Token_TTL,
			   (Login_Pwd_Hash IS NOT NULL AND Login_Pwd_Hash <> '') AS Has_Login_Pwd,
			   Create_Time, Update_Time
		FROM account
		ORDER BY Id ASC
	`
	var args []interface{}
	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		return list, fmt.Errorf("ERROR 查询账号列表失败，错误:%w, SQL:%s, 参数:%v", err, baseQuery, args)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			acc    Account_List_type
			expire sql.NullTime
			hasPwd bool
		)
		err = rows.Scan(
			&acc.Id, &acc.Access_Key, &acc.Enable, &expire,
			&acc.Access_Token_TTL, &acc.Refresh_Token_TTL, &hasPwd,
			&acc.Create_Time, &acc.Update_Time,
		)
		if err != nil {
			return list, fmt.Errorf("ERROR 遍历账号列表失败，错误:%w", err)
		}
		acc.Has_Login_Pwd = hasPwd
		if expire.Valid {
			acc.Expire_At = expire.Time
		}
		list = append(list, acc)
	}
	if err = rows.Err(); err != nil {
		return list, fmt.Errorf("ERROR 账号列表结果集异常，错误:%w", err)
	}
	return list, nil
}

// Account__Count 统计账号总数（与 Account__Query 保持一致的分页语义）
func Account__Count(page uint, pageSize uint) (count uint, err error) {
	baseQuery := "SELECT COUNT(`Id`) FROM `account`"
	var args []interface{}
	if page != 0 {
		offset := (page - 1) * pageSize
		baseQuery += " LIMIT ?, ?"
		args = append(args, offset, pageSize)
	}
	err = DB.QueryRow(baseQuery, args...).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		count = 0
	} else if err != nil {
		err = fmt.Errorf("[Account__Count] 查询失败 | SQL=%s | err=%w", baseQuery, err)
		log.Print(err)
	}
	return count, err
}

// Account__Del 按 Id 批量删除账号
func Account__Del(ids ...uint) (err error) {
	if len(ids) == 0 {
		return nil
	}
	for idx, id := range ids {
		if id == 0 {
			return fmt.Errorf("ERROR 第%d条账号ID(Id)不能为空", idx+1)
		}
		if _, execErr := DB.Exec("DELETE FROM `account` WHERE `Id` = ?", id); execErr != nil {
			return fmt.Errorf("ERROR 第%d条账号删除失败, ID:%d, 错误:%w", idx+1, id, execErr)
		}
	}
	return nil
}
