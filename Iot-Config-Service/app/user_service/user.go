/*
* 日期: 2026.3.22 PM11:21
* 作者: 范范zwf
* 作用: 用户服务查询模块
 */

package userservice

import (
	"main/Init"
	m_redis "main/db/redis"

	"encoding/json"
	"fmt"
	"log"
	"time"
)

/*
******************用户权限查询******************
 */

// 用户服务接口鉴权
type User_Authority_Exist__Get_type struct {
	User__Access_Token string // 用户刷新令牌
	Authority_Theme    string // 权限主题
}

type User_Authority_Exist__type struct {
	Authority_Exist    bool
	User__Access_Token string
	Authority_Theme    string
}

type User_Authority_Exist__Byte_type struct {
	User_Body_Standard
	Data User_Authority_Exist__type
}

// 用户权限查询
func User_Authority_Exist(reqData User_Authority_Exist__Get_type) (r User_Authority_Exist__type, err error) {

	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	url := fmt.Sprintf("%s/api/v1.0/user/authority", Init.Config.User_Service.Url)
	code, body, err := api_post(url, string(jsonBytes))
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据
	var response User_Authority_Exist__Byte_type
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 || response.Code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s", code, response.Msg)
		log.Print(err)
		return
	}

	if IsWithin(response.Timestamp, 10*time.Second) {
		err = fmt.Errorf("ERROR API请求失败，请求超时，Expires_in: %s", response.Timestamp.Format(time.RFC3339Nano))
		log.Print(err)
		return
	}

	r = response.Data
	return
}

/*
******************查询当前用户登陆状态******************
 */

type Access_Token_redis struct {
	User_Id       uint      // 用户id
	Expires_in    time.Time // 访问令牌过期时间
	Refresh_Token string    // 本访问令牌的刷新令牌
	Login_Ip      string    // 登录ip

	Salt            string // 随机盐
	RSA_Private_Key string // RSA私钥
	RSA_Public_Key  string // RSA公钥
}

type User_Status__Get_type struct {
	User_Id            uint
	User__Access_Token string // 用户刷新令牌
}

type User_Status__type struct {
	Code int    // 执行码
	Msg  string // 执行说明

	User__Access_Token string // 访问令牌
	Access_Token_redis Access_Token_redis
}

type User_status__Byte_type struct {
	User_Body_Standard
	Data User_Status__type
}

// 用户权限查询
//
//	输入：User__Access_Token用户刷新令牌
func User_Status(User__Access_Token string) (r User_Status__type, err error) {
	reqData := User_Status__Get_type{
		User__Access_Token: User__Access_Token,
	}

	var jsonBytes []byte
	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err = json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	url := fmt.Sprintf("%s/api/v1.0/user/login/status", Init.Config.User_Service.Url)
	code, body, err := api_post(url, string(jsonBytes))
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据
	var response User_status__Byte_type
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 || response.Code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s", code, response.Msg)
		log.Print(err)
		return
	}

	if IsWithin(response.Timestamp, 10*time.Second) {
		err = fmt.Errorf("ERROR API请求失败，请求超时，Expires_in: %s", response.Timestamp.Format(time.RFC3339Nano))
		log.Print(err)
		return
	}

	r = response.Data
	return
}

// 缓存用户登陆状态
func Read_Cache_User_status(User__Access_Token string) (r Access_Token_redis, err error) {
	var read_key string
	read_key, err = m_redis.Read_Key(User__Access_Token)
	if err == m_redis.Nil {
		var user_status User_Status__type
		user_status, err = User_Status(User__Access_Token)
		if err != nil {
			log.Println("ERROR API请求失败：", err)
			return
		}

		var jsonBytes []byte
		jsonBytes, err = json.Marshal(user_status)
		if err != nil {
			log.Println("ERROR JSON序列化失败：", err)
			return
		}

		ttlDuration := time.Until(user_status.Access_Token_redis.Expires_in)
		if ttlDuration <= 0 {
			log.Printf("WARN: 令牌已过期 Token: %s", User__Access_Token)
			return
		}

		err = m_redis.Write_Key_list(m_redis.KeyValue{
			Key:   "User__Access_Token", // 对应Redis的key
			Value: string(jsonBytes),    // 对应Redis的value（已转成字符串）
			TTL:   ttlDuration,          // 可选：单个key的过期时间
		})
		r = user_status.Access_Token_redis
	} else if err != nil {
		log.Println("ERROR 读取缓存失败：", err)
		return
	} else {
		err = json.Unmarshal([]byte(read_key), &r)
		if err != nil {
			log.Println("ERROR JSON解析失败：", err)
		}
	}

	return
}
