/*
* 日期: 2025.12.23 16:50
* 作者: 范范zwf
* 作用: api-用户相关
 */
package web

import (
	"encoding/json"
	"log"
	"main/Init"
	"main/db/db_point"
	db_mysql "main/db/mysql"
	db_redis "main/db/redis"
	"net/http"
	"sync"

	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
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

/*
***************前端web socket实时推送***************
 */
// -------------------------- 1. 核心结构体（和你的业务对齐） --------------------------
// Update_Value_type 点位更新数据结构
type Update_Value_type struct {
	Tag   string  `json:"tag"`   // 点位标签（和客户端订阅的Tag完全匹配）
	Value float64 `json:"value"` // 点位值
	Time  string  `json:"time"`  // 更新时间
}

// WSClient WebSocket客户端结构体
type WSClient struct {
	conn    *websocket.Conn
	Points  map[string]bool // 订阅的Tag集合
	closeCh chan struct{}   // 关闭信号
	writeMu sync.Mutex      // 单个客户端写锁
}

// -------------------------- 2. 全局配置 & 变量 --------------------------
var (
	// 修复WebSocket升级器（解决521错误核心）
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// 生产环境建议限定具体域名，示例：return r.Header.Get("Origin") == "http://你的前端域名"
			return true // 允许所有跨域，解决前端跨域导致的升级失败
		},
		ReadBufferSize:  4096, // 增大缓冲区，适配中文Tag
		WriteBufferSize: 4096,
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			// 自定义WebSocket升级错误，避免返回521
			w.WriteHeader(status)
			_, _ = w.Write([]byte(reason.Error()))
		},
	}

	// 客户端管理
	web_socket_clients = make(map[*WSClient]struct{})
	wsRWMu             sync.RWMutex // 读写锁

	// 推送配置
	writeTimeout = 5 * time.Second // 增大超时，适配网络波动
	maxGoroutine = 100
	pushChan     = make(chan pushTask, 2000) // 增大队列，适配高频推送
)

type pushTask struct {
	client     *WSClient
	updateList []db_point.Update_Value_type
}

// -------------------------- 3. 核心：适配你的路由的WebSocket处理函数 --------------------------
// api_app_monitor_ws 对应你的实际路由 /Gui/v1.0/Monitor/ws
func api_app_monitor_ws(ctx *gin.Context) {
	// 1. 鉴权（替换为你的真实验证逻辑，示例保留token验证）

	// 测试阶段可先注释鉴权，排查连接问题：
	// if token != "valid-token-123456" {
	// 	ctx.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
	// 	return
	// }

	// 2. 解析Tags参数（关键：处理中文、//分隔、URL编码）
	tagsStr := ctx.Query("tags")
	if tagsStr == "" {
		ctx.Set("Response", []any{417, "请求无数据"})
		return
	}
	// 解码URL编码的Tag（比如%E6%B5%8B%E8%AF%95 → 测试）
	// decodedTags, err := url.QueryUnescape(tagsStr)
	// if err != nil {
	// 	log.Printf("Tags解码失败: %v", err)
	// 	ctx.JSON(http.StatusBadRequest, gin.H{"error": "tags参数格式错误"})
	// 	return
	// }

	// 解析//分隔的Tag（你的Tag格式：//hezi_1//测试modbus_tcp/点位11）
	// 先去掉开头的//，再按//分割
	// tagParts := strings.Split(strings.TrimPrefix(decodedTags, "//"), "//")
	// tags := make(map[string]bool)
	// for _, tag := range tagParts {
	// 	tag = strings.TrimSpace(tag)
	// 	if tag != "" {
	// 		tags[tag] = true // 最终订阅的Tag：hezi_1、测试modbus_tcp/点位11
	// 	}
	// }

	var tags_list []string

	err := json.Unmarshal([]byte(tagsStr), &tags_list)
	if err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}
	if len(tagsStr) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "未订阅任何有效标签"})
		return
	}
	log.Printf("ERROR 解析到客户端订阅标签: %v", tags_list)

	tags := make(map[string]bool)
	for _, tag := range tags_list {
		tags[tag] = true
	}

	// 3. 升级HTTP连接为WebSocket（解决521错误的核心步骤）
	// 强制设置响应头，避免升级失败
	ctx.Header("Upgrade", "websocket")
	ctx.Header("Connection", "upgrade")
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, ctx.Writer.Header())
	if err != nil {
		log.Printf("ERROR WebSocket升级失败: %v", err)
		// 返回明确的500错误，而非521
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "连接升级失败"})
		return
	}

	// 4. 注册客户端
	client := &WSClient{
		conn:    conn,
		Points:  tags,
		closeCh: make(chan struct{}),
	}
	wsRWMu.Lock()
	web_socket_clients[client] = struct{}{}
	wsRWMu.Unlock()
	log.Printf("ERROR 客户端[%p]连接成功，订阅标签: %v", client, tags)

	// 5. 监听客户端断开（必须读取消息，否则连接会被立即断开）
	go func() {
		defer CloseWSClient(client)
		// 设置读超时，避免阻塞
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		conn.SetPongHandler(func(string) error {
			// 心跳响应：重置读超时
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			return nil
		})

		// 循环读取客户端消息（即使不处理，也要读）
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("ERROR 客户端[%p]断开连接: %v", client, err)
				break
			}
		}
	}()

	// 6. 主动发送心跳（避免连接被网关/浏览器断开）
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				client.writeMu.Lock()
				// 发送ping心跳
				err := conn.WriteMessage(websocket.PingMessage, nil)
				client.writeMu.Unlock()
				if err != nil {
					return
				}
			case <-client.closeCh:
				return
			}
		}
	}()
}

