/*
* 日期: 2026.3.11 PM1:55
* 作者: 范范zwf
* 作用: 用户服务接口
 */

package userservice

import (
	"main/Init"
	r "main/db/redis"
	"sync"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Go 1.21+ 推荐方案：用 sync.Mutex.TryLock + 循环重试（更优雅）
func TryLockWithTimeout(mu *sync.Mutex, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	// 循环尝试加锁，直到超时
	for time.Now().Before(deadline) {
		if mu.TryLock() { // TryLock 非阻塞，成功返回 true，失败返回 false
			return true
		}
		// 短暂休眠，避免CPU空转
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

type User_Body_Standard struct {
	Code      int
	Msg       string
	Timestamp time.Time
}

func api_post(url string, json_payload string) (statusCode int, body string, err error) {

	payload := strings.NewReader(json_payload)

	client := &http.Client{}

	var req *http.Request
	req, err = http.NewRequest("POST", url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	// URL不包含登录相关路径，跳过登录鉴权
	if !strings.Contains(url, "login") {
		var access_token string
		access_token, err = Access_Token_Value()
		if err != nil {
			log.Println("ERROR ", err)
			return
		}
		req.Header.Add("F_Api_Access_Token", access_token)
	}

	var res *http.Response
	res, err = client.Do(req)
	if err != nil {
		log.Println("ERROR ", err)
		return
	}
	defer res.Body.Close()

	statusCode = res.StatusCode
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d", res.StatusCode)
		log.Print(err)
		return
	}

	var body_byte []byte
	body_byte, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("ERROR ", err)
		return
	}

	body = string(body_byte)
	return
}

// IsWithin10Seconds 判断目标时间是否在最近10秒内
// targetTime: 要判断的时间
// 返回值: true=在10秒内，false=超出10秒
func IsWithin10Seconds(targetTime time.Time) bool {
	// 1. 获取当前时间
	now := time.Now()

	// 2. 计算时间差（当前时间 - 目标时间）
	duration := now.Sub(targetTime)

	// 3. 判断差值是否≤10秒（注意：如果targetTime是未来时间，duration为负，也返回false）
	// 10秒的常量写法：10 * time.Second
	return duration <= 10*time.Second && duration >= -10*time.Second
}

// 用户服务接口鉴权
type Refresh_Token_Get_type struct {
	ApiKey string
	Secret string
}

type Refresh_Token_type struct {
	Api_Id              uint
	F_Api_Refresh_Token string
	F_Api_Expires_in    time.Time
}

type Refresh_Token_Body_type struct {
	User_Body_Standard
	Data Refresh_Token_type
}

// 获取刷新令牌
func Get_Login_Refresh_Token(reqData Refresh_Token_Get_type) (refresh Refresh_Token_type, err error) {
	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	url := fmt.Sprintf("%s/api/v1.0/login/refresh_token", Init.Config.User_Service.Url)
	code, body, err := api_post(url, string(jsonBytes))
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据
	var response Refresh_Token_Body_type
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s", code, response.Msg)
		log.Print(err)
		return
	}

	if IsWithin10Seconds(response.Timestamp) {
		err = fmt.Errorf("ERROR API请求失败，请求超时，Expires_in: %s", response.Timestamp.Format(time.RFC3339Nano))
		log.Print(err)
		return
	}

	var refresh_jsonBytes []byte
	refresh_jsonBytes, err = json.Marshal(refresh)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	err = r.Write_Key_list([]r.KeyValue{
		{
			Key:   "user_service_refresh_token",
			Value: string(refresh_jsonBytes),
			TTL:   time.Until(refresh.F_Api_Expires_in),
		},
	})
	if err != nil {
		log.Println("ERROR Redis写入失败：", err)
		return
	}

	refresh = response.Data
	return
}

// 获取刷新令牌信息
func Refresh_Token_Info() (refresh Refresh_Token_type, err error) {
	Info, err := r.Read_Key("user_service_refresh_token")

	if err == redis.Nil || Info == "" {
		log.Print("warning Redis读取失败：没有找到访问令牌")
		refresh, err = Get_Login_Refresh_Token(Refresh_Token_Get_type{
			ApiKey: Init.Config.User_Service.ApiKey,
			Secret: Init.Config.User_Service.Secret,
		})
		if err != nil {
			log.Println("ERROR 获取刷新令牌失败：", err)
			return
		}
		return
	} else if err != nil {
		log.Println("ERROR Redis读取失败：", err)
		return
	}
	err = json.Unmarshal([]byte(Info), &refresh)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	return
}

// 用户服务接口鉴权
type Access_Token_Get_type struct {
	Api_Id              uint
	F_Api_Refresh_Token string
}

type Access_Token_type struct {
	Api_Id             int
	F_Api_Access_Token string
	F_Api_Expires_in   time.Time
}

type Access_Token_Body_type struct {
	User_Body_Standard
	Data Access_Token_type
}

// 获取刷新令牌
func Get_Access_Token(reqData Access_Token_Get_type) (access Access_Token_type, err error) {
	// 2. 将结构体序列化为JSON字节数组（核心步骤）
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	url := fmt.Sprintf("%s/api/v1.0/login/access_token", Init.Config.User_Service.Url)
	code, body, err := api_post(url, string(jsonBytes))
	if err != nil {
		log.Println("ERROR API请求失败：", err)
		return
	}

	// 3. 解析响应数据
	var response Access_Token_Body_type
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s", code, response.Msg)
		log.Print(err)
		return
	}

	if IsWithin10Seconds(response.Timestamp) {
		err = fmt.Errorf("ERROR API请求失败，请求超时，Expires_in: %s", response.Timestamp.Format(time.RFC3339Nano))
		log.Print(err)
		return
	}

	var access_jsonBytes []byte
	access_jsonBytes, err = json.Marshal(access)
	if err != nil {
		log.Println("ERROR JSON序列化失败：", err)
		return
	}

	err = r.Write_Key_list([]r.KeyValue{
		{
			Key:   "user_service_access_token",
			Value: string(access_jsonBytes),
			TTL:   time.Until(access.F_Api_Expires_in),
		},
	})
	if err != nil {
		log.Println("ERROR Redis写入失败：", err)
		return
	}

	access = response.Data

	return
}

