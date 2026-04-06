/*
* 日期: 2026.3.11 PM1:55
* 作者: 范范zwf
* 作用: 用户服务登陆接口
 */

package user_service

import (
	p_config "main/Init"
	p_redis "main/db/redis"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

// 业务通用响应（可自行修改）
type Body_Standard struct {
	Code      int
	Msg       string
	Timestamp time.Time
}

// APIClient 封装双Token客户端
type APIClient_struct struct {
	Url    string
	ApiKey string
	Secret string
	mu     sync.Mutex // 防止并发重复刷新Token

	client *http.Client
}

/* 定义接口 */
type APIClient_interfaces interface {
	// 刷新令牌请求
	Refresh_RefreshToken() (err error)
	// 存储刷新令牌
	Store_RefreshToken(data Authentication_Refresh_Token_type) (err error)
	// 获取刷新令牌
	Get_RefreshToken() (r Authentication_Refresh_Token_type, err error)

	// 请求令牌请求
	Refresh_AccessToken() (err error)
	// 存储访问令牌
	Store_AccessToken(data Authentication_Access_Token_Response_type) (err error)
	// 获取访问令牌
	Get_AccessToken() (r Authentication_Access_Token_Response_type, err error)

	// 发送请求，自动处理Token过期
	Request(endpoint string, header map[string]string, body []byte) (response http.Response, err error)
}

// 创建APIClient
func NewAPIClient(
	Url string,
	ApiKey string,
	Secret string,
	Timeout time.Duration,
) *APIClient_struct {
	return &APIClient_struct{
		Url:    Url,
		ApiKey: ApiKey,
		Secret: Secret,
		mu:     sync.Mutex{},
		client: &http.Client{Timeout: Timeout},
	}
}

// 发送刷新令牌请求

// Access_Token 请求体
type Authentication_Refresh_Token_Request_type struct {
	ApiKey string
	Secret string
}

// Access_Token 返回内容
type Authentication_Refresh_Token_type struct {
	Api_Id              uint
	F_Api_Refresh_Token string
	F_Api_Expires_in    time.Time
}

// 刷新令牌响应体
type Authentication_Refresh_Token_Response_type struct {
	Body_Standard
	Data Authentication_Refresh_Token_type
}

func (c *APIClient_struct) Refresh_RefreshToken() (data Authentication_Refresh_Token_Response_type, err error) {
	// 构造刷新请求体（你要求的 4 个字段）

	reqBody := Authentication_Refresh_Token_Request_type{
		ApiKey: c.ApiKey,
		Secret: c.Secret,
	}

	var jsonData []byte
	jsonData, err = json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf("ERROR JSON编码错误: %v", err)
		log.Print(err)
		return
	}
	path := "/api/v1.0/login/refresh_token"

	var (
		code      int
		data_type []byte
	)
	code, data_type, err = c.DoRequest("POST", path, jsonData, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		err = fmt.Errorf("ERROR 刷新Token请求失败: %v", err)
		log.Print(err)
		return
	}

	err = json.Unmarshal(data_type, &data)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 || data.Code != 200 {
		err = fmt.Errorf("ERROR 刷新Token失败, 消息: %s", data.Msg)
		log.Print(err)
		return
	}

	err = c.Store_RefreshToken(data.Data)
	if err != nil {
		err = fmt.Errorf("ERROR 存储刷新Token失败: %v", err)
		log.Print(err)
	}

	return
}

// 存储刷新令牌
func (c *APIClient_struct) Store_RefreshToken(data Authentication_Refresh_Token_type) (err error) {
	var jsonData []byte
	jsonData, err = json.Marshal(data)
	if err != nil {
		err = fmt.Errorf("ERROR JSON编码错误: %v", err)
		log.Print(err)
		return
	}

	err = p_redis.Write_Key_list(p_redis.KeyValue{
		Key:   "F_Api_Refresh_Token",
		Value: string(jsonData),
		TTL:   time.Until(data.F_Api_Expires_in),
	})
	if err != nil {
		err = fmt.Errorf("ERROR Redis写入失败：%v", err)
		log.Print(err)
	}
	return

}

// 获取刷新令牌
func (c *APIClient_struct) Get_RefreshToken() (r Authentication_Refresh_Token_type, err error) {
	var value string
	value, err = p_redis.Read_Key("F_Api_Refresh_Token")
	if err == p_redis.Nil {
		var refresh Authentication_Refresh_Token_Response_type
		refresh, err = c.Refresh_RefreshToken()
		r = refresh.Data
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR Redis读取失败：%v", err)
		log.Print(err)
		return
	}
	err = json.Unmarshal([]byte(value), &r)
	if err != nil {
		err = fmt.Errorf("ERROR JSON解析失败：%v", err)
		log.Print(err)
		return
	}
	return
}