// -------------------------- 4. 推送逻辑（适配你的Tag格式） --------------------------
func InitWSPushPool() {
	for i := 0; i < maxGoroutine; i++ {
		go func() {
			for task := range pushChan {
				processPushTask(task)
			}
		}()
	}
}

func processPushTask(task pushTask) {
	client := task.client
	updateList := task.updateList

	// 过滤客户端订阅的Tag（完全匹配）
	var pushList []db_point.Update_Value_type
	for _, update := range updateList {
		if exist, ok := client.Points[update.Tag]; ok && exist {
			pushList = append(pushList, update)
		}
	}
	if len(pushList) == 0 {
		return
	}

	// 序列化消息
	msg, err := json.Marshal(pushList)
	if err != nil {
		log.Printf("ERROR 客户端[%p]JSON编码失败: %v", client, err)
		return
	}

	// 安全写入连接
	client.writeMu.Lock()
	defer client.writeMu.Unlock()

	select {
	case <-client.closeCh:
		return
	default:
	}

	client.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	err = client.conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Printf("ERROR 客户端[%p]推送失败: %v", client, err)
		CloseWSClient(client)
	}
}

func api_app_monitor_ws_Push(update_list []db_point.Update_Value_type) error {
	wsRWMu.RLock()

	clients := make([]*WSClient, 0, len(web_socket_clients))
	for client := range web_socket_clients {
		clients = append(clients, client)
	}
	wsRWMu.RUnlock()

	for _, client := range clients {
		select {
		case pushChan <- pushTask{client: client, updateList: update_list}:
		default:
			log.Printf("ERROR 推送队列满，客户端[%p]任务丢弃", client)
		}
	}
	return nil
}

// -------------------------- 5. 客户端管理 --------------------------
func CloseWSClient(client *WSClient) {
	select {
	case <-client.closeCh:
		return
	default:
		close(client.closeCh)
		_ = client.conn.Close()
		wsRWMu.Lock()
		delete(web_socket_clients, client)
		wsRWMu.Unlock()
		log.Printf("ERROR 客户端[%p]已清理", client)
	}
}

func init() {
	db_point.Update_Subscriber(api_app_monitor_ws_Push)
}

func gui_api(r *gin.Engine) {

	InitWSPushPool()

	r.POST("/Gui/v1.0/Login/Name", User_Login_Name)                 // 用户名登陆
	r.POST("/Gui/v1.0/Login/Access_Token", User_Access_Token_query) // 获取访问令牌

	r.POST("/Gui/v1.0/User/Get/Count", User_All_Count)           // 获取条数
	r.POST("/Gui/v1.0/User/Get/Query", User_All_Query)           // 分页查询
	r.POST("/Gui/v1.0/User/Get/Info", User_Get_Info)             // 获取用户信息
	r.POST("/Gui/v1.0/User/Get/Info_Array", User_Get_Info_Array) // 查询多个用户信息
	r.POST("/Gui/v1.0/User/Get/Search", User_get_Info_Search)    // 搜索用户信息

	r.GET("/Gui/v1.0/Monitor/ws", api_app_monitor_ws) // 推送更新值

}
