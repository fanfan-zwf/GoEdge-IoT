/*
* 日期: 2026.3.21 PM10:11
* 作者: 范范zwf
* 作用: mysql 检查表结构
 */

package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// ---------------------- 1. 字段级别配置 ----------------------
type ColumnRule struct {
	ColumnName   string // 字段名
	ColumnType   string // 字段类型（如 int(11)、varchar(255)、int unsigned）
	IsAutoInc    bool   // 是否自增（程序预期）
	IsRequired   bool   // 是否非空
	IsPrimaryKey bool   // 是否主键（自增字段必须设为 true）
	IsIndex      bool   // 是否普通索引
	IsUnique     bool   // 是否唯一索引
	DefaultValue string // 默认值规则：
	// - ""        ：无默认值（可空字段→数据库默认 NULL；非空字段→必须传值）
	// - "任意字符串"：字符串默认值（代码自动加单引号）
	// - "0"/"123" ：数值默认值
	// - "CURRENT_TIMESTAMP"：时间默认值
	Comment string // 字段注释
}

// ---------------------- 2. 表级别配置 ----------------------
type TableRule struct {
	TableName    string       // 数据库表名（推荐小写，避免和关键字冲突）
	TableComment string       // 表注释（写入数据库）
	Columns      []ColumnRule // 字段列表
}

// ---------------------- 3. 程序预期的表规则（精简版） ----------------------
// 定义程序预期的表规则（按需修改为你的实际表/字段）
var expectTableRules = []TableRule{
	{
		TableName:    "User",
		TableComment: "存储用户基础信息",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "Name", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "用户名"},
			{ColumnName: "Passwd", ColumnType: "varchar(200)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "密码"},
			{ColumnName: "Permissions", ColumnType: "int", IsAutoInc: false, IsRequired: true, DefaultValue: "1", Comment: "权限"},
			{ColumnName: "Discontinued", ColumnType: "tinyint(1)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "0", Comment: "停用"},
			{ColumnName: "Phone", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: false, IsUnique: true, DefaultValue: "", Comment: "电话"},
			{ColumnName: "Email", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: false, IsUnique: true, DefaultValue: "", Comment: "邮箱"},
			{ColumnName: "Avatar", ColumnType: "varchar(200)", IsAutoInc: false, IsRequired: false, DefaultValue: "", Comment: "头像路径"},
			{ColumnName: "Refresh_Token_bits", ColumnType: "int", IsAutoInc: false, IsRequired: true, DefaultValue: "2048", Comment: "刷新令牌RSA密钥长度"},
			{ColumnName: "Access_Token_bits", ColumnType: "int", IsAutoInc: false, IsRequired: true, DefaultValue: "1024", Comment: "访问令牌RSA密钥长度"},
			{ColumnName: "Refresh_Token_TTL", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, DefaultValue: "86400", Comment: "刷新令牌过期时间s"},
			{ColumnName: "Access_Token_TTL", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, DefaultValue: "1800", Comment: "访问令牌过期时间s"},
		},
	}, {
		TableName:    "User_Terminal",
		TableComment: "用户终端",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "User_Id", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "用户id"},
			{ColumnName: "Terminal_Uuid", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: false, DefaultValue: "", Comment: "终端id"},
			{ColumnName: "Device_Name", ColumnType: "varchar(500)", IsAutoInc: false, IsRequired: false, DefaultValue: "", Comment: "设备名称"},
			{ColumnName: "Ip", ColumnType: "varchar(80)", IsAutoInc: false, IsRequired: false, DefaultValue: "", Comment: "请求的ip"},
			{ColumnName: "Del", ColumnType: "tinyint(1)", IsAutoInc: false, IsRequired: true, DefaultValue: "", Comment: "删除"},
		},
	}, {
		TableName:    "Set",
		TableComment: "设置",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "Type", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "类型"},
			{ColumnName: "Msg", ColumnType: "text", IsAutoInc: false, IsRequired: true, DefaultValue: "", Comment: "终端id"},
		},
	}, {
		TableName:    "Log",
		TableComment: "日志",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "Type", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "login: 登录日志"},
			{ColumnName: "Message", ColumnType: "varchar(255)", IsAutoInc: false, IsRequired: true, DefaultValue: "", Comment: "描述"},
			{ColumnName: "Time", ColumnType: "datetime", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "时间"},
			{ColumnName: "User_Id", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: false, IsUnique: true, DefaultValue: "", Comment: "用户id"},
		},
	}, {
		TableName:    "Group",
		TableComment: "用户组",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "Name", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "'未命名组'", Comment: "组名称"},
			{ColumnName: "Explain", ColumnType: "varchar(255)", IsAutoInc: false, IsRequired: false, DefaultValue: "", Comment: "描述"},
		},
	}, {
		TableName:    "Group_User",
		TableComment: "用户对应的用户组",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "User_Id", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "用户id"},
			{ColumnName: "Group_Id", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "用户组id"},
			{ColumnName: "Administrator", ColumnType: "tinyint(1)", IsAutoInc: false, IsRequired: true, DefaultValue: "", Comment: "是否是管理员"},
		},
	}, {
		TableName:    "Authority",
		TableComment: "权限",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "Name", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "权限名称"},
			{ColumnName: "Theme", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "权限主题"},
			{ColumnName: "Explain", ColumnType: "text", IsAutoInc: false, IsRequired: false, DefaultValue: "", Comment: "说明"},
		},
	}, {
		TableName:    "Authority_User",
		TableComment: "用户对应的权限",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "User_Id", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "Authority_Id", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "Enable", ColumnType: "tinyint(1)", IsAutoInc: false, IsRequired: false, DefaultValue: "1", Comment: "使能"},
		},
	}, {
		TableName:    "Api",
		TableComment: "api接口授权",
		Columns: []ColumnRule{
			{ColumnName: "Id", ColumnType: "int unsigned", IsAutoInc: true, IsRequired: true, IsPrimaryKey: true, IsUnique: true, DefaultValue: "", Comment: ""},
			{ColumnName: "User_Id", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "用户id"},
			{ColumnName: "ApiKey", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "key"},
			{ColumnName: "Secret", ColumnType: "varchar(100)", IsAutoInc: false, IsRequired: true, IsUnique: true, DefaultValue: "", Comment: "秘密"},
			{ColumnName: "Allow_Ip", ColumnType: "tinyint(1)", IsAutoInc: false, IsRequired: false, DefaultValue: "", Comment: "允许的ip"},
			{ColumnName: "Discontinued", ColumnType: "tinyint(1)", IsAutoInc: false, IsRequired: true, DefaultValue: "0", Comment: "停用"},
			{ColumnName: "Refresh_Token_bits", ColumnType: "int", IsAutoInc: false, IsRequired: true, DefaultValue: "2048", Comment: "刷新令牌RSA密钥长度"},
			{ColumnName: "Access_Token_bits", ColumnType: "int", IsAutoInc: false, IsRequired: true, DefaultValue: "1024", Comment: "访问令牌RSA密钥长度"},
			{ColumnName: "Refresh_Token_TTL", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, DefaultValue: "86400", Comment: "刷新令牌过期时间s"},
			{ColumnName: "Access_Token_TTL", ColumnType: "int unsigned", IsAutoInc: false, IsRequired: true, DefaultValue: "1800", Comment: "访问令牌过期时间s"},
		},
	},
}

