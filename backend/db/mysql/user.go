/*
* 日期: 2026.3.5 PM11:18
* 作者: 范范zwf
* 作用: mysql 用户逻辑
 */

package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"main/Init"
	"regexp"
	"slices"
	"strings"
	"time"
)

/*
***************用户***************
 */
// 登陆结构体
type User__table_type struct {
	Id                 uint
	Name               string // 用户名
	Permissions        uint   // 权限
	Refresh_Token_Time uint   // 过期时间设定（s）
	Discontinued       bool   // 停用
}

type User__all_table_type struct {
	User__table_type
	Passwd string // 密码
}

// 创建一个映射表，将字符串名称与常量值关联起来
var User__Info_Search_Type = []string{"Name", "Phone", "Email"}

// 通过用户名密码查询基础配置
func User__NamePasswd_Query(User_Name string, User_Passwd string) (User User__table_type, err error) {
	if len(User_Name) == 0 || len(User_Passwd) == 0 {
		err = fmt.Errorf("用户名或密码为空")
		log.Print(err.Error())
		return
	}

	query := `
	SELECT
		Id,
		Name,
		Permissions,
		Refresh_Token_Time,
		Discontinued
	FROM
		User
	WHERE
		Name = ? AND
		Passwd = ?
	`

	err = DB.QueryRow(query, User_Name, User_Passwd).Scan(
		&User.Id,
		&User.Name,
		&User.Permissions,
		&User.Refresh_Token_Time,
		&User.Discontinued,
	)
	if err != nil {
		log.Print(err.Error())
	}
	return
}

// 查询用户信息
func User__Info_Query(User_Id uint) (User User__table_type, err error) {
	if User_Id == 0 {
		err = fmt.Errorf("User_Id不能是0")
		log.Print(err.Error())
		return
	}

	query := `
	SELECT
		Id,
		Name,
		Permissions,
		Refresh_Token_Time,
		Discontinued
	FROM
		User
	WHERE
		Id = ?
	`
	err = DB.QueryRow(query, User_Id).Scan(
		&User.Id,
		&User.Name,
		&User.Permissions,
		&User.Refresh_Token_Time,
		&User.Discontinued,
	)
	if err != nil {
		log.Print(err.Error())
	}
	return
}