var Access_Token_Info_Mu sync.Mutex

// 获取访问令牌信息
func Access_Token_Info() (access Access_Token_type, err error) {

	if !TryLockWithTimeout(&Access_Token_Info_Mu, 10*time.Second) {
		err = fmt.Errorf("ERROR 尝试 10 秒超时获取锁")
		return
	} else {
		defer Access_Token_Info_Mu.Unlock()
	}

	Info, err := r.Read_Key("user_service_access_token")

	if err == redis.Nil || Info == "" {
		log.Print("warning Redis读取失败：没有找到访问令牌")
		var refresh Refresh_Token_type
		refresh, err = Refresh_Token_Info()
		if err != nil {
			log.Println("ERROR 获取刷新令牌失败：", err)
			return
		}

		access, err = Get_Access_Token(Access_Token_Get_type{
			Api_Id:              refresh.Api_Id,
			F_Api_Refresh_Token: refresh.F_Api_Refresh_Token,
		})
		if err != nil {
			log.Println("ERROR 获取访问令牌失败：", err)
			return
		}
		return
	} else if err != nil {
		log.Println("ERROR Redis读取失败：", err)
		return
	}
	err = json.Unmarshal([]byte(Info), &access)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	return
}

// 获取访问令牌值
func Access_Token_Value() (access_token string, err error) {
	var access Access_Token_type
	access, err = Access_Token_Info()
	if err != nil {
		log.Println("ERROR 获取访问令牌失败：", err)
		return
	}
	access_token = access.F_Api_Access_Token

	return
}

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
func User_Authority_Exist(reqData User_Authority_Exist__Get_type) (exist bool, err error) {

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

	if code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s", code, response.Msg)
		log.Print(err)
		return
	}

	if IsWithin10Seconds(response.Timestamp) {
		err = fmt.Errorf("ERROR API请求失败，请求超时，Expires_in: %s", response.Timestamp.Format(time.RFC3339Nano))
		log.Print(err)
		return
	}

	return
}

/*
******************查询当前用户登陆状态******************
 */

type User_status__Get_type struct {
	User__Access_Token string // 用户刷新令牌
}

type User_status__type struct {
	Code int    // 执行码
	Msg  string // 执行说明

	User_Id       uint      // 用户id
	Expires_in    time.Time // 访问令牌过期时间
	Refresh_Token string    // 本访问令牌的刷新令牌
}

type User_status__Byte_type struct {
	User_Body_Standard
	Data User_status__type
}

// 用户权限查询
//
//	输入：User__Access_Token用户刷新令牌
func User_status(User__Access_Token string) (exist bool, err error) {
	reqData := User_status__Get_type{
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

	if code != 200 {
		err = fmt.Errorf("ERROR API请求失败，状态码: %d, 消息: %s", code, response.Msg)
		log.Print(err)
		return
	}

	if IsWithin10Seconds(response.Timestamp) {
		err = fmt.Errorf("ERROR API请求失败，请求超时，Expires_in: %s", response.Timestamp.Format(time.RFC3339Nano))
		log.Print(err)
		return
	}

	return
}