// ---------------------- 5. 核心工具函数 ----------------------

// 1. 校验自增字段配置（提前拦截MySQL约束错误）
func validateAutoIncConfig(tableRule TableRule) error {
	autoIncCols := make([]string, 0)
	primaryKeyCols := make([]string, 0)

	// 统计自增字段和主键字段
	for _, col := range tableRule.Columns {
		if col.IsAutoInc {
			autoIncCols = append(autoIncCols, col.ColumnName)
		}
		if col.IsPrimaryKey {
			primaryKeyCols = append(primaryKeyCols, col.ColumnName)
		}
	}

	// 规则1：自增字段只能有一个
	if len(autoIncCols) > 1 {
		return fmt.Errorf("表[%s]配置了多个自增字段：%v（MySQL仅允许一个自增字段）", tableRule.TableName, autoIncCols)
	}

	// 规则2：自增字段必须设为主键
	if len(autoIncCols) == 1 && len(primaryKeyCols) == 0 {
		return fmt.Errorf("表[%s]自增字段[%s]未配置为主键（IsPrimaryKey=true）", tableRule.TableName, autoIncCols[0])
	}

	// 规则3：自增字段必须是主键字段
	if len(autoIncCols) == 1 && len(primaryKeyCols) > 0 {
		isAutoIncPrimary := false
		for _, pkCol := range primaryKeyCols {
			if pkCol == autoIncCols[0] {
				isAutoIncPrimary = true
				break
			}
		}
		if !isAutoIncPrimary {
			return fmt.Errorf("表[%s]自增字段[%s]未设为主键，主键字段为：%v", tableRule.TableName, autoIncCols[0], primaryKeyCols)
		}
	}

	return nil
}

