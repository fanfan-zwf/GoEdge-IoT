/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: mysql 用户逻辑
 */

package mysql

import (
	"main/Init"
	"regexp"
	"strings"

	"database/sql"
	"fmt"
	"log"
	"slices"

	"time"
)

func init() {
	Init.Log__Add2 = Log__Add2
}

/*
***************用户***************
 */
// 登陆结构体
type User__table_type struct {
	Id                 uint
	Name               string // 用户名
	Avatar             string // 头像url地址
	Permissions        uint   // 权限
	Refresh_Token_Time uint   // 过期时间设定（s）
	Discontinued       bool   // 停用
	Phone              string // 电话
	Email              string // 邮箱
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

	var (
		Phone sql.NullString
		Email sql.NullString
	)
	query := `
	SELECT
		Id,
		Name,
		Permissions,
		Refresh_Token_Time,
		Phone,
		Email,
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
		&Phone,
		&Email,
		&User.Discontinued,
	)
	if err != nil {
		log.Print(err.Error())
	}
	User.Phone = Phone.String
	User.Email = Email.String
	return
}

// 查询用户信息
func User__Info_Query(User_Id uint) (User User__table_type, err error) {
	if User_Id == 0 {
		err = fmt.Errorf("User_Id不能是0")
		log.Print(err.Error())
		return
	}

	var (
		Phone  sql.NullString
		Email  sql.NullString
		Avatar sql.NullString
	)

	query := `
	SELECT
		Id,
		Name,
		Permissions,
		Refresh_Token_Time,
		Discontinued,
		Phone,
		Email,
		Avatar
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
		&Phone,
		&Email,
		&Avatar,
	)
	if err != nil {
		log.Print(err.Error())
	}

	User.Phone = Phone.String
	User.Email = Email.String
	User.Avatar = Avatar.String

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
		Discontinued,
		Phone,
		Email,
		Avatar
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
			User   User__table_type
			Phone  sql.NullString
			Email  sql.NullString
			Avatar sql.NullString
		)

		err = rows.Scan(
			&User.Id,
			&User.Name,
			&User.Permissions,
			&User.Refresh_Token_Time,
			&User.Discontinued,
			&Phone,
			&Email,
			&Avatar,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}
		User.Phone = Phone.String
		User.Email = Email.String
		User.Avatar = Avatar.String
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
			Avatar,
			Permissions,
			Refresh_Token_Time,
			Discontinued,
			Phone,
			Email
		FROM
			User 
		WHERE ? LIKE ? LIMIT ? 
		`

	var (
		rows   *sql.Rows
		Phone  sql.NullString
		Email  sql.NullString
		Avatar sql.NullString
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
			&Avatar,
			&User.Permissions,
			&User.Refresh_Token_Time,
			&User.Discontinued,
			&Phone,
			&Email,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}
		User.Phone = Phone.String
		User.Email = Email.String
		User.Avatar = Avatar.String
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
		SET Name = CAST(? AS CHAR)
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

// 修改电话
func User__Phone_Update(User_Id uint, User_Phone string) (err error) {
	if User_Id == 0 {
		err = fmt.Errorf("User_Id不能是0  ")
		log.Print(err.Error())
		return
	}
	if User_Phone != "" {
		// 正则表达计算->电话
		var matched bool
		matched, err = regexp.MatchString(Init.Regex_Phone, User_Phone)
		if err != nil {
			err = fmt.Errorf("电话正则表达式计算错误")
			log.Print(err.Error())
			return
		}
		if !matched {
			err = fmt.Errorf("输入不合法")
			log.Print(err.Error())
			return
		}
	}

	query := `
	UPDATE User
		SET Phone = ?
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, sql.NullString{
		String: User_Phone,       // 空字符串
		Valid:  User_Phone != "", // 表示是 NULL
	}, User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 修改邮箱
