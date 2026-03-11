/*
* 日期: 2025.12.23 16:50
* 作者: 范范zwf
* 作用: api-用户相关
 */
package web

import (
	"main/Init"
	db_mysql "main/db/mysql"
	db_redis "main/db/redis"

	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

/*
***************登陆***************
 */
var (
	User_Refresh_Token_Length uint = 200 // 刷新令牌长度
	User_Access_Token_Length  uint = 120 // 访问令牌长度
)

// 用户名登陆
func User_Login_Name(ctx *gin.Context) {
	var jsondata struct {
		Name   string
		Passwd string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 正则表达式计算
	matched, err := regexp.MatchString(Init.Regex_Name, jsondata.Name)
	if err != nil {
		ctx.Set("Response", []any{StatusRegex, "正则表达式计算错误"})
		return
	}
	if !matched {
		ctx.Set("Response", []any{403, "用户名输入不合法"})
		return
	}
	// 正则表达计算
	matched, err = regexp.MatchString(Init.Regex_Passwd_sha3_256, jsondata.Passwd)
	if err != nil {
		ctx.Set("Response", []any{StatusRegex, "正则表达式计算错误"})
		return
	}
	if !matched {
		ctx.Set("Response", []any{403, "密码输入不合法"})
		return
	}

	// 判断用户名密码是否正确
	User, err := db_mysql.User__NamePasswd_Query(jsondata.Name, jsondata.Passwd)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{403, "用户名或密码错误"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	if User.Discontinued {
		ctx.Set("Response", []any{403, "此用户已禁用"})
		return
	}

	// 生成随即刷新令牌
	Refresh_Token, err := create_token(User_Refresh_Token_Length)
	if err != nil {
		ctx.Set("Response", []any{541, err.Error()})
		return
	}
	fmt.Print(Refresh_Token, "========\n")
	var (
		Ip            = ctx.ClientIP()
		Header        = ctx.Request.Header.Get("User-Agent")
		Terminal_Uuid = ctx.Request.Header.Get("F_Terminal_Uuid")
	)

	now := time.Now()
	Expiration := time.Duration(User.Refresh_Token_Time) * time.Second
	Expires_in := now.Add(Expiration)

	// 把刷新令牌写入对应用户的表里

	err = db_mysql.User_Terminal__Add(db_mysql.User_Terminal__table_type{
		User_Id:       User.Id,       // 用户id
		Terminal_Uuid: Terminal_Uuid, // 终端uuid
		Device_Name:   Header,        // 设备名称
		Ip:            Ip,            // 登陆ip
	})
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	err = db_redis.Refresh_Token_Add(User.Id, Refresh_Token, db_redis.Refresh_Token_redis_type{
		User_Id:       User.Id,                          // 用户id
		Terminal_Uuid: Terminal_Uuid,                    // 用户终端Id
		Expires_in:    Expires_in.Format(time.DateTime), // 访问令牌过期时间
	}, Expiration)

	if err != nil {
		ctx.Set("Response", []any{StatusRedis, err.Error()})
		return
	}

	db_mysql.Log__Add(db_mysql.Log__table_type{
		User_Id: User.Id,                                      // 用户id
		Type:    "login",                                      // 类型
		Message: fmt.Sprintf("登陆成功 IP:%s;请求头:%s", Ip, Header), // 描述
		Time:    time.Now(),                                   // 时间
	})

	ctx.Set("Response", []any{200, "ok", gin.H{
		"User_Id":         User.Id,
		"F_Refresh_Token": Refresh_Token,
		"F_Expires_in":    Expires_in.Format(time.DateTime),
	}})
}

// 获取访问令牌
func User_Access_Token_query(ctx *gin.Context) {
	var jsondata struct {
		User_Id         uint
		F_Refresh_Token string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 判断刷新令牌是否过期
	Access_Token_redis, err := db_redis.Refresh_Token_Query(jsondata.User_Id, jsondata.F_Refresh_Token)
	if err == redis.Nil {
		ctx.Set("Response", []any{401, "刷新令牌过期"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusRedis, err.Error()})
		return
	}

	// 生成随即刷新令牌
	Access_Token, err := create_token(User_Access_Token_Length)
	if err != nil {
		ctx.Set("Response", []any{541, err.Error()})
		return
	}

	value, err := db_mysql.Set_Type_Query("User_Access_Token_Time")
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	User_Access_Token_Time, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 把刷新令牌写入对应用户的表里
	now := time.Now()
	User_Access_Token_Time_Second := time.Duration(User_Access_Token_Time) * time.Second
	Expires_in := now.Add(User_Access_Token_Time_Second)
	err = db_redis.Access_Token_Add(
		Access_Token,
		db_redis.Access_Token_redis_type{
			User_Id:       Access_Token_redis.User_Id,       // 用户id
			Expires_in:    Expires_in.Format(time.DateTime), // 访问令牌过期时间
			Refresh_Token: jsondata.F_Refresh_Token,         // 本访问令牌的刷新令牌
		},
		User_Access_Token_Time_Second,
	)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", gin.H{
		"F_Access_Token": Access_Token,
		"F_Expires_in":   Expires_in.Format(time.DateTime),
	}})
}

/*
***************用户***************
 */
// 获取用户信息
func User_Get_Info(ctx *gin.Context) {
	var jsondata struct {
		User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}

	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.User_Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	var User db_mysql.User__table_type
	User, err = db_mysql.User__Info_Query(jsondata.User_Id)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	fmt.Print(User)
	ctx.Set("Response", []any{200, "ok", User})
}

// 获取多个用户信息
func User_Get_Info_Array(ctx *gin.Context) {
	var jsondata struct {
		User_Id_Array []uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= Init.User_Permissions {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}
	var User []db_mysql.User__table_type
	User, err = db_mysql.User__Info_Array_Query(jsondata.User_Id_Array)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", User})
}

// 获取条数
func User_All_Count(ctx *gin.Context) {
	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Count, err := db_mysql.User__All_Count()
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 分页查询 Page页码(0代表全部) Page_Size每页条数
func User_All_Query(ctx *gin.Context) {
	var jsondata struct {
		Page      uint
		Page_Size uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Count, err := db_mysql.User__All_Query(jsondata.Page, jsondata.Page_Size)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 搜索用户信息
func User_get_Info_Search(ctx *gin.Context) {
	var jsondata struct {
		Search string
		Type   string
		Number uint // 输出数量
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= Init.User_Permissions {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	var User_Info_array []db_mysql.User__table_type
	User_Info_array, err = db_mysql.User__Info_Array_Search(jsondata.Search, jsondata.Type, jsondata.Number)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", User_Info_array})

}

// 增加用户
func User_Set_Add(ctx *gin.Context) {
	var jsondata db_mysql.User__all_table_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 正则表达计算->用户名
	matched, err := regexp.MatchString(Init.Regex_Name, jsondata.Name)
	if err != nil {
		ctx.Set("Response", []any{StatusRegex, "用户名正则表达式计算错误"})
		return
	}
	if !matched {
		ctx.Set("Response", []any{403, "输入不合法"})
		return
	}

	// 正则表达计算->密码
	matched, err = regexp.MatchString(Init.Regex_Passwd_sha3_256, jsondata.Passwd)
	if err != nil {
		ctx.Set("Response", []any{StatusRegex, "密码正则表达式计算错误"})
		return
	}
	if !matched {
		ctx.Set("Response", []any{403, "输入不合法"})
		return
	}

	// 正则表达计算->电话
	if jsondata.Phone != "" {
		matched, err = regexp.MatchString(Init.Regex_Phone, jsondata.Phone)
		if err != nil {
			ctx.Set("Response", []any{StatusRegex, "电话正则表达式计算错误"})
			return
		}
		if !matched {
			ctx.Set("Response", []any{403, "输入不合法"})
			return
		}
	}

	// 正则表达计算->邮箱
	if jsondata.Email != "" {
		matched, err = regexp.MatchString(Init.Regex_Email, jsondata.Phone)
		if err != nil {
			ctx.Set("Response", []any{StatusRegex, "邮箱正则表达式计算错误"})
			return
		}
		if !matched {
			ctx.Set("Response", []any{403, "输入不合法"})
			return
		}
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	if jsondata.Id == 0 {
		jsondata.Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	_, err = db_mysql.User__Add(jsondata)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("增加用户:%s", jsondata.Name))
	ctx.Set("Response", []any{200, "ok"})
}

// 设置用户名
func User_Set_Name(ctx *gin.Context) {
	var jsondata struct {
		Name    string
		User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 正则表达计算
	matched, err := regexp.MatchString(Init.Regex_Name, jsondata.Name)
	if err != nil {
		ctx.Set("Response", []any{StatusRegex, "正则表达式计算错误"})
		return
	}
	if !matched {
		ctx.Set("Response", []any{403, "输入不合法"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.User_Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	err = db_mysql.User__Name_Update(jsondata.User_Id, jsondata.Name)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("用户id:%d 修改名称:%s", jsondata.User_Id, jsondata.Name))

	ctx.Set("Response", []any{200, "ok"})
}

// 设置密码
func User_Set_Passwd(ctx *gin.Context) {
	var jsondata struct {
		Passwd  string
		User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 正则表达式计算
	matched, err := regexp.MatchString(Init.Regex_Passwd_sha3_256, jsondata.Passwd)
	if err != nil {
		ctx.Set("Response", []any{StatusRegex, "正则表达式计算错误"})
		return
	}
	if !matched {
		ctx.Set("Response", []any{403, "输入不合法"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.User_Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	err = db_mysql.User__Passwd_Update(jsondata.User_Id, jsondata.Passwd)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("用户id:%d 修改密码", jsondata.User_Id))
	ctx.Set("Response", []any{200, "ok"})
}

// 设置电话
func User_Set_Phone(ctx *gin.Context) {
	var jsondata struct {
		Phone   string
		User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if jsondata.Phone != "" {
		// 正则表达式计算
		matched, err := regexp.MatchString(Init.Regex_Phone, jsondata.Phone)
		if err != nil {
			ctx.Set("Response", []any{StatusRegex, "正则表达式计算错误"})
			return
		}
		if !matched {
			ctx.Set("Response", []any{403, "输入不合法"})
			return
		}
	}

	// 用户id不存在，赋值登陆的用户iddb_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("修改用户id:%d 电话:%s", jsondata.User_Id, jsondata.Phone))
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.User_Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	err = db_mysql.User__Phone_Update(jsondata.User_Id, jsondata.Phone)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("用户id:%d 修改电话:%s", jsondata.User_Id, jsondata.Phone))
	ctx.Set("Response", []any{200, "ok"})
}

// 设置邮箱
func User_Set_Email(ctx *gin.Context) {
	var jsondata struct {
		Email   string
		User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if jsondata.Email != "" {
		// 正则表达式计算
		matched, err := regexp.MatchString(Init.Regex_Email, jsondata.Email)
		if err != nil {
			ctx.Set("Response", []any{StatusRegex, "正则表达式计算错误"})
			return
		}
		if !matched {
			ctx.Set("Response", []any{403, "输入不合法"})
			return
		}
	}
	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.User_Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	err = db_mysql.User__Email_Update(jsondata.User_Id, jsondata.Email)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("用户id:%d 修改邮箱:%s", jsondata.User_Id, jsondata.Email))
	ctx.Set("Response", []any{200, "ok"})
}

// 删除用户
func User_Set_Del(ctx *gin.Context) {
	var jsondata struct {
		User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.User_Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	err = db_mysql.User__Del(jsondata.User_Id)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("用户id:%d 修改", jsondata.User_Id))
	ctx.Set("Response", []any{200, "ok"})
}

// 设置停用
func User_Set_Discontinued(ctx *gin.Context) {
	var jsondata struct {
		User_Id      uint
		Discontinued bool
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.User_Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	err = db_mysql.User__Discontinued_Update(jsondata.User_Id, jsondata.Discontinued)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("用户id:%d 停用%t", jsondata.User_Id, jsondata.Discontinued))
	ctx.Set("Response", []any{200, "ok"})
}

// 头像url地址
func User_Set_Avatar(ctx *gin.Context) {
	var jsondata struct {
		User_Id uint
		Avatar  string
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if jsondata.Avatar != "" {
		// 正则表达式计算
		matched, err := regexp.MatchString(Init.Regex_URL, jsondata.Avatar)
		if err != nil {
			ctx.Set("Response", []any{StatusRegex, "正则表达式计算错误"})
			return
		}
		if !matched {
			ctx.Set("Response", []any{403, "输入不合法"})
			return
		}
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {
		yes, err := db_mysql.Group_User__Permission(jsondata.User_Id, Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{520, err.Error()})
			return
		}
		if !yes {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	err = db_mysql.User__Avatar_Update(jsondata.User_Id, jsondata.Avatar)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "User", fmt.Sprintf("用户id:%d 头像%s", jsondata.User_Id, jsondata.Avatar))
	ctx.Set("Response", []any{200, "ok"})
}

/*
***************权限***************
 */
// 获取权限条数
func Authority__All_Count(ctx *gin.Context) {
	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Count, err := db_mysql.Authority__All_Count()
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 分页查询权限 Page页码(0代表全部) Page_Size每页条数
func Authority__All_Query(ctx *gin.Context) {
	var jsondata struct {
		Page      uint
		Page_Size uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= Init.User_Permissions {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Count, err := db_mysql.Authority__All_Query(jsondata.Page, jsondata.Page_Size)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 查询指定权限Id
func Authority__Id_Array_Query(ctx *gin.Context) {
	var jsondata struct {
		Authority_Id []uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= Init.User_Permissions {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Authority_Array, err := db_mysql.Authority__Id_Array_Query(jsondata.Authority_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Authority_Array})
}

// 搜索权限
func Authority__Search(ctx *gin.Context) {
	var jsondata struct {
		Search string
		Type   string
		Number uint // 输出数量
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= Init.User_Permissions {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	var Authority_Array []db_mysql.Authority__table_type
	Authority_Array, err = db_mysql.Authority__Array_Search(jsondata.Search, jsondata.Type, jsondata.Number)
	if err != nil {
		ctx.Set("Response", []any{520, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Authority_Array})

}

// 增加权限
func Authority__Add(ctx *gin.Context) {
	var jsondata db_mysql.Authority__table_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if jsondata.Id != 0 {
		ctx.Set("Response", []any{500, "增加权限, 权限Id是0"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= Init.User_Permissions {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Authority_Id, err := db_mysql.Authority__Add(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "Authority",
		fmt.Sprintf("增加 名称:%s 主题%s 说明%s", jsondata.Name, jsondata.Theme, jsondata.Explain),
	)
	ctx.Set("Response", []any{200, "ok", Authority_Id})
}

// 修改权限
func Authority__Update(ctx *gin.Context) {
	var jsondata db_mysql.Authority__table_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if jsondata.Id == 0 {
		ctx.Set("Response", []any{500, "修改权限, 权限Id不应该是0"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= Init.User_Permissions {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	err = db_mysql.Authority__Update(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "Authority",
		fmt.Sprintf("修改 名称:%s 主题%s 说明%s", jsondata.Name, jsondata.Theme, jsondata.Explain),
	)
	ctx.Set("Response", []any{200, "ok"})
}

// 删除权限
func Authority__Del(ctx *gin.Context) {
	var jsondata struct {
		Authority_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if jsondata.Authority_Id == 0 {
		ctx.Set("Response", []any{403, "Authority_Id不应该是0"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	if Permissions >= Init.User_Permissions {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	err = db_mysql.Authority__Del(jsondata.Authority_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(Token_User_Id, "Authority",
		fmt.Sprintf("删除 Authority_Id:%d", jsondata.Authority_Id),
	)
	ctx.Set("Response", []any{200, "ok"})
}

/*
***************用户对应的权限***************
 */

// 分页查询全部用户权限条数
func Authority_User__All_Count(ctx *gin.Context) {
	var jsondata struct {
		User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}

	if jsondata.User_Id != Token_User_Id {
		// 用户Id和登陆的用户id不一致，判断是否有权限
		Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{StatusMysql, err.Error()})
			return
		}
		if Permissions >= Init.User_Permissions {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	var Count uint
	if jsondata.User_Id == 0 {
		Count, err = db_mysql.Authority_User__All_Count()
	} else {
		Count, err = db_mysql.Authority_User__User_Count(jsondata.User_Id)
	}

	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", Count})
}

// 分页查询全部用户权限 Page页码(0代表全部) Page_Size每页条数
func Authority_User__All_Query(ctx *gin.Context) {
	var jsondata struct {
		User_Id   uint
		Page      uint
		Page_Size uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}

	if jsondata.User_Id != Token_User_Id {
		// 用户Id和登陆的用户id不一致，判断是否有权限
		Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{StatusMysql, err.Error()})
			return
		}
		if Permissions >= Init.User_Permissions {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	var Authority_Array []db_mysql.Authority_User__table_type
	if jsondata.User_Id == 0 {
		Authority_Array, err = db_mysql.Authority_User__All_Query(jsondata.Page, jsondata.Page_Size)
	} else {
		Authority_Array, err = db_mysql.Authority_User__User_Query(jsondata.User_Id, jsondata.Page, jsondata.Page_Size)
	}

	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "无数据"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	if len(Authority_Array) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", Authority_Array})
}

// 权限使能设定
func Authority_User__Enable(ctx *gin.Context) {
	var jsondata struct {
		Authority_User__Id uint
		Enable             bool
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	err = db_mysql.Authority_User__Enable(jsondata.Authority_User__Id, jsondata.Enable)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	var User_Id uint
	User_Id, err = Token_User_Id(ctx)
	db_mysql.Log__Add2(User_Id, "Authority_User",
		fmt.Sprintf("权限使能设定 Authority_User_Id:%d Enable:%t", jsondata.Authority_User__Id, jsondata.Enable),
	)
	ctx.Set("Response", []any{200, "ok"})
}

// 权限增加
func Authority_User__Add(ctx *gin.Context) {
	var jsondata db_mysql.Authority_User__table_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if jsondata.Id != 0 {
		ctx.Set("Response", []any{417, "参数不正确Id应该是0"})
		return
	}

	err = db_mysql.Authority_User__Add(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	var User_Id uint
	User_Id, err = Token_User_Id(ctx)
	db_mysql.Log__Add2(User_Id, "Authority_User",
		fmt.Sprintf("权限使能设定 Authority_User__Id:%d Enable:%t", jsondata.Authority_Id, jsondata.Enable),
	)

	ctx.Set("Response", []any{200, "ok"})
}

// 权限删除
func Authority_User__Del(ctx *gin.Context) {
	var jsondata struct {
		Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if jsondata.Id == 0 {
		ctx.Set("Response", []any{417, "参数不正确,Id不应该是0"})
		return
	}

	err = db_mysql.Authority_User__Del(jsondata.Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	var User_Id uint
	User_Id, err = Token_User_Id(ctx)
	db_mysql.Log__Add2(User_Id, "Authority_User",
		fmt.Sprintf("删除 Authority_User__Id:%d", jsondata.Id),
	)

	ctx.Set("Response", []any{200, "ok"})
}

/*
***************分组***************
 */
// 获取权限条数
func Group__All_Count(ctx *gin.Context) {
	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Count, err := db_mysql.Group__All_Count()
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 分页查询权限 Page页码(0代表全部) Page_Size每页条数
func Group__All_Query(ctx *gin.Context) {
	var jsondata struct {
		Page      uint
		Page_Size uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Count, err := db_mysql.Group__All_Query(jsondata.Page, jsondata.Page_Size)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 增加权限
func Group__Add(ctx *gin.Context) {
	var jsondata db_mysql.Group__table_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if jsondata.Id != 0 {
		ctx.Set("Response", []any{500, "增加权限, 权限Id是0"})
		return
	}

	var User_Id uint
	User_Id, err = Token_User_Id(ctx)
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	err = db_mysql.Group__Add(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(User_Id, "Group",
		fmt.Sprintf("删除 Name:%s Explain:%s", jsondata.Name, jsondata.Explain),
	)

	ctx.Set("Response", []any{200, "ok"})
}

// 修改权限
func Group__Update(ctx *gin.Context) {
	var jsondata db_mysql.Group__table_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if jsondata.Id == 0 {
		ctx.Set("Response", []any{500, "修改权限, 权限Id不应该是0"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	err = db_mysql.Group_Update(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(User_Id, "Group",
		fmt.Sprintf("删除 Id:%d Name:%s Explain:%s", jsondata.Id, jsondata.Name, jsondata.Explain),
	)
	ctx.Set("Response", []any{200, "ok"})
}

// 删除权限
func Group__Del(ctx *gin.Context) {
	var jsondata struct {
		Group_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if jsondata.Group_Id == 0 {
		ctx.Set("Response", []any{403, "Authority_Id不应该是0"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	err = db_mysql.Group__Del(jsondata.Group_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(User_Id, "Group",
		fmt.Sprintf("删除 Id:%d", jsondata.Group_Id),
	)
	ctx.Set("Response", []any{200, "ok"})
}

/*
***************用户分组***************
 */

// 查询指定组的用户条数
func Group_User__Group_Count(ctx *gin.Context) {
	var jsondata struct {
		Group_User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Count, err := db_mysql.Group_User__Group_Count(jsondata.Group_User_Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 分页查询指定组的用户 Page页码(0代表全部) Page_Size每页条数
func Group_User__Group_Query(ctx *gin.Context) {
	var jsondata struct {
		Group_User_Id uint
		Page          uint
		Page_Size     uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	Token_User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	Group_User, err := db_mysql.Group_User__Group_Query(jsondata.Group_User_Id, jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "无数据"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	if len(Group_User) == 0 {
		ctx.Set("ResponsGroup_User__Administratore", []any{404, "无数据"})
		return
	}
	ctx.Set("Response", []any{200, "ok", Group_User})
}

// 组管理员设定
func Group_User__Administrator(ctx *gin.Context) {
	var jsondata struct {
		Group_User__Id uint
		Administrator  bool
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	err = db_mysql.Group_User__Administrator(jsondata.Group_User__Id, jsondata.Administrator)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(User_Id, "Group_User",
		fmt.Sprintf("删除 Group_User__Id:%d Administrator:%t", jsondata.Group_User__Id, jsondata.Administrator),
	)
	ctx.Set("Response", []any{200, "ok"})
}

// 用户分组增加
func Group_User__Add(ctx *gin.Context) {
	var jsondata db_mysql.Group_User__table_type
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	err = db_mysql.Group_User__Add(jsondata)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(User_Id, "Group_User",
		fmt.Sprintf("删除 Id:%d User_Id:%d Group_Id:%d Administrator:%t",
			jsondata.Id,
			jsondata.User_Id,
			jsondata.Group_Id,
			jsondata.Administrator,
		),
	)
	ctx.Set("Response", []any{200, "ok"})
}

// 用户分组删除
func Group_User__Del(ctx *gin.Context) {
	var jsondata struct {
		Group_User__Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	User_Id, ok := value.(uint)
	if !ok {
		ctx.Set("Response", []any{500, "未知错误"})
		return
	}
	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	err = db_mysql.Group_User__Del(jsondata.Group_User__Id)
	if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	db_mysql.Log__Add2(User_Id, "Group_User",
		fmt.Sprintf("删除 Id:%d  ",
			jsondata.Group_User__Id,
		),
	)
	ctx.Set("Response", []any{200, "ok"})
}

type Log__api_type struct {
	Id      uint
	User_Id uint   // 用户id
	Type    string // 类型
	Message string // 描述
	Time    string // 时间
}

// 查询指定用户日志数量
func Log__User_Count(ctx *gin.Context) {
	var jsondata struct {
		User_Id uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if jsondata.User_Id == 0 {
		jsondata.User_Id = User_Id
	} else {
		// 用户Id和登陆的用户id不一致，判断是否有权限
		Permissions, err := db_mysql.User__Permissions_Query(User_Id)
		if err != nil {
			ctx.Set("Response", []any{StatusMysql, err.Error()})
			return
		}
		if Permissions >= 100 {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}
	}

	var Count uint
	Count, err = db_mysql.Log__User_Count(jsondata.User_Id)
	if err == sql.ErrNoRows {
		ctx.Set("Response", []any{404, "无数据"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	if Count == 0 {
		ctx.Set("Response", []any{404, "ok", Count})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 分页查询指定用户日志
func Log__User_Query(ctx *gin.Context) {
	var jsondata struct {
		User_Id   uint
		Page      uint
		Page_Size uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if jsondata.User_Id == 0 {
		jsondata.User_Id = User_Id
	} else {
		// 用户Id和登陆的用户id不一致，判断是否有权限
		Permissions, err := db_mysql.User__Permissions_Query(User_Id)
		if err != nil {
			ctx.Set("Response", []any{StatusMysql, err.Error()})
			return
		}
		if Permissions >= 100 {
			ctx.Set("Response", []any{403, "无权限"})
			return
		}

	}

	log_data_array, err := db_mysql.Log__User_Query(jsondata.User_Id, jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows || len(log_data_array) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	var log_data []Log__api_type
	for _, v := range log_data_array {
		log_data = append(log_data, Log__api_type{
			Id:      v.Id,
			User_Id: v.User_Id,                    // 用户id
			Type:    v.Type,                       // 类型
			Message: v.Message,                    // 描述
			Time:    v.Time.Format(time.DateTime), // 时间
		})
	}

	ctx.Set("Response", []any{200, "ok", log_data})
}

// 查询全部日志数量
func Log__All_Count(ctx *gin.Context) {

	User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	var Count uint
	Count, err = db_mysql.Log__All_Count()
	if err != nil && err != sql.ErrNoRows {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	if Count == 0 {
		ctx.Set("Response", []any{404, "ok", Count})
		return
	}
	ctx.Set("Response", []any{200, "ok", Count})
}

// 分页全部用户日志
func Log__All_Query(ctx *gin.Context) {
	var jsondata struct {
		Page      uint
		Page_Size uint
	}
	err := ctx.BindJSON(&jsondata)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	Permissions, err := db_mysql.User__Permissions_Query(User_Id)
	if err != nil {
		return
	}
	if Permissions >= 100 {
		ctx.Set("Response", []any{403, "无权限"})
		return
	}

	log_data_array, err := db_mysql.Log__All_Query(jsondata.Page, jsondata.Page_Size)
	if err == sql.ErrNoRows || len(log_data_array) == 0 {
		ctx.Set("Response", []any{404, "无数据"})
		return
	} else if err != nil {
		ctx.Set("Response", []any{StatusMysql, err.Error()})
		return
	}

	var log_data []Log__api_type
	for _, v := range log_data_array {
		log_data = append(log_data, Log__api_type{
			Id:      v.Id,
			User_Id: v.User_Id,                    // 用户id
			Type:    v.Type,                       // 类型
			Message: v.Message,                    // 描述
			Time:    v.Time.Format(time.DateTime), // 时间
		})
	}

	ctx.Set("Response", []any{200, "ok", log_data})
}

func gui_api(r *gin.Engine) {
	r.POST("/Gui/v1.0/Login/Name", User_Login_Name)                 // 用户名登陆
	r.POST("/Gui/v1.0/Login/Access_Token", User_Access_Token_query) // 获取访问令牌

	r.POST("/Gui/v1.0/User/Get/Count", User_All_Count)           // 获取条数
	r.POST("/Gui/v1.0/User/Get/Query", User_All_Query)           // 分页查询
	r.POST("/Gui/v1.0/User/Get/Info", User_Get_Info)             // 获取用户信息
	r.POST("/Gui/v1.0/User/Get/Info_Array", User_Get_Info_Array) // 查询多个用户信息
	r.POST("/Gui/v1.0/User/Get/Search", User_get_Info_Search)    // 搜索用户信息

	r.POST("/Gui/v1.0/User/Set/Add", User_Set_Add)                   // 增加用户
	r.POST("/Gui/v1.0/User/Set/Name", User_Set_Name)                 // 设置用户名
	r.POST("/Gui/v1.0/User/Set/Passwd", User_Set_Passwd)             // 设置密码
	r.POST("/Gui/v1.0/User/Set/Phone", User_Set_Phone)               // 设置电话
	r.POST("/Gui/v1.0/User/Set/Email", User_Set_Email)               // 设置邮箱
	r.POST("/Gui/v1.0/User/Set/Del", User_Set_Del)                   // 删除用户
	r.POST("/Gui/v1.0/User/Set/Discontinued", User_Set_Discontinued) // 设置停用
	r.POST("/Gui/v1.0/User/Set/Avatar", User_Set_Avatar)             // 头像url地址

	r.POST("/Gui/v1.0/Authority/Count", Authority__All_Count)   // 获取权限条数
	r.POST("/Gui/v1.0/Authority/Query", Authority__All_Query)   // 分页查询权限
	r.POST("/Gui/v1.0/Authority/Id", Authority__Id_Array_Query) // 查询指定权限Id
	r.POST("/Gui/v1.0/Authority/Search", Authority__Search)     // 搜索权限
	r.POST("/Gui/v1.0/Authority/Add", Authority__Add)           // 增加权限
	r.POST("/Gui/v1.0/Authority/Update", Authority__Update)     // 修改权限
	r.POST("/Gui/v1.0/Authority/Del", Authority__Del)           // 删除权限

	r.POST("/Gui/v1.0/Authority_User/Count", Authority_User__All_Count) // 分页查询全部用户权限条数
	r.POST("/Gui/v1.0/Authority_User/Query", Authority_User__All_Query) // 分页查询全部用户权限 Page页码(0代表全部) Page_Size每页条数
	r.POST("/Gui/v1.0/Authority_User/Enable", Authority_User__Enable)   // 权限使能设定
	r.POST("/Gui/v1.0/Authority_User/Add", Authority_User__Add)         // 权限增加
	r.POST("/Gui/v1.0/Authority_User/Del", Authority_User__Del)         // 权限删除

	r.POST("/Gui/v1.0/Group/Count", Group__All_Count) // 获取权限条数
	r.POST("/Gui/v1.0/Group/Query", Group__All_Query) // 分页查询权限
	r.POST("/Gui/v1.0/Group/Add", Group__Add)         // 增加权限
	r.POST("/Gui/v1.0/Group/Update", Group__Update)   // 修改权限
	r.POST("/Gui/v1.0/Group/Del", Group__Del)         // 删除权限

	r.POST("/Gui/v1.0/Group_User/Count", Group_User__Group_Count)           // 查询指定组的用户条数
	r.POST("/Gui/v1.0/Group_User/Query", Group_User__Group_Query)           // 分页查询指定组的用户 Page页码(0代表全部) Page_Size每页条数
	r.POST("/Gui/v1.0/Group_User/Administrator", Group_User__Administrator) // 组管理员设定
	r.POST("/Gui/v1.0/Group_User/Add", Group_User__Add)                     // 用户分组增加
	r.POST("/Gui/v1.0/Group_User/Del", Group_User__Del)

	r.POST("/Gui/v1.0/Log/User/Count", Log__User_Count) // 查询指定用户日志数量
	r.POST("/Gui/v1.0/Log/User/Query", Log__User_Query) // 分页查询指定用户日志
	r.POST("/Gui/v1.0/Log/All/Count", Log__All_Count)   // 查询全部日志数量
	r.POST("/Gui/v1.0/Log/All/Query", Log__All_Query)   // 分页全部用户日志
}