// 2. 校验字段类型是否为MySQL合法类型（支持任意varchar(n)）
func validateColumnType(tableName string, columns []ColumnRule) error {
	// 基础合法类型（无需指定长度的类型）
	baseValidTypes := map[string]bool{
		// 数值类型
		"int": true, "int unsigned": true,
		"tinyint": true,
		"bigint":  true,
		// 字符串类型
		"text": true, "longtext": true,
		// 时间类型
		"datetime": true, "timestamp": true,
	}

	// 支持带长度的类型前缀（如 int(11)、varchar(200)）
	validPrefixes := map[string]bool{
		"int(":     true,
		"tinyint(": true,
		"bigint(":  true,
		"varchar(": true,
	}

	for _, col := range columns {
		colType := strings.TrimSpace(col.ColumnType)
		isLegal := false

		// 1. 先检查是否是基础合法类型（无长度）
		if baseValidTypes[colType] {
			isLegal = true
		}

		// 2. 检查是否是带长度的合法类型（如 varchar(200)、int(11)）
		if !isLegal {
			for prefix := range validPrefixes {
				if strings.HasPrefix(colType, prefix) {
					// 验证长度部分是否是合法数字（如 (200) → 200 是数字）
					lenPart := strings.TrimPrefix(colType, prefix)
					lenPart = strings.TrimSuffix(lenPart, ")")
					if isNumeric(lenPart) {
						isLegal = true
						break
					}
				}
			}
		}

		// 3. 特殊处理：int(11) unsigned 这类带修饰符的类型
		if !isLegal && strings.Contains(colType, " ") {
			parts := strings.Split(colType, " ")
			if len(parts) == 2 && parts[1] == "unsigned" {
				baseType := parts[0]
				// 检查基础类型是否合法（如 int(11) 是合法的）
				if baseValidTypes[baseType] {
					isLegal = true
				} else {
					for prefix := range validPrefixes {
						if strings.HasPrefix(baseType, prefix) {
							lenPart := strings.TrimPrefix(baseType, prefix)
							lenPart = strings.TrimSuffix(lenPart, ")")
							if isNumeric(lenPart) {
								isLegal = true
								break
							}
						}
					}
				}
			}
		}

		// 类型非法 → 返回错误
		if !isLegal {
			return fmt.Errorf("表[%s]字段[%s]配置了非法MySQL类型：%s",
				tableName, col.ColumnName, colType)
		}
	}
	return nil
}

// 辅助函数：判断字符串是否为数字
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// 3. MySQL关键字列表（避免表名/字段名冲突）
var mysqlKeywords = map[string]bool{
	"group": true, "explain": true, "set": true, "user": true,
	"select": true, "insert": true, "update": true, "delete": true,
	"from": true, "where": true, "order": true, "by": true,
}

// 校验并处理MySQL关键字（表名/字段名加反引号）
func escapeKeyword(name string) string {
	lowerName := strings.ToLower(name)
	if mysqlKeywords[lowerName] {
		return fmt.Sprintf("`%s`", name)
	}
	return name
}

// 4. 自动为字符串默认值添加单引号（避免语法错误）
func formatDefaultValue(col ColumnRule) string {
	if col.DefaultValue == "" || col.DefaultValue == "NULL" {
		return col.DefaultValue
	}

	// 已经带单引号的，直接返回
	if strings.HasPrefix(col.DefaultValue, "'") && strings.HasSuffix(col.DefaultValue, "'") {
		return col.DefaultValue
	}

	// 数值/时间类型（如 0、CURRENT_TIMESTAMP），直接返回
	if isNumeric(col.DefaultValue) || strings.ToUpper(col.DefaultValue) == "CURRENT_TIMESTAMP" {
		return col.DefaultValue
	}

	// 字符串类型，自动加单引号
	return fmt.Sprintf("'%s'", col.DefaultValue)
}

