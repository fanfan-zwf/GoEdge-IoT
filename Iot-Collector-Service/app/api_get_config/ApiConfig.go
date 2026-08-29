package apigetconfig

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"main/Init"
	"main/db/mysql"
	"net/http"
	"strings"
	"time"

	"resty.dev/v3"
)

const (
	url_collector_GetBasicAuth        = "/api/app/v1.0/login"                        // 配置登录接口
	url_collector_Drive_Config__Query = "/api/app/v1.0/collector/drive/config/query" // 配置查询驱动接口
	url_collector_Point_Config__Query = "/api/app/v1.0/collector/point/config/query" // 配置查询点位接口
)

// configServiceBaseURL 校验配置服务地址并拼接完整请求 URL
// 自动补全 http:// 协议头；address 为空时返回错误
func configServiceBaseURL(path string) (string, error) {
	addr := Init.Config.Config_Service.Address
	if addr == "" {
		return "", fmt.Errorf("Config_Service.Address 未配置，请检查 config.yaml")
	}
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}
	return strings.TrimRight(addr, "/") + path, nil
}

// 标准响应结构体
type body_struct_Resp[T any] struct {
	Code      int
	Msg       string
	Data      T
	Timestamp time.Time
}

// parseResponseBody 泛型函数：解析标准响应并将 Data 反序列化为指定类型
func parseResponseBody[T any](respBytes []byte) (body_struct_Resp[T], error) {
	var body body_struct_Resp[T]
	if err := json.Unmarshal(respBytes, &body); err != nil {
		return body, fmt.Errorf("解析响应异常: %w", err)
	}
	return body, nil
}

var collector_AccessToken string

/*
 ***********************获取基础认证凭证***********************
 */
func Collector_GetBasicAuth() (string, error) {
	client := resty.New().
		SetTimeout(Init.Config.Config_Service.SetTimeout).            // 设置超时时间
		SetRetryCount(Init.Config.Config_Service.SetRetryCount).      // 401 最多重试 1 次，避免死循环
		SetRetryWaitTime(Init.Config.Config_Service.SetRetryWaitTime) // 设置重试等待时间

	// 拼接完整请求 URL
	reqURL, err := configServiceBaseURL(url_collector_GetBasicAuth)
	if err != nil {
		return "", err
	}

	// 业务发起请求
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Collector_Basic", base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", Init.Config.APP.Label, Init.Config.APP.Passwd))).
		Post(reqURL)

	if err != nil {
		return "", fmt.Errorf("请求异常: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("请求失败，状态码: %d", resp.StatusCode())
	}

	respBody, err := parseResponseBody[string](resp.Bytes())
	if err != nil {
		return "", err
	}

	collector_AccessToken = respBody.Data
	log.Printf("INFO 获取基础认证凭证成功，token: %s\n", collector_AccessToken)
	return collector_AccessToken, nil

}