func User__Email_Update(User_Id uint, Email string) (err error) {
	if User_Id == 0 {
		err = fmt.Errorf("User_Id不能是0 ")
		log.Print(err.Error())
		return
	}

	if Email != "" {
		// 正则表达计算->用户名
		var matched bool
		matched, err = regexp.MatchString(Init.Regex_Email, Email)
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
	}

	query := `
	UPDATE User
		SET Email = ?
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, sql.NullString{
		String: Email,       // 空字符串
		Valid:  Email != "", // 表示是 NULL
	}, User_Id)
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

// 头像url地址
func User__Avatar_Update(User_Id uint, Avatar string) (err error) {
	defer func(err error) {
		if err != nil {
			log.Print(err.Error())
		}
	}(err)

	if User_Id == 0 {
		err = fmt.Errorf("User_Id不应该是0")
		return
	}

	if Avatar != "" {
		// 正则表达式计算
		var matched bool
		matched, err = regexp.MatchString(Init.Regex_URL, Avatar)
		if err != nil {
			err = fmt.Errorf("用户名正则表达式计算错误")
			return
		}
		if !matched {
			err = fmt.Errorf("输入不合法")
			return
		}
	}

	query := `
	UPDATE User
		SET Avatar = ? 
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Avatar, User_Id)
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
		User(Name, Passwd, Permissions, Refresh_Token_Time, Discontinued, Phone, Email) 
		VALUES(?,?,?,?,?,?,?)
	`
	// 修改数据库
	var result sql.Result
	result, err = DB.Exec(query,
		value.Name,
		value.Passwd,
		value.Permissions,
		value.Refresh_Token_Time,
		value.Discontinued,
		sql.NullString{
			String: value.Phone,       // 空字符串
			Valid:  value.Phone != "", // 表示是 NULL
		},
		sql.NullString{
			String: value.Email,       // 空字符串
			Valid:  value.Email != "", // 表示是 NULL
		})
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

	query := "SELECT `Id`, `Name`, `Avatar`, `Permissions`, `Refresh_Token_Time`, `Discontinued`, `Phone`, `Email` FROM `User` ORDER BY `Id` DESC "

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
			User   User__table_type
			Phone  sql.NullString
			Email  sql.NullString
			Avatar sql.NullString
		)
		err = rows.Scan(
			&User.Id,
			&User.Name,               // 用户名
			&Avatar,                  // 头像url地址
			&User.Permissions,        // 权限
			&User.Refresh_Token_Time, // 过期时间设定（s）
			&User.Discontinued,       // 停用
			&Phone,                   // 电话
			&Email,                   // 邮箱
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		User.Phone = Phone.String
		User.Email = Email.String
		User.Avatar = Avatar.String

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
***************权限***************
 */
// 权限表结构体
type Authority__table_type struct {
	Id      uint
	Name    string // 权限名称
	Theme   string // 权限主题
	Explain string // 说明
}

var Authority__Search_Type = []string{"Name", "Theme", "Explain"}

// 分页查询权限条数
func Authority__All_Count() (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Authority
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询权限 Page页码(0代表全部) Page_Size每页条数
func Authority__All_Query(Page uint, Page_Size uint) (Authority_Array []Authority__table_type, err error) {
	query := "SELECT `Id`, `Name`, `Theme`, `Explain` FROM `Authority` ORDER BY `Id` DESC "

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
		var Authority Authority__table_type
		err = rows.Scan(
			&Authority.Id,
			&Authority.Name,
			&Authority.Theme,
			&Authority.Explain,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Authority_Array = append(Authority_Array, Authority)
	}

	return
}

// 分页查询权限 Page页码(0代表全部) Page_Size每页条数
func Authority__User_Id_Query(Page uint, Page_Size uint) (Authority_Array []Authority__table_type, err error) {
	query := "SELECT `Id`, `Name`, `Theme`, `Explain` FROM `Authority` ORDER BY `Id` DESC "

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
		var Authority Authority__table_type
		err = rows.Scan(
			&Authority.Id,
			&Authority.Name,
			&Authority.Theme,
			&Authority.Explain,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Authority_Array = append(Authority_Array, Authority)
	}

	return
}

// 查询指定权限Id
func Authority__Id_Array_Query(Authority_Id []uint) (Authority_Array []Authority__table_type, err error) {
	Authority_Id_len := len(Authority_Id)
	if Authority_Id_len == 0 {
		return
	}

	// 构建占位符
	placeholders := make([]string, Authority_Id_len)
	args := make([]interface{}, Authority_Id_len)
	for i, id := range Authority_Id {
		if id == 0 {
			err = fmt.Errorf("Authority_Id不能是0")
			log.Print(err.Error())
			return
		}
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("SELECT `Id`, `Name`, `Theme`, `Explain` FROM `Authority` WHERE `Id` IN (%s)", strings.Join(placeholders, ","))

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
		var Authority Authority__table_type
		err = rows.Scan(
			&Authority.Id,
			&Authority.Name,
			&Authority.Theme,
			&Authority.Explain,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Authority_Array = append(Authority_Array, Authority)
	}

	return
}

// 查询多个用户信息 传递:搜索值，搜索方式，数量
func Authority__Array_Search(Search string, Type string, Number uint) (Authority_Array []Authority__table_type, err error) {
	if Type != "" && !slices.Contains(Authority__Search_Type, Type) {
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

	query := "SELECT `Id`, `Name`, `Theme`, `Explain` FROM `Authority` WHERE ? LIKE ? LIMIT ? "

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
		var Authority Authority__table_type
		err = rows.Scan(
			&Authority.Id,
			&Authority.Name,
			&Authority.Theme,
			&Authority.Explain,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Authority_Array = append(Authority_Array, Authority)
	}

	return
}

// 增加权限
func Authority__Add(Value Authority__table_type) (Authority_Id uint, err error) {
	if Value.Id != 0 {
		err = fmt.Errorf("增加权限,Id需要为空")
		return
	}
	query := "INSERT INTO `Authority`(`Name`, `Theme`, `Explain`) VALUES(?,?,?) "
	// 修改数据库
	var result sql.Result
	result, err = DB.Exec(query, Value.Name, Value.Theme, Value.Explain)
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
	Authority_Id = uint(LastInsertId)

	return
}

// 修改权限
func Authority__Update(Value Authority__table_type) (err error) {
	if Value.Id == 0 {
		err = fmt.Errorf("修改权限,Id不能为空")
		return
	}

	query := "UPDATE `Authority` SET `Name` = ?, `Theme` = ?, `Explain` = ? WHERE `Id` = ? "
	// 修改数据库
	_, err = DB.Exec(query, Value.Name, Value.Theme, Value.Explain, Value.Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 删除权限
func Authority__Del(Authority_Id uint) (err error) {
	if Authority_Id == 0 {
		err = fmt.Errorf("删除权限,Id不能为空")
		return
	}

	query := "DELETE FROM `Authority` WHERE `Id` = ? "
	// 修改数据库
	_, err = DB.Exec(query, Authority_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

/*
***************用户对应的权限***************
 */
// 用户对应的权限表结构体
type Authority_User__table_type struct {
	Id           uint
	User_Id      uint // 用户id
	Authority_Id uint // 权限id
	Enable       bool // 使能
}

// 分页查询全部用户权限条数
func Authority_User__All_Count() (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Authority_User
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询全部用户权限 Page页码(0代表全部) Page_Size每页条数
func Authority_User__All_Query(Page uint, Page_Size uint) (Authority_Array []Authority_User__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Authority_Id,
		Enable
	FROM
		Authority_User
	`

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ?, ? "
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
		var Authority Authority_User__table_type
		err = rows.Scan(
			&Authority.Id,
			&Authority.User_Id,
			&Authority.Authority_Id,
			&Authority.Enable,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Authority_Array = append(Authority_Array, Authority)
	}

	return
}