// 5. 读取表的实际字段信息
func getTableColumns(tableName string) (map[string]ColumnRule, error) {
	query := `
		SELECT COLUMN_NAME, COLUMN_TYPE, EXTRA, IS_NULLABLE, COLUMN_DEFAULT, COLUMN_COMMENT, COLUMN_KEY
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
	`
	rows, err := DB.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("读取表[%s]字段失败：%w", tableName, err)
	}
	defer rows.Close()

	actualColumns := make(map[string]ColumnRule)
	for rows.Next() {
		var (
			colName    string
			colType    string
			extra      string
			isNullable string
			colDefault sql.NullString // 数据库默认值（NULL时Valid=false）
			colComment sql.NullString
			columnKey  string // 判断是否为主键（PRI）
		)
		if err := rows.Scan(&colName, &colType, &extra, &isNullable, &colDefault, &colComment, &columnKey); err != nil {
			return nil, fmt.Errorf("扫描字段失败：%w", err)
		}

		// 解析默认值（核心修复：兼容NULL/空字符串/无默认值）
		var defaultValue string
		if !colDefault.Valid {
			// 数据库COLUMN_DEFAULT为NULL → 标记为"NULL"
			defaultValue = "NULL"
		} else {
			// 数据库有默认值 → 直接映射
			defaultValue = colDefault.String
			// 特殊处理：空字符串默认值（数据库中是''，程序中映射为'\'\''）
			if defaultValue == "" {
				defaultValue = "'''"
			}
		}

		// 解析注释
		comment := ""
		if colComment.Valid {
			comment = colComment.String
		}

		actualColumns[colName] = ColumnRule{
			ColumnName:   colName,
			ColumnType:   colType,
			IsAutoInc:    extra == "auto_increment",
			IsRequired:   isNullable == "NO",
			IsPrimaryKey: columnKey == "PRI", // 数据库中是否为主键
			DefaultValue: defaultValue,
			Comment:      comment,
		}
	}
	return actualColumns, nil
}

// 6. 读取表的实际注释
func getTableComment(tableName string) (string, error) {
	query := `
		SELECT TABLE_COMMENT FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
	`
	var comment sql.NullString
	err := DB.QueryRow(query, tableName).Scan(&comment)
	if err != nil {
		return "", fmt.Errorf("读取表[%s]注释失败：%w", tableName, err)
	}
	if comment.Valid {
		return comment.String, nil
	}
	return "", nil
}

// 7. 自动创建表（适配所有约束）
func createTable(tableRule TableRule) error {
	// 提前校验自增配置
	if err := validateAutoIncConfig(tableRule); err != nil {
		return err
	}

	var colDefs []string
	var indexDefs []string   // 存储索引定义
	var primaryKeyDef string // 主键定义（统一拼接，更规范）

	// 拼接字段定义
	for _, col := range tableRule.Columns {
		// 字段名避关键字
		colName := escapeKeyword(col.ColumnName)
		colDef := fmt.Sprintf("%s %s", colName, col.ColumnType)

		// 自增属性
		if col.IsAutoInc {
			colDef += " AUTO_INCREMENT"
		}
		// 非空属性
		if col.IsRequired {
			colDef += " NOT NULL"
		} else {
			colDef += " NULL"
		}

		// 默认值（自动加引号）
		if col.DefaultValue != "" {
			formattedDefault := formatDefaultValue(col)
			colDef += fmt.Sprintf(" DEFAULT %s", formattedDefault)
		}

		// 字段注释
		if col.Comment != "" {
			colDef += fmt.Sprintf(" COMMENT '%s'", escapeSingleQuote(col.Comment))
		}

		// 主键定义（单独拼接）
		if col.IsPrimaryKey {
			primaryKeyDef = fmt.Sprintf("PRIMARY KEY (%s)", colName)
		}

		// 索引定义（单独拼接）
		if col.IsIndex && !col.IsPrimaryKey && !col.IsUnique {
			// 普通索引：KEY idx_colname (colname)
			indexName := fmt.Sprintf("idx_%s", strings.ToLower(col.ColumnName))
			indexDefs = append(indexDefs, fmt.Sprintf("KEY `%s` (%s)", indexName, colName))
		}

		// 唯一索引定义（单独拼接）
		if col.IsUnique && !col.IsPrimaryKey {
			// 唯一索引：UNIQUE KEY uk_colname (colname)
			indexName := fmt.Sprintf("uk_%s", strings.ToLower(col.ColumnName))
			indexDefs = append(indexDefs, fmt.Sprintf("UNIQUE KEY `%s` (%s)", indexName, colName))
		}

		colDefs = append(colDefs, colDef)
	}

	// 添加主键定义（如果有）
	if primaryKeyDef != "" {
		colDefs = append(colDefs, primaryKeyDef)
	}

	// 添加索引定义（如果有）
	if len(indexDefs) > 0 {
		colDefs = append(colDefs, indexDefs...)
	}

	// 表名避关键字
	escapedTableName := escapeKeyword(tableRule.TableName)
	// 拼接创建表 SQL
	createSQL := fmt.Sprintf(
		"CREATE TABLE %s (%s) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT '%s'",
		escapedTableName,
		strings.Join(colDefs, ", "),
		escapeSingleQuote(tableRule.TableComment),
	)

	_, err := DB.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("创建表 [%s] 失败，SQL：%s，错误：%w", tableRule.TableName, createSQL, err)
	}
	log.Printf("表 [%s] 创建成功，表注释：%s", tableRule.TableName, tableRule.TableComment)
	return nil
}

