/*
* 日期: 2026.3.24 PM8:49
* 作者: 范范zwf
* 作用: 用户服务查询模块
 */

package user_service

import (
	r "main/db/redis"

	"encoding/json"
	"fmt"
	"log"
	"time"
)

/*
******************查询当前用户登陆状态******************
* url: /api/v1.0/user/login/status
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
	// User__Id           uint
	User__Access_Token string // 用户刷新令牌
}

type User_Status_BodyData_type struct {
	Code int    // 执行码
	Msg  string // 执行说明

	User_Access_Token       string // 访问令牌
	User_Access_Token_redis Access_Token_redis
}

type User_status__Body_type struct {
	Body_Standard
	Data User_Status_BodyData_type
}

// 用户权限查询
//
//	输入：User__Access_Token用户刷新令牌
func Api__User_Status(User_Access_Token string) (r Access_Token_redis, err error) {
	reqData := User_Status__Get_type{
		User__Access_Token: User_Access_Token,
	}

	var jsonBytes []byte
	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err = json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}
	var (
		code int
		body []byte
	)
	code, body, err = client.Request("/api/v1.0/user/login/status", jsonBytes)
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据Read_Cache_User_status
	var response User_status__Body_type
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 || response.Code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s 请求: %s 响应: %s", code, response.Msg, string(jsonBytes), string(body))
		log.Print(err)
		return
	}

	r = response.Data.User_Access_Token_redis
	return
}

// 缓存用户状态
func Cache_User_status(User_Access_Token string) (result Access_Token_redis, err error) {
	redis_key := fmt.Sprintf("user_status:%s", User_Access_Token)
	var read_key string

	// 使用包别名 r 调用 Read_Key
	read_key, err = r.Read_Key(redis_key)

	// 修复点：此时 r 指代包，r.Nil 指代 redis.Nil
	if err == r.Nil {
		var access_token Access_Token_redis
		access_token, err = Api__User_Status(User_Access_Token)
		if err != nil {
			log.Println("ERROR API 请求失败：", err)
			return
		}

		var jsonBytes []byte
		jsonBytes, err = json.Marshal(access_token)
		if err != nil {
			log.Println("ERROR JSON 序列化失败：", err)
			return
		}

		ttlDuration := time.Until(access_token.Expires_in)
		if ttlDuration <= 0 {
			log.Printf("WARN: 令牌已过期 Token: %s", User_Access_Token)
			err = r.Nil
			return
		}

		err = r.Write_Key_list(r.KeyValue{
			Key:   redis_key,
			Value: string(jsonBytes),
			TTL:   ttlDuration,
		})
		if err != nil {
			return
		}

		// 赋值给返回值 result
		result = access_token
	} else if err != nil {
		log.Println("ERROR 读取缓存失败：", err)
		return
	} else {
		err = json.Unmarshal([]byte(read_key), &result)
		if err != nil {
			log.Println("ERROR JSON 解析失败：", err)
		}
	}

	return
}

/*
******************接口登陆状态查询******************
* url: /api/v1.0/api/login/status
 */
type Api_Access_Token_redis_type struct {
	Api_Id        uint      // 用户id
	Expires_in    time.Time // 访问令牌过期时间
	Refresh_Token string    // 本访问令牌的刷新令牌
	Allow_Ip      string    // 允许ip
	Login_Ip      string    // 登录ip

	Salt            string // 随机盐
	RSA_Private_Key string // RSA私钥
	RSA_Public_Key  string // RSA公钥
}

type Api_Status__Get_type struct {
	Api_Access_Token string // 权限主题
}

type Api_Status__type struct {
	Code                   uint
	Msg                    string
	Api_Access_Token       string
	Api_Access_Token_redis Api_Access_Token_redis_type
}

type Api_Status__Byte_type struct {
	Body_Standard
	Data Api_Status__type
}

// 用户权限查询
func Api__Api_Status(Api_Access_Token string) (r Api_Status__type, err error) {
	reqData := Api_Status__Get_type{
		Api_Access_Token: Api_Access_Token,
	}

	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	code, body, err := client.Request("/api/v1.0/api/login/status", jsonBytes)
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据
	var response Api_Status__Byte_type
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 || response.Code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s", code, response.Msg)
		log.Print(err)
		return
	}

	r = response.Data
	return
}