// 分页查询指定用户权限条数
func Authority_User__User_Count(User_Id uint) (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Authority_User
	WHERE
		User_Id = ?
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query, User_Id).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询指定用户权限 Page页码(0代表全部) Page_Size每页条数
func Authority_User__User_Query(User_Id uint, Page uint, Page_Size uint) (Authority_Array []Authority_User__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Authority_Id,
		Enable
	FROM
		Authority_User
	WHERE
		User_Id = ?
	`

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ? OFFSET ? "
		Page -= 1
		rows, err = DB.Query(query, User_Id, Page_Size, Page)
	} else {
		rows, err = DB.Query(query, User_Id)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var Authority Authority_User__table_type
		err = rows.Scan(
			&Authority.Id,
			&Authority.User_Id,
			&Authority.Authority_Id,
			&Authority.Enable,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Authority_Array = append(Authority_Array, Authority)
	}

	return
}

// 分页查询指定权限的用户权限条数
func Authority_User__Authority_Count(Authority_Id uint) (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Authority_User
	WHERE
		Authority_Id = ?
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query, Authority_Id).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询指定权限的用户权限 Page页码(0代表全部) Page_Size每页条数
func Authority_User__Authority_Query(Authority_Id uint, Page uint, Page_Size uint) (Authority_Array []Authority_User__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Authority_Id,
		Enable
	FROM
		Authority_User
	WHERE
		Authority_Id = ?
	`

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ? OFFSET ? "
		Page -= 1
		rows, err = DB.Query(query, Authority_Id, Page_Size, Page)
	} else {
		rows, err = DB.Query(query, Authority_Id)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var Authority Authority_User__table_type
		err = rows.Scan(
			&Authority.Id,
			&Authority.User_Id,
			&Authority.Authority_Id,
			&Authority.Enable,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Authority_Array = append(Authority_Array, Authority)
	}

	return
}