func Collector_Drive_Config__Query() (data []mysql.CollectorGet_Drive_Config_type, err error) {
	client := resty.New().
		SetTimeout(Init.Config.Config_Service.SetTimeout).            // 设置超时时间
		SetRetryCount(Init.Config.Config_Service.SetRetryCount).      // 401 最多重试 1 次，避免死循环
		SetRetryWaitTime(Init.Config.Config_Service.SetRetryWaitTime) // 设置重试等待时间

	// 1. 每次请求发送前，自动带上 token（包括重试的那次请求）
	client.AddRequestMiddleware(func(c *resty.Client, req *resty.Request) error {
		req.SetHeader("Collector_Token", collector_AccessToken)
		return nil
	})

	// 2. 设置重试条件：当返回 401 时，允许触发重试
	client.AddRetryConditions(func(resp *resty.Response, err error) bool {
		if err != nil {
			return true // 网络异常等也重试
		}
		return resp.StatusCode() == http.StatusUnauthorized
	})

	// 3. 重试前的回调钩子：401 进来，刷新 token
	client.AddRetryHooks(func(resp *resty.Response, err error) {
		log.Printf("INFO 从【配置服务】触发401，执行刷新token =====\n")
		if resp != nil && resp.StatusCode() == http.StatusUnauthorized {
			if _, errRefresh := Collector_GetBasicAuth(); errRefresh != nil {
				log.Printf("ERROR 刷新token失败：%v\n", errRefresh)
			}
		}
	})

	// 拼接完整请求 URL
	reqURL, err := configServiceBaseURL(url_collector_Drive_Config__Query)
	if err != nil {
		return nil, err
	}

	// 业务发起请求
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Post(reqURL)

	if err != nil {
		return nil, fmt.Errorf("驱动配置请求异常: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("驱动配置请求失败，状态码: %d", resp.StatusCode())
	}

	respBody, err := parseResponseBody[[]mysql.CollectorGet_Drive_Config_type](resp.Bytes())
	if err != nil {
		return nil, fmt.Errorf("驱动配置响应解析失败: %w", err)
	}
	return respBody.Data, nil
}

func Collector_Point_Config__Query() (data []mysql.CollectorGet_Point_Config_type, err error) {
	client := resty.New().
		SetTimeout(Init.Config.Config_Service.SetTimeout).            // 设置超时时间
		SetRetryCount(Init.Config.Config_Service.SetRetryCount).      // 401 最多重试 1 次，避免死循环
		SetRetryWaitTime(Init.Config.Config_Service.SetRetryWaitTime) // 设置重试等待时间

	// 1. 每次请求发送前，自动带上 token（包括重试的那次请求）
	client.AddRequestMiddleware(func(c *resty.Client, req *resty.Request) error {
		req.SetHeader("Collector_Token", collector_AccessToken)
		return nil
	})

	// 2. 设置重试条件：当返回 401 时，允许触发重试
	client.AddRetryConditions(func(resp *resty.Response, err error) bool {
		if err != nil {
			return true // 网络异常等也重试
		}
		return resp.StatusCode() == http.StatusUnauthorized
	})

	// 3. 重试前的回调钩子：401 进来，刷新 token
	client.AddRetryHooks(func(resp *resty.Response, err error) {
		log.Printf("INFO 从【配置服务】触发401，执行刷新token =====\n")
		if resp != nil && resp.StatusCode() == http.StatusUnauthorized {
			if _, errRefresh := Collector_GetBasicAuth(); errRefresh != nil {
				log.Printf("ERROR 刷新token失败：%v\n", errRefresh)
			}
		}
	})

	// 拼接完整请求 URL
	reqURL, err := configServiceBaseURL(url_collector_Point_Config__Query)
	if err != nil {
		return nil, err
	}

	// 业务发起请求
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Post(reqURL)

	if err != nil {
		return nil, fmt.Errorf("点位配置请求异常: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("点位配置请求失败，状态码: %d", resp.StatusCode())
	}

	respBody, err := parseResponseBody[[]mysql.CollectorGet_Point_Config_type](resp.Bytes())
	if err != nil {
		return nil, fmt.Errorf("点位配置响应解析失败: %w", err)
	}
	return respBody.Data, nil
}

// refreshToken 刷新 token，替换成真实的刷新 token 逻辑
func refreshToken() error {
	fmt.Println("===== 触发401，执行刷新token =====")
	// TODO: 调用刷新 token 的 HTTP 接口，获取新 access_token
	collector_AccessToken = "new_token_xxxx_xxxx"
	return nil
}

// FetchWithRetry 请求指定 URL，401 时自动刷新 token 并重试 1 次
func FetchWithRetry(url string) (string, error) {
	client := resty.New().
		SetTimeout(Init.Config.Config_Service.SetTimeout).            // 设置超时时间
		SetRetryCount(Init.Config.Config_Service.SetRetryCount).      // 401 最多重试 1 次，避免死循环
		SetRetryWaitTime(Init.Config.Config_Service.SetRetryWaitTime) // 设置重试等待时间

	// 1. 每次请求发送前，自动带上 token（包括重试的那次请求）
	client.AddRequestMiddleware(func(c *resty.Client, req *resty.Request) error {
		req.SetHeader("Authorization", "Bearer "+collector_AccessToken)
		return nil
	})

	// 2. 设置重试条件：当返回 401 时，允许触发重试
	client.AddRetryConditions(func(resp *resty.Response, err error) bool {
		if err != nil {
			return true // 网络异常等也重试
		}
		return resp.StatusCode() == http.StatusUnauthorized
	})

	// 3. 重试前的回调钩子：401 进来，刷新 token
	client.AddRetryHooks(func(resp *resty.Response, err error) {
		if resp != nil && resp.StatusCode() == http.StatusUnauthorized {
			if errRefresh := refreshToken(); errRefresh != nil {
				fmt.Printf("刷新token失败：%v\n", errRefresh)
			}
		}
	})

	// 业务发起请求
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Get(url)

	if err != nil {
		return "", fmt.Errorf("请求异常: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("请求失败，状态码: %d", resp.StatusCode())
	}

	return resp.String(), nil
}

func main() {
	collector_AccessToken = "old_expired_token" // 初始过期 token

	body, err := FetchWithRetry("https://your-api-url.com/test")
	if err != nil {
		panic(err)
	}
	fmt.Println(body)
}