// 查询多个用户信息
func User__Info_Array_Query(User_Id_array []uint) (User_array []User__table_type, err error) {
	User_Id_array_len := len(User_Id_array)
	if User_Id_array_len == 0 {
		err = fmt.Errorf("User_Id_array为空")
		return
	}
	if User_Id_array_len > 1000 {
		err = fmt.Errorf("User_Id_array过长")
		return
	}

	// 构建占位符和参数
	placeholders := make([]string, User_Id_array_len)
	args := make([]interface{}, User_Id_array_len)

	for i, id := range User_Id_array {
		if id == 0 {
			err = fmt.Errorf("User_Id不能是0")
			log.Print(err.Error())
			return
		}
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
	SELECT
		Id,
		Name,
		Permissions,
		Refresh_Token_Time,
		Discontinued
	FROM
		User
	WHERE
		Id IN (%s)
	`, strings.Join(placeholders, ","))

	var (
		rows *sql.Rows
	)
	rows, err = DB.Query(query, args...)
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			User User__table_type
		)

		err = rows.Scan(
			&User.Id,
			&User.Name,
			&User.Permissions,
			&User.Refresh_Token_Time,
			&User.Discontinued,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}
		User_array = append(User_array, User)
	}

	return
}

// 查询多个用户信息 传递:搜索值，搜索方式，数量
func User__Info_Array_Search(Search string, Type string, Number uint) (User_array []User__table_type, err error) {
	if Type != "" && !slices.Contains(User__Info_Search_Type, Type) {
		err = fmt.Errorf("搜索类型不存在")
		return
	} else if Type == "" {
		Type = "Name"
	}

	if Number == 0 {
		Number = 10
	} else if Number > 1000 {
		err = fmt.Errorf("搜索数据太多了")
		return
	}

	query := `
		SELECT 
			Id,
			Name,
			Permissions,
			Refresh_Token_Time,
			Discontinued
		FROM
			User 
		WHERE ? LIKE ? LIMIT ? 
		`

	var (
		rows *sql.Rows
	)
	rows, err = DB.Query(query, Type, Search, Number)
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var User User__table_type
		err = rows.Scan(
			&User.Id,
			&User.Name,
			&User.Permissions,
			&User.Refresh_Token_Time,
			&User.Discontinued,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}
		User_array = append(User_array, User)
	}

	return
}

// 修改名称
func User__Name_Update(User_Id uint, User_Name string) (err error) {
	if User_Id == 0 || len(User_Name) == 0 {
		err = fmt.Errorf("User_Id不能是0 或 User_Name长度是0")
		log.Print(err.Error())
		return
	}

	// 正则表达计算->用户名
	var matched bool
	matched, err = regexp.MatchString(Init.Regex_Name, User_Name)
	if err != nil {
		err = fmt.Errorf("用户名正则表达式计算错误")
		log.Print(err.Error())
		return
	}
	if !matched {
		err = fmt.Errorf("输入不合法")
		log.Print(err.Error())
		return
	}

	query := `
	UPDATE User
		3
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, User_Name, User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 修改密码
func User__Passwd_Update(User_Id uint, User_Passwd string) (err error) {
	if User_Id == 0 || len(User_Passwd) == 0 {
		err = fmt.Errorf("User_Id不能是0 或 User_Passwd长度是0")
		log.Print(err.Error())
		return
	}

	query := `
	UPDATE User
		SET Passwd = CAST(? AS CHAR)
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, User_Passwd, User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 修改停用
func User__Discontinued_Update(User_Id uint, Discontinued bool) (err error) {
	if User_Id == 0 {
		err = fmt.Errorf("User_Id不应该是0")
		return
	}
	query := `
	UPDATE User
		SET Discontinued = ? 
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Discontinued, User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 增加用户
func User__Add(value User__all_table_type) (Id uint, err error) {
	if value.Id == 0 {
		err = fmt.Errorf("Id不能对于0")
		return
	}

	if len(value.Name) == 0 || len(value.Passwd) == 0 {
		err = fmt.Errorf("User_Name长度是0 或 User_Passwd长度是0")
		log.Print(err.Error())
		return
	}

	if value.Refresh_Token_Time == 0 {
		value.Refresh_Token_Time = 604800
	}

	query := `
	INSERT
		INTO
		User(Name, Passwd, Permissions, Refresh_Token_Time, Discontinued) 
		VALUES(?,?,?,?,?)
	`
	// 修改数据库
	var result sql.Result
	result, err = DB.Exec(query,
		value.Name,
		value.Passwd,
		value.Permissions,
		value.Refresh_Token_Time,
		value.Discontinued)
	if err != nil {
		log.Print(err.Error())
		return
	}

	// 影响的id
	var LastInsertId int64
	LastInsertId, err = result.LastInsertId()
	if err != nil {
		log.Print(err.Error())
	}
	Id = uint(LastInsertId)

	return
}

// 删除用户
func User__Del(User_Id uint) (err error) {
	if User_Id == 0 {
		err = fmt.Errorf("User_Id不能是0")
		log.Print(err.Error())
		return
	}

	query := `
	DELETE
	FROM
		User
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 用户刷新令牌有效时间 effective:true的的代表有效
func User__Refresh_Token_Effective(Expires_in time.Time) (effective bool, err error) {
	now := time.Now()
	effective = !now.After(Expires_in)
	return
}

// 查询用户权限
func User__Permissions_Query(User_Id uint) (Permissions uint, err error) {
	if User_Id == 0 {
		err = fmt.Errorf("User_Id不能是0")
		log.Print(err.Error())
		return
	}

	query := `
	SELECT
		Permissions
	FROM
		User
	WHERE
		Id = ?
	`
	err = DB.QueryRow(query, User_Id).Scan(
		&Permissions,
	)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询用户条数
func User__All_Count() (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		User
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询用户 Page页码(0代表全部) Page_Size每页条数
func User__All_Query(Page uint, Page_Size uint) (User_Array []User__table_type, err error) {
	if Page_Size > 5000 {
		err = fmt.Errorf("每页数量不能超过5k")
		log.Print(err.Error())
		return
	}

	query := "SELECT `Id`, `Name`, `Permissions`, `Refresh_Token_Time`, `Discontinued` FROM `User` ORDER BY `Id` DESC "

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ?,? "
		Page -= 1
		rows, err = DB.Query(query, Page, Page_Size)
	} else {
		rows, err = DB.Query(query)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			User User__table_type
		)
		err = rows.Scan(
			&User.Id,
			&User.Name,               // 用户名
			&User.Permissions,        // 权限
			&User.Refresh_Token_Time, // 过期时间设定（s）
			&User.Discontinued,       // 停用
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		User_Array = append(User_Array, User)
	}

	return
}

/*
***************用户终端***************
 */
// 用户终端结构体
type User_Terminal__table_type struct {
	Id            uint
	User_Id       uint   // 用户id
	Terminal_Uuid string // 终端id
	Device_Name   string // 设备名称
	Ip            string // 登陆ip
}

// 增加用户终端
func User_Terminal__Add(User_Terminal User_Terminal__table_type) (err error) {
	query := `
	INSERT
		INTO
		User_Terminal(User_Id, Terminal_Uuid, Device_Name, Ip)
	VALUES(?,?,?,?)
	`
	// 修改数据库
	_, err = DB.Exec(query,
		User_Terminal.User_Id,
		sql.NullString{
			String: User_Terminal.Terminal_Uuid,
			Valid:  User_Terminal.Terminal_Uuid != "",
		},
		sql.NullString{
			String: User_Terminal.Device_Name,
			Valid:  User_Terminal.Device_Name != "",
		},
		sql.NullString{
			String: User_Terminal.Ip,
			Valid:  User_Terminal.Ip != "",
		},
	)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 查询刷新令牌
func User_Terminal__AccessToken_Query(User_Id uint, Access_Token string) (User_Terminal User_Terminal__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Terminal_Uuid,
		Device_Name,
		Ip,
		Refresh_Token,
		Refresh_Token_Expires_in
	FROM
		User_Terminal
	WHERE
	    Del = 0 AND
		User_Id = ? AND
		Access_Token = ?
	LIMIT 1
	`
	var (
		Terminal_Uuid, Device_Name, Ip sql.NullString
	)
	// 修改数据库
	err = DB.QueryRow(query, User_Id, Access_Token).Scan(
		&User_Terminal.Id,
		&User_Terminal.User_Id,
		&Terminal_Uuid,
		&Device_Name,
		&Ip,
	)
	if err != nil {
		log.Print(err.Error())
	}
	User_Terminal.Terminal_Uuid = Terminal_Uuid.String
	User_Terminal.Device_Name = Device_Name.String
	User_Terminal.Ip = Ip.String

	return
}

// 删除
func User_Terminal__Del(User_Terminal_Id uint) (err error) {
	query := `
	DELETE
	FROM
		User_Terminal
	WHERE
		Id = ?
	`
	// 修改数据库
	// 修改数据库
	_, err = DB.Exec(query, User_Terminal_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

/*
***************日志***************
 */
// 日志表结构体
type Log__table_type struct {
	Id      uint
	User_Id uint      // 用户id
	Type    string    // 类型
	Message string    // 描述
	Time    time.Time // 时间
}

// 分页查询日志条数
func Log__All_Count() (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Log
	ORDER BY Time DESC
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询日志 Page页码(0代表全部) Page_Size每页条数
func Log__All_Query(Page uint, Page_Size uint) (Log_Array []Log__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Type,
		Message,
		Time
	FROM
		Log
	ORDER BY Time DESC
	`

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ?,? "
		Page -= 1
		rows, err = DB.Query(query, Page, Page_Size)
	} else {
		rows, err = DB.Query(query)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			Log_table Log__table_type
			User_Id   sql.NullInt64
		)

		err = rows.Scan(
			&Log_table.Id,
			&User_Id,
			&Log_table.Type,
			&Log_table.Message,
			&Log_table.Time,
		)

		if err != nil {
			log.Print(err.Error())
			return
		}

		Log_table.User_Id = uint(User_Id.Int64)

		Log_Array = append(Log_Array, Log_table)
	}

	return
}

// 增加日志
func Log__Add(Value Log__table_type) (Id uint, err error) {
	if Value.Id != 0 {
		err = fmt.Errorf("增加日志,Id需要为空")
		return
	}
	query := `
	INSERT
		INTO
		Log(User_Id, Type, Message, Time)
	VALUES(?,?,?,?)
	`
	// 修改数据库
	var result sql.Result
	result, err = DB.Exec(query,
		sql.NullInt64{
			Int64: int64(Value.User_Id),
			Valid: Value.User_Id != 0,
		},
		Value.Type, Value.Message, Value.Time)
	if err != nil {
		log.Print(err.Error())
		return
	}

	// 影响的id
	var LastInsertId int64
	LastInsertId, err = result.LastInsertId()
	if err != nil {
		log.Print(err.Error())
	}
	Id = uint(LastInsertId)

	return
}

// 增加日志
func Log__Add2(User_Id uint, Type string, Message string) (Id uint, err error) {
	query := `
	INSERT
		INTO
		Log(User_Id, Type, Message, Time)
	VALUES(?,?,?,?)
	`
	// 修改数据库
	var result sql.Result
	result, err = DB.Exec(query, sql.NullInt64{
		Int64: int64(User_Id),
		Valid: User_Id != 0,
	}, Type, Message, time.Now())
	if err != nil {
		log.Print(err.Error())
		return
	}

	// 影响的id
	var LastInsertId int64
	LastInsertId, err = result.LastInsertId()
	if err != nil {
		log.Print(err.Error())
	}
	Id = uint(LastInsertId)

	return
}

// 查询用户Id日志总条数
func Log__User_Count(User_Id uint) (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Log
	WHERE
		User_Id = ?
	ORDER BY Time DESC
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query, User_Id).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 查询全部日志 Page页码(0代表全部) Page_Size每页条数
func Log__User_Query(User_Id uint, Page uint, Page_Size uint) (Log_Array []Log__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Type,
		Message,
		Time
	FROM
		Log
	WHERE
		User_Id = ?
	ORDER BY Time DESC
	`

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ?,? "
		Page -= 1
		rows, err = DB.Query(query, User_Id, Page, Page_Size)
	} else {
		rows, err = DB.Query(query, User_Id)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var Log_table Log__table_type
		err = rows.Scan(
			&Log_table.Id,
			&Log_table.User_Id,
			&Log_table.Type,
			&Log_table.Message,
			&Log_table.Time,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Log_Array = append(Log_Array, Log_table)
	}

	return
}

/*
***************菜单***************
 */
// 菜单表结构体
type Set__table_type struct {
	Id   uint
	Type uint   // 类型
	Msg  string // 用户组id
}

// 查询全部菜单 Page页码(0代表全部) Page_Size每页条数
func Set__All_Query() (Set_Array []Set__table_type, err error) {
	query := `
	SELECT
		Id,
		Type,
		Json
	FROM
		Set 
	`

	var (
		rows *sql.Rows
	)
	rows, err = DB.Query(query)
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var Set Set__table_type
		err = rows.Scan(
			&Set.Id,
			&Set.Type,
			&Set.Msg,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Set_Array = append(Set_Array, Set)
	}

	return
}

// 查询指定类型
func Set_Type_Query(Type string) (value string, err error) {
	query := " SELECT `Msg` FROM `Set` WHERE `Type` = ? "
	err = DB.QueryRow(query, Type).Scan(&value)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 菜单增加
func Set__Add(Value Set__table_type) (err error) {
	if Value.Id != 0 {
		err = fmt.Errorf("菜单增加,Id需要为空")
		return
	}

	query := `
	INSERT
		INTO
		Set(Type, Json)
	VALUES(?,?,?)
	`
	// 修改数据库
	_, err = DB.Exec(query, Value.Type, Value.Msg)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 菜单删除
func Set__Del(Set_Id uint) (err error) {
	query := `
	DELETE
	FROM
		Set
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Set_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}