// 获取访问令牌
type Authentication_Access_Request_type struct {
	Api_Id              uint
	F_Api_Refresh_Token string
}

// 访问令牌返回内容
type Authentication_Access_Token_type struct {
	F_Api_Access_Token string
	F_Api_Expires_in   time.Time
}

// 访问令牌响应体
type Authentication_Access_Token_Response_type struct {
	Body_Standard
	Data Authentication_Access_Token_type
}

func (c *APIClient_struct) Refresh_AccessToken() (data Authentication_Access_Token_Response_type, err error) {
	var Refresh Authentication_Refresh_Token_type
	Refresh, err = c.Get_RefreshToken()
	if err != nil {
		log.Print(err)
		return
	}

	reqBody := Authentication_Access_Request_type{
		Api_Id:              Refresh.Api_Id,
		F_Api_Refresh_Token: Refresh.F_Api_Refresh_Token,
	}

	var jsonData []byte
	jsonData, err = json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf("ERROR JSON编码错误: %v", err)
		log.Print(err)
		return
	}

	path := "/api/v1.0/login/access_token"

	var (
		code      int
		data_type []byte
	)

	code, data_type, err = c.DoRequest("POST", path, jsonData, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		err = fmt.Errorf("ERROR 刷新Token请求失败: %v", err)
		log.Print(err)
		return
	}

	err = json.Unmarshal(data_type, &data)
	if err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	if code != 200 || data.Code != 200 {
		err = fmt.Errorf("ERROR 刷新Token失败, 消息: %s", data.Msg)
		log.Print(err)
		return
	}

	err = c.Store_AccessToken(data.Data)
	if err != nil {
		err = fmt.Errorf("ERROR 存储刷新Token失败: %v", err)
		log.Print(err)
	}

	return
}

func (c *APIClient_struct) Store_AccessToken(data Authentication_Access_Token_type) (err error) {
	var jsonData []byte
	jsonData, err = json.Marshal(data)
	if err != nil {
		err = fmt.Errorf("ERROR JSON编码错误: %v", err)
		log.Print(err)
		return
	}

	err = p_redis.Write_Key_list(p_redis.KeyValue{
		Key:   "F_Api_Access_Token",
		Value: string(jsonData),
		TTL:   time.Until(data.F_Api_Expires_in),
	})
	if err != nil {
		err = fmt.Errorf("ERROR Redis写入失败：%v", err)
		log.Print(err)
	}
	return
}

func (c *APIClient_struct) Get_AccessToken() (r Authentication_Access_Token_type, err error) {
	var value string
	value, err = p_redis.Read_Key("F_Api_Access_Token")
	if err == p_redis.Nil {
		var access Authentication_Access_Token_Response_type
		access, err = c.Refresh_AccessToken()
		r = access.Data
		return
	} else if err != nil {
		err = fmt.Errorf("ERROR Redis读取失败：%v", err)
		log.Print(err)
		return
	}
	err = json.Unmarshal([]byte(value), &r)
	if err != nil {
		err = fmt.Errorf("ERROR JSON解析失败：%v", err)
		log.Print(err)
		return
	}
	return
}
func (c *APIClient_struct) DoRequest(method string, path string, body []byte, header map[string]string) (code int, data []byte, err error) {

	var req *http.Request
	url := fmt.Sprintf("%s%s", c.Url, path)

	req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		err = fmt.Errorf("ERROR 创建请求失败: %v", err)
		log.Print(err)
		return
	}

	for k, v := range header {
		req.Header.Add(k, v)
	}

	var res *http.Response
	res, err = c.client.Do(req)
	if err != nil {
		// 修复: 当 err != nil 时，res 可能为 nil，直接访问 res.StatusCode 会导致 panic
		err = fmt.Errorf("ERROR API请求失败，Url: %s, 错误: %v", url, err)
		fmt.Print(c.Url, path, "=========\n")
		log.Print(err)
		return
	}
	defer res.Body.Close()

	code = res.StatusCode
	data, err = ioutil.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("ERROR 读取响应失败, path:%s, err:%v", path, err)
		log.Print(err)
		return
	}

	return
}
func (c *APIClient_struct) Request(path string, body []byte) (code int, data []byte, err error) {

	var r Authentication_Access_Token_type
	r, err = c.Get_AccessToken()

	code, data, err = c.DoRequest("POST", path, body, map[string]string{
		"Content-Type":       "application/json",
		"F_Api_Access_Token": r.F_Api_Access_Token,
	})
	if err != nil {
		err = fmt.Errorf("ERROR API请求失败, path:%s, err:%v", path, err)
		log.Print(err)
		return
	}
	return
}

var client *APIClient_struct

func New() {
	client = NewAPIClient(
		p_config.Config.User_Service.Url,
		p_config.Config.User_Service.ApiKey,
		p_config.Config.User_Service.Secret,
		p_config.Config.User_Service.Timeout,
	)
}