// 权限使能设定
func Authority_User__Enable(Authority_User_Id uint, Authority_Enable bool) (err error) {
	query := `
	UPDATE Authority_User
		SET Enable = ? 
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Authority_Enable, Authority_User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 权限增加
func Authority_User__Add(Value Authority_User__table_type) (err error) {
	if Value.Id != 0 {
		err = fmt.Errorf("增加权限,Id需要为空")
		return
	}

	query := `
	INSERT
		INTO
		Authority_User(User_Id, Authority_Id, Enable)
	VALUES(?,?,?)
	`
	// 修改数据库
	_, err = DB.Exec(query, Value.User_Id, Value.Authority_Id, Value.Enable)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 权限删除
func Authority_User__Del(Authority_User_Id uint) (err error) {
	query := `
	DELETE
	FROM
		Authority_User
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Authority_User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 同用户id和权限主题查询是否有这个权限
func Authority_User__Query_AuthorityTheme_Exist(User_Id uint, Authority_Theme string) (Exist bool, err error) {
	query := `
	    SELECT
	        Authority_User.Enable
		FROM Authority_User
		INNER JOIN Authority ON Authority_User.Authority_Id = Authority.Id 
		INNER JOIN User ON Authority_User.User_Id = User.Id 
		WHERE User.Id=? AND Authority.Theme=?
		LIMIT 1
	`
	err = DB.QueryRow(query, User_Id, Authority_Theme).Scan(
		&Exist,
	)
	if err != nil {
		log.Print(err.Error())
	}
	return
}

/*
***************用户组***************
 */
// 用户组表结构体
type Group__table_type struct {
	Id      uint
	Name    string // 组名称
	Explain string // 组说明
}

// 用户组全部条数
func Group__All_Count() (Count uint, err error) {
	query := "SELECT COUNT(`Id`) FROM `Group` ORDER BY `Id` DESC "
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询用户组 Page页码(0代表全部) Page_Size每页条数
func Group__All_Query(Page uint, Page_Size uint) (Group_Array []Group__table_type, err error) {
	query := "SELECT `Id`, `Name`, `Explain` FROM `Group` ORDER BY `Id` DESC "

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ? OFFSET ? "
		Page -= 1
		rows, err = DB.Query(query, Page_Size, Page)
	} else {
		rows, err = DB.Query(query)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var Group Group__table_type
		err = rows.Scan(
			&Group.Id,
			&Group.Name,
			&Group.Explain,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Group_Array = append(Group_Array, Group)
	}

	return
}

// 用户组增加
func Group__Add(Value Group__table_type) (err error) {
	if Value.Id != 0 {
		err = fmt.Errorf("用户组增加,Id需要为空")
		return
	}

	query := "INSERT INTO `Group`(`Name`, `Explain`) VALUES(?,?) "
	// 修改数据库
	_, err = DB.Exec(query, Value.Name, Value.Explain)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 用户组增加
func Group_Update(Value Group__table_type) (err error) {
	if Value.Id == 0 {
		err = fmt.Errorf("修改分组,Id不能为空")
		return
	}

	query := "UPDATE `Group` SET `Name` = ?, `Explain` = ? WHERE `Id` = ? "

	// 修改数据库
	_, err = DB.Exec(query, Value.Name, Value.Explain)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 权限删除
func Group__Del(Group_Id uint) (err error) {
	query := "DELETE FROM `Group` WHERE `Id` = ? "
	// 修改数据库
	_, err = DB.Exec(query, Group_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

/*
***************用户组管理***************
 */
// 用户对应的用户组表结构体
type Group_User__table_type struct {
	Id            uint
	User_Id       uint // 用户id
	Group_Id      uint // 用户组id
	Administrator bool // 是否是管理员
}

// 查询全部用户组条数
func Group_User__All_Count() (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Group_User
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询全部用户组 Page页码(0代表全部) Page_Size每页条数
func Group_User__All_Query(Page uint, Page_Size uint) (Group_User_Array []Group_User__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Group_Id,
		Administrator
	FROM
		Group_User
	`

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ?, ? "
		Page -= 1
		rows, err = DB.Query(query, Page_Size, Page)
	} else {
		rows, err = DB.Query(query)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var Group_User Group_User__table_type
		err = rows.Scan(
			&Group_User.Id,
			&Group_User.User_Id,
			&Group_User.Group_Id,
			&Group_User.Administrator,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Group_User_Array = append(Group_User_Array, Group_User)
	}

	return
}

// 查询指定用户的组条数
func Group_User__User_Count(User_Id uint) (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Group_User
	WHERE
		User_Id = ?
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query, User_Id).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询指定用户的组 Page页码(0代表全部) Page_Size每页条数
func Group_User__User_Query(User_Id uint, Page uint, Page_Size uint) (Group_User_Array []Group_User__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Group_Id,
		Administrator
	FROM
		Group_User
	WHERE
		User_Id = ?
	`

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ? OFFSET ? "
		Page -= 1
		rows, err = DB.Query(query, User_Id, Page_Size, Page)
	} else {
		rows, err = DB.Query(query, User_Id)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var Group_User Group_User__table_type
		err = rows.Scan(
			&Group_User.Id,
			&Group_User.User_Id,
			&Group_User.Group_Id,
			&Group_User.Administrator,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Group_User_Array = append(Group_User_Array, Group_User)
	}

	return
}

// 查询指定组的用户条数
func Group_User__Group_Count(Group_Id uint) (Count uint, err error) {
	query := `
	SELECT
		COUNT(Id)
	FROM
		Group_User
	WHERE
		Group_Id = ?
	`
	// 是否搜索全部 对于0侧全部
	err = DB.QueryRow(query, Group_Id).Scan(&Count)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 分页查询指定组的用户 Page页码(0代表全部) Page_Size每页条数
func Group_User__Group_Query(Group_Id uint, Page uint, Page_Size uint) (Group_User_Array []Group_User__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		Group_Id,
		Administrator
	FROM
		Group_User
	WHERE
		Group_Id = ?
	`

	var (
		rows *sql.Rows
	)
	// 是否搜索全部 对于0侧全部
	if Page != 0 {
		query += "LIMIT ? OFFSET ? "
		Page -= 1
		rows, err = DB.Query(query, Group_Id, Page_Size, Page)
	} else {
		rows, err = DB.Query(query, Group_Id)
	}
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var Group_User Group_User__table_type
		err = rows.Scan(
			&Group_User.Id,
			&Group_User.User_Id,
			&Group_User.Group_Id,
			&Group_User.Administrator,
		)
		if err != nil {
			log.Print(err.Error())
			return
		}

		Group_User_Array = append(Group_User_Array, Group_User)
	}

	return
}

// 查询这个用户是否是这个用户组的管理员
func Group_User__Administrator_Exist(User_Id uint, Group_Id uint) (Exist bool, err error) {
	query := `
	    SELECT
	        Administrator
		FROM Group_User 
		WHERE User_Id=? AND Group_Id=?
		LIMIT 1
	`
	err = DB.QueryRow(query, User_Id, Group_Id).Scan(
		&Exist,
	)
	if err != nil {
		log.Print(err.Error())
	}
	return
}

// 组管理员设定
func Group_User__Administrator(Group_User_Id uint, Administrator bool) (err error) {
	query := `
	UPDATE Group_User
		SET Administrator = ? 
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Administrator, Group_User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 组增加
func Group_User__Add(Value Group_User__table_type) (err error) {
	if Value.Id != 0 || Value.User_Id == 0 || Value.Group_Id == 0 {
		err = fmt.Errorf("组增加，参数错误")
		return
	}

	query := `
	INSERT
		INTO
		Group_User(User_Id, Group_Id, Administrator)
	VALUES(?,?,?)
	`
	// 修改数据库
	_, err = DB.Exec(query, Value.User_Id, Value.Group_Id, Value.Administrator)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 组删除Group_User__Permission
func Group_User__Del(Group_User_Id uint) (err error) {
	query := `
	DELETE
	FROM
		Group_User
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Group_User_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 查询用户A和用户B，用户A否是管理员
func Group_User__User_Administrator(User_Id uint, Administrator_User_Id uint) (count uint, err error) {
	var Administrator_Group_User []Group_User__table_type
	Administrator_Group_User, err = Group_User__User_Query(Administrator_User_Id, 0, 0)
	if err == sql.ErrNoRows {
		err = nil
		return
	} else if err != nil {
		log.Print(err.Error())
		return
	}

	for _, v := range Administrator_Group_User {
		if !v.Administrator {
			continue
		}

		Administrator_Group_User, err = Group_User__Group_Query(v.Group_Id, 0, 10)
		if err == sql.ErrNoRows {
			err = nil
			continue
		} else if err != nil {
			log.Print(err.Error())
			return
		}
		count += 1
	}
	return
}

// 查询用户A和用户B，用户A否是有权限
func Group_User__Permission(User_Id uint, Administrator_User_Id uint) (yes bool, err error) {
	var count uint
	count, err = Group_User__User_Administrator(User_Id, Administrator_User_Id)

	if err != sql.ErrNoRows && err != nil {
		return
	}

	if count == 0 {
		var Permissions uint
		Permissions, err = User__Permissions_Query(Administrator_User_Id)
		if err != nil {
			return
		}
		fmt.Print(Permissions, Init.User_Permissions, "=====\n")

		if Permissions < Init.User_Permissions {
			yes = true
			return
		}
		return
	}

	yes = true
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

/*
***************接口***************
 */
// 菜单表结构体
type Api__table_type struct {
	Id                 uint
	User_Id            uint // 用户id
	ApiKey             string
	Secret             string // 秘密
	Refresh_Token_Time uint   // 过期时间设定（s）
	Allow_Ip           string // ip
	Discontinued       bool   // 是否禁用
	Refresh_Token_bits int    // 刷新令牌RSA私钥长度
	Access_Token_bits  int    // 访问令牌RSA公钥长度
}

// 查询接口信息
func Api__Query_ApiKey(ApiKey string) (Api Api__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		ApiKey,
		Secret,
		Refresh_Token_Time,
		Allow_Ip,
		Discontinued,
		Refresh_Token_bits,
		Access_Token_bits
	FROM
		Api
	WHERE
		ApiKey = ?
	`
	err = DB.QueryRow(query, ApiKey).Scan(
		&Api.Id,
		&Api.User_Id,
		&Api.ApiKey,
		&Api.Secret,
		&Api.Refresh_Token_Time,
		&Api.Allow_Ip,
		&Api.Discontinued,
		&Api.Refresh_Token_bits,
		&Api.Access_Token_bits,
	)
	if err != nil {
		log.Print(err.Error())
	}

	if Api.Discontinued {
		err = fmt.Errorf("Warning ApiKey:%s 已经禁用了", ApiKey)
	}
	return
}

// 查询接口信息
func Api__Query() (Api Api__table_type, err error) {
	query := `
	SELECT
		Id,
		User_Id,
		ApiKey,
		Secret,
		Refresh_Token_Time,
		Allow_Ip,
		Discontinued,
		Refresh_Token_bits,
		Access_Token_bits
	FROM
		Api
	`
	err = DB.QueryRow(query).Scan(
		&Api.Id,
		&Api.User_Id,
		&Api.ApiKey,
		&Api.Secret,
		&Api.Refresh_Token_Time,
		&Api.Allow_Ip,
		&Api.Discontinued,
		&Api.Refresh_Token_bits,
		&Api.Access_Token_bits,
	)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 查询接口信息
func Api__Query_Id__AccessTokenBits(Id uint) (Access_Token_bits int, err error) {
	query := `
	SELECT
		Access_Token_bits
	FROM
		Api
	WHERE
		Id = ?
	`
	err = DB.QueryRow(query, Id).Scan(&Access_Token_bits)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 增加接口
func Api__Add(Value Api__table_type) (Id uint, err error) {
	if Value.Id != 0 {
		err = fmt.Errorf("增加接口,Id需要为空")
		return
	}
	query := `
	INSERT
		INTO
		Api(User_Id, ApiKey, Secret, Refresh_Token_Time, Allow_Ip, Discontinued, Refresh_Token_bits, Access_Token_bits)
	VALUES(?,?,?,?,?,?,?,?)
	`
	// 修改数据库
	var result sql.Result
	result, err = DB.Exec(query,
		Value.User_Id,
		Value.ApiKey,
		Value.Secret,
		Value.Refresh_Token_Time,
		Value.Allow_Ip,
		Value.Discontinued,
		Value.Refresh_Token_bits,
		Value.Access_Token_bits,
	)
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

// 禁用api_id
func Api__Discontinued(Api_Id uint, Discontinued bool) (err error) {
	query := `
	UPDATE
		Api
	SET
		Discontinued = ?
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Discontinued, Api_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}

// 删除
func Api__Del(Api_Id uint) (err error) {
	query := `
	DELETE
	FROM
		Api
	WHERE
		Id = ?
	`
	// 修改数据库
	_, err = DB.Exec(query, Api_Id)
	if err != nil {
		log.Print(err.Error())
	}

	return
}
