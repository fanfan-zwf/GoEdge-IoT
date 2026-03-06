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

	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	}

	// 用户Id和登陆的用户id不一致，判断是否有权限
	if jsondata.User_Id != Token_User_Id {

		permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{500, err.Error()})
			return
		}

		if permissions >= 100 {
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

	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if permissions >= 100 {
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
	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if permissions >= 100 {
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

	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if permissions >= 100 {
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

	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if permissions >= 100 {
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

	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	} else {
		permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{500, err.Error()})
			return
		}

		if permissions >= 100 {
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

	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	} else {
		permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{500, err.Error()})
			return
		}

		if permissions >= 100 {
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

	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	} else {
		permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{500, err.Error()})
			return
		}

		if permissions >= 100 {
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

	// 读取当前请求用户id
	Token_User_Id, err := Token_User_Id(ctx)
	if err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	if jsondata.User_Id == 0 {
		jsondata.User_Id = Token_User_Id
	} else {
		permissions, err := db_mysql.User__Permissions_Query(Token_User_Id)
		if err != nil {
			ctx.Set("Response", []any{500, err.Error()})
			return
		}

		if permissions >= 100 {
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

func gui_api(r *gin.Engine) {
	r.POST("/Gui/v1.0/Login/Name", User_Login_Name)                 // 用户名登陆
	r.POST("/Gui/v1.0/Login/Access_Token", User_Access_Token_query) // 获取访问令牌

	r.POST("/Gui/v1.0/User/Get/Count", User_All_Count)           // 获取条数
	r.POST("/Gui/v1.0/User/Get/Query", User_All_Query)           // 分页查询
	r.POST("/Gui/v1.0/User/Get/Info", User_Get_Info)             // 获取用户信息
	r.POST("/Gui/v1.0/User/Get/Info_Array", User_Get_Info_Array) // 查询多个用户信息
	r.POST("/Gui/v1.0/User/Get/Search", User_get_Info_Search)    // 搜索用户信息

}