// 缓存接口状态
func Cache_Api_status(Api_Access_Token string) (result Api_Access_Token_redis_type, err error) {
	redis_key := fmt.Sprintf("Api_Status:%s", Api_Access_Token)
	var read_key string
	read_key, err = r.Read_Key(redis_key)

	// 修复点：同上
	if err == r.Nil {
		var api_status Api_Status__type
		api_status, err = Api__Api_Status(Api_Access_Token)
		if err != nil {
			log.Println("ERROR API 请求失败：", err)
			return
		}

		if api_status.Code == 401 {
			err = r.Nil
			return
		} else if !(api_status.Code >= 200 && api_status.Code < 300) {
			err = fmt.Errorf("%s", api_status.Msg)
			return
		}

		var jsonBytes []byte
		jsonBytes, err = json.Marshal(api_status)
		if err != nil {
			log.Println("ERROR JSON 序列化失败：", err)
			return
		}

		ttlDuration := time.Until(api_status.Api_Access_Token_redis.Expires_in)
		if ttlDuration <= 0 {
			log.Printf("WARN: 令牌已过期 Token: %s", api_status.Api_Access_Token)
			err = r.Nil
			return
		}

		err = r.Write_Key_list(r.KeyValue{
			Key:   redis_key,
			Value: string(jsonBytes),
			TTL:   ttlDuration,
		})
		if err != nil {
			return
		}

		result = api_status.Api_Access_Token_redis
	} else if err != nil {
		log.Println("ERROR 读取缓存失败：", err)
		return
	} else {
		err = json.Unmarshal([]byte(read_key), &result)
		if err != nil {
			log.Println("ERROR JSON 解析失败：", err)
		}
	}

	return
}

/*
******************查询当前用户是否有这个权限******************
* url: /api/v1.0/user/authority
 */

type Api_User_Authority_Exist__Get_type struct {
	User__Access_Token string // 用户刷新令牌
	Authority_Theme    string // 权限主题
}

type Api_User_Authority_Exist__type struct {
	Authority_Exist    bool
	User__Access_Token string
	Authority_Theme    string
}

type Api_User_Authority_Exist__Byte_type struct {
	Body_Standard
	Data Api_User_Authority_Exist__type
}

func Api_User_Authority_Exist(reqData Api_User_Authority_Exist__Get_type) (r Api_User_Authority_Exist__type, err error) {

	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	code, body, err := client.Request("/api/v1.0/user/authority", jsonBytes)
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据
	var response Api_User_Authority_Exist__Byte_type
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 || response.Code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s", code, response.Msg)
		log.Print(err)
		return
	}

	r = response.Data
	return
}

// 缓存用户权限状态
func Cache_User_Authority_status(reqData Api_User_Authority_Exist__Get_type) (result Api_User_Authority_Exist__type, err error) {
	redis_key := fmt.Sprintf("User_Authority_Exist:%s:%s", reqData.User__Access_Token, reqData.Authority_Theme)

	var read_key string
	read_key, err = r.Read_Key(redis_key)
	if err == r.Nil {
		var (
			middle_TTL time.Duration
		)
		result, err = Api_User_Authority_Exist(reqData)
		if err != nil {
			log.Println("ERROR API 请求失败：", err)
			middle_TTL = 30 * time.Second
		} else {
			middle_TTL = 10 * time.Minute
		}

		var jsonBytes []byte
		jsonBytes, err = json.Marshal(result)
		if err != nil {
			log.Println("ERROR JSON 序列化失败：", err)
			return
		}

		err = r.Write_Key_list(r.KeyValue{
			Key:   redis_key,
			Value: string(jsonBytes),
			TTL:   middle_TTL,
		})
		if err != nil {
			return
		}

	} else if err != nil {
		log.Println("ERROR 读取缓存失败：", err)
		return
	}

	err = json.Unmarshal([]byte(read_key), &result)
	if err != nil {
		log.Println("ERROR JSON 解析失败：", err)
	}

	return
}