// 8. 自动新增缺失的字段
func addMissingColumn(tableName string, col ColumnRule) error {
	// 禁止为已存在的表新增自增字段（MySQL约束）
	if col.IsAutoInc {
		return fmt.Errorf("禁止为已存在的表[%s]新增自增字段[%s]（需表创建时定义）", tableName, col.ColumnName)
	}

	// 字段名避关键字
	colName := escapeKeyword(col.ColumnName)
	colDef := fmt.Sprintf("%s %s", colName, col.ColumnType)

	// 非空属性
	if col.IsRequired {
		colDef += " NOT NULL"
	} else {
		colDef += " NULL"
	}

	// 默认值（自动加引号）
	if col.DefaultValue != "" {
		formattedDefault := formatDefaultValue(col)
		colDef += fmt.Sprintf(" DEFAULT %s", formattedDefault)
	}

	// 字段注释
	if col.Comment != "" {
		colDef += fmt.Sprintf(" COMMENT '%s'", escapeSingleQuote(col.Comment))
	}

	// 禁止新增字段时设为主键
	if col.IsPrimaryKey {
		return fmt.Errorf("新增字段[%s.%s]时禁止设置主键（需手动修改表结构）", tableName, col.ColumnName)
	}

	// 表名避关键字
	escapedTableName := escapeKeyword(tableName)
	addSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", escapedTableName, colDef)
	_, err := DB.Exec(addSQL)
	if err != nil {
		return fmt.Errorf("新增字段[%s.%s]失败，SQL：%s，错误：%w", tableName, col.ColumnName, addSQL, err)
	}
	log.Printf("表[%s]新增字段[%s]成功", tableName, col.ColumnName)
	return nil
}

// 9. 修改表注释
func alterTableComment(tableName, expectComment string) error {
	// 表名避关键字
	escapedTableName := escapeKeyword(tableName)
	alterSQL := fmt.Sprintf(
		"ALTER TABLE %s COMMENT = '%s'",
		escapedTableName,
		escapeSingleQuote(expectComment),
	)
	_, err := DB.Exec(alterSQL)
	if err != nil {
		return fmt.Errorf("修改表[%s]注释失败：%w", tableName, err)
	}
	log.Printf("表[%s]注释已更新为：%s", tableName, expectComment)
	return nil
}

// 10. 转义单引号（避免注释/默认值中的单引号导致SQL错误）
func escapeSingleQuote(str string) string {
	return strings.ReplaceAll(str, "'", "''")
}

