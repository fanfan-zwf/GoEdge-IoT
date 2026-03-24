/*
* 日期: 2026.3.24 PM8:49
* 作者: 范范zwf
* 作用: 用户服务查询模块
 */

package user_service

import (
	"encoding/json"
	"fmt"
	"log"
	"main/Init"
	"time"
)

/*
******************用户权限查询******************
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

type Api__Api_Status__Get_type struct {
	Api_Id           string // 用户刷新令牌
	Api_Access_Token string // 权限主题
}

type Api__Api_Status__type struct {
	Code                   uint
	Msg                    string
	Api_Access_Token       string
	Api_Access_Token_redis Api_Access_Token_redis_type
}

type Api__Api_Status__Byte_type struct {
	Body_Standard
	Data Api__Api_Status__type
}

// 用户权限查询
func Api__Api_Status(reqData Api__Api_Status__Get_type) (r Api__Api_Status__type, err error) {

	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	url := fmt.Sprintf("%s/api/v1.0/api/login/status", Init.Config.User_Service.Url)
	code, body, err := api_post(url, string(jsonBytes))
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据
	var response Api__Api_Status__Byte_type
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
******************查询当前用户是否有这个权限******************
* url: /api/v1.0/user/authority
 */

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
	Body_Standard
	Data User_Authority_Exist__type
}

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
******************查询当前接口登陆状态******************
* url: /api/v1.0/api/login/status
 */

type Api_Access_Token_redis struct {
	Api_Id        uint      // 用户id
	Expires_in    time.Time // 访问令牌过期时间
	Refresh_Token string    // 本访问令牌的刷新令牌
	Login_Ip      string    // 登录ip

	Salt            string // 随机盐
	RSA_Private_Key string // RSA私钥
	RSA_Public_Key  string // RSA公钥
}

type Api_Status__Get_type struct {
	Api_Id           uint
	Api_Access_Token string // 用户刷新令牌
}

type Api_Status__type struct {
	Code int    // 执行码
	Msg  string // 执行说明

	Api_Access_Token       string // 访问令牌
	Api_Access_Token_redis Api_Access_Token_redis
}

type Api_status__Byte_type struct {
	Body_Standard
	Data Api_Status__type
}

// 输入：Api__Access_Token用户刷新令牌
func Api_Status(Api_Access_Token string) (r Api_Status__type, err error) {
	reqData := Api_Status__Get_type{
		Api_Access_Token: Api_Access_Token,
	}

	var jsonBytes []byte
	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err = json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	url := fmt.Sprintf("%s/api/v1.0/api/login/status", Init.Config.User_Service.Url)
	code, body, err := api_post(url, string(jsonBytes))
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据
	var response Api_status__Byte_type
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