// ---------------------- 6. 核心校验逻辑（最终版） ----------------------
func CheckAndFixTableStructure(tableRule TableRule) {
	tableName := tableRule.TableName

	// 前置校验：自增配置 + 字段类型
	if err := validateAutoIncConfig(tableRule); err != nil {
		panic(fmt.Sprintf("表[%s]自增配置非法：%v", tableName, err))
	}
	if err := validateColumnType(tableName, tableRule.Columns); err != nil {
		panic(fmt.Sprintf("表[%s]字段类型配置非法：%v", tableName, err))
	}

	// 1. 检查表是否存在，不存在则创建
	exists, err := checkTableExists(tableName)
	if err != nil {
		panic(fmt.Sprintf("检查表[%s]存在性异常：%v", tableName, err))
	}
	if !exists {
		if err := createTable(tableRule); err != nil {
			panic(fmt.Sprintf("创建表[%s]失败：%v", tableName, err))
		}
		return
	}

	// 2. 校验表注释
	actualTableComment, err := getTableComment(tableName)
	if err != nil {
		panic(fmt.Sprintf("读取表[%s]注释失败：%v", tableName, err))
	}
	if actualTableComment != tableRule.TableComment {
		if err := alterTableComment(tableName, tableRule.TableComment); err != nil {
			panic(fmt.Sprintf("修改表[%s]注释失败：%v", tableName, err))
		}
	}

	// 3. 读取实际字段信息
	actualCols, err := getTableColumns(tableName)
	if err != nil {
		panic(fmt.Sprintf("读取表[%s]字段异常：%v", tableName, err))
	}

	// 4. 遍历预期字段，校验+修复
	for _, expectCol := range tableRule.Columns {
		actualCol, ok := actualCols[expectCol.ColumnName]

		// 字段不存在 → 新增
		if !ok {
			if err := addMissingColumn(tableName, expectCol); err != nil {
				panic(fmt.Sprintf("表[%s]新增字段[%s]失败：%v", tableName, expectCol.ColumnName, err))
			}
			continue
		}

		// 字段存在 → 校验属性
		var errMsg []string

		// 类型校验
		if actualCol.ColumnType != expectCol.ColumnType {
			errMsg = append(errMsg, fmt.Sprintf("类型不匹配（预期：%s，实际：%s）", expectCol.ColumnType, actualCol.ColumnType))
		}
		// 自增校验
		if actualCol.IsAutoInc != expectCol.IsAutoInc {
			errMsg = append(errMsg, fmt.Sprintf("自增属性不匹配（预期：%t，实际：%t）", expectCol.IsAutoInc, actualCol.IsAutoInc))
		}
		// 非空校验
		if actualCol.IsRequired != expectCol.IsRequired {
			errMsg = append(errMsg, fmt.Sprintf("非空属性不匹配（预期：%t，实际：%t）", expectCol.IsRequired, actualCol.IsRequired))
		}
		// 主键校验
		if actualCol.IsPrimaryKey != expectCol.IsPrimaryKey {
			errMsg = append(errMsg, fmt.Sprintf("主键属性不匹配（预期：%t，实际：%t）", expectCol.IsPrimaryKey, actualCol.IsPrimaryKey))
		}

		// 核心修复：默认值校验（新增自增字段豁免逻辑）
		var isDefaultMatch bool
		// 特殊豁免：自增字段（无论配置如何，默认值都判定为匹配）
		if expectCol.IsAutoInc {
			isDefaultMatch = true
		} else {
			// 普通字段的默认值校验逻辑
			switch {
			// 场景 1：预期无默认值 (DefaultValue="")
			case expectCol.DefaultValue == "":
				// 子场景 1.1：可空字段 → 实际默认值为 NULL 则匹配
				if !expectCol.IsRequired && actualCol.DefaultValue == "NULL" {
					isDefaultMatch = true
				}
				// 子场景 1.2：非空字段 → 实际无默认值（"" 或 "NULL"）均视为匹配
				// 解释：MySQL 中非空无默认值字段，COLUMN_DEFAULT 可能为 NULL 字符串，代表必须传值
				if expectCol.IsRequired && (actualCol.DefaultValue == "" || actualCol.DefaultValue == "NULL") {
					isDefaultMatch = true
				}
			// 场景 2：预期有默认值 → 严格匹配
			default:
				isDefaultMatch = (actualCol.DefaultValue == expectCol.DefaultValue)
			}
		}
		if !isDefaultMatch {
			errMsg = append(errMsg, fmt.Sprintf("默认值不匹配（预期：%s，实际：%s）", expectCol.DefaultValue, actualCol.DefaultValue))
		}

		// 注释校验
		if actualCol.Comment != expectCol.Comment {
			errMsg = append(errMsg, fmt.Sprintf("注释不匹配（预期：%s，实际：%s）", expectCol.Comment, actualCol.Comment))
		}

		// 属性不匹配 → Panic
		if len(errMsg) > 0 {
			panic(fmt.Sprintf("表[%s]字段[%s]属性不匹配：%s", tableName, expectCol.ColumnName, strings.Join(errMsg, "；")))
		}
	}

	log.Printf("表[%s]结构完全符合预期", tableName)
}

// 辅助函数：检查表是否存在
func checkTableExists(tableName string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
	`
	var count int
	err := DB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查表[%s]存在性失败：%w", tableName, err)
	}
	return count > 0, nil
}

// ---------------------- 7. 主函数 ----------------------
func CheckSqlStructure() {
	// 遍历所有表规则，执行校验+修复
	for _, tableRule := range expectTableRules {
		log.Printf("\n========== 开始处理表：%s ==========", tableRule.TableName)
		CheckAndFixTableStructure(tableRule)
	}
}
