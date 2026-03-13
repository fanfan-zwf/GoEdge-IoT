package web

import (
	"encoding/base64"
	"main/Init"
	db_redis "main/db/redis"

	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	// _ "github.com/icattlecoder/godaemon"
	// "sync"
)

const (
	Status_No_Login = 401

	StatusMysql = 520 // mysql错误
	StatusRedis = 521 // redis错误
	StatusRegex = 522 // 正则表达式计算错误

)

// 全局注册标准相应
func Response_Use() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		r, exists := ctx.Get("Response")
		if !exists {
			return
		}

		value, ok := r.([]any)
		if !ok {
			ctx.JSON(501, "未知返回")
			return
		}

		code, ok := value[0].(int)
		if !ok {
			ctx.JSON(501, "no code")
			return
		}

		Msg, ok := value[1].(string)
		if !ok {
			ctx.JSON(501, "no Msg")
			return
		}

		var data any
		if len(value) >= 3 {
			data = value[2]
		}

		ctx.JSON(code, gin.H{
			"Code":      code,
			"Msg":       Msg,
			"Data":      data,
			"Timestamp": time.Now().Format(time.RFC3339Nano),
		})
	}
}

// 允许跨域请求
func allowAllCors() gin.HandlerFunc {

	return func(ctx *gin.Context) {
		for _, v := range Init.Config.API.Header {
			ctx.Writer.Header().Set(v.Key, v.Value)
		}
		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(200)
			return
		}
		ctx.Next()
	}
}

// 生成随即盐
// 传参 长度 uint
// 返回 token string
func GenerateSecureRandomString(length int) (string, error) {
	// base64编码后，每4个字符对应3个原始字节，所以先计算需要的原始字节数
	byteLength := (length * 3) / 4
	if (length*3)%4 != 0 {
		byteLength += 1 // 向上取整，保证长度足够
	}

	// 生成加密安全的随机字节
	randomBytes := make([]byte, byteLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("生成随机字节失败: %v", err)
	}

	// 转base64（URL安全编码，避免+、/等特殊字符）
	randomStr := base64.URLEncoding.EncodeToString(randomBytes)

	// 截取到指定长度（避免base64补位的=符号）
	return randomStr[:length], nil
}

// 全局启用token
// 这里需要注意以下，还需要改正优化
func token_use() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var ClientIP = ctx.ClientIP()
		// 1. 先获取路径
		FullPath := ctx.FullPath()
		// 如果 FullPath 为空，可能是路由未匹配，尝试获取请求路径
		if FullPath == "" {
			FullPath = ctx.Request.URL.Path
			return
		}

		// 2. 检查是否是免token的路径
		if strings.HasPrefix(FullPath, "/Gui/v1.0/Login") ||
			strings.HasPrefix(FullPath, "/Api/v1.0/Login") {
			fmt.Printf("路径 %s 无需token授权\n", FullPath)
			ctx.Next()
			return
		}

		if strings.HasPrefix(FullPath, "/Gui/v1.0") {
			// 3. 获取并验证token
			accessToken := ctx.Request.Header.Get("F_Access_Token")
			if accessToken == "" {
				ctx.Set("Response", []any{401, "缺少访问令牌"})
				ctx.Abort()
				return
			}

			Access_Token_redis, err := db_redis.Access_Token_Query(accessToken)
			if err != nil {
				fmt.Print(err, "token无效\n")
			}
			if err == redis.Nil {
				ctx.Set("Response", []any{401, "访问令牌过期或无效"})
				ctx.Abort()
				return
			} else if err != nil {
				ctx.Set("Response", []any{521, err.Error()})
				ctx.Abort()
				return
			}

			if Access_Token_redis.Login_Ip != ClientIP && Access_Token_redis.Login_Ip != "" {
				ctx.Set("Response", []any{403, fmt.Sprintf("ip:%s 禁止请求", ClientIP)})
				return
			}

			ctx.Set("User_Id", Access_Token_redis.User_Id)
			ctx.Set("Access_Token_redis", Access_Token_redis)
			ctx.Next()
		} else if strings.HasPrefix(FullPath, "/Api/v1.0") {
			// 3. 获取并验证token
			accessToken := ctx.Request.Header.Get("F_Api_Access_Token")
			if accessToken == "" {
				ctx.Set("Response", []any{401, "缺少访问令牌"})
				ctx.Abort()
				return
			}

			Access_Token_redis, err := db_redis.Api_Access_Token_Query(accessToken)
			if err == redis.Nil {
				ctx.Set("Response", []any{401, "访问令牌过期或无效"})
				ctx.Abort()
				return
			} else if err != nil {
				ctx.Set("Response", []any{521, err.Error()})
				ctx.Abort()
				return
			}

			if Access_Token_redis.Login_Ip != ClientIP && Access_Token_redis.Login_Ip != "" {
				ctx.Set("Response", []any{403, fmt.Sprintf("ip:%s 禁止请求", ClientIP)})
				return
			}

			ctx.Set("Api_Id", Access_Token_redis.Api_Id)
			ctx.Set("Api_Access_Token_redis", Access_Token_redis)
			ctx.Next()
		} else {
			ctx.Set("Response", []any{404, "未知类型"})
			ctx.Abort()
		}
	}
}

func Token_User_Id(ctx *gin.Context) (User_Id uint, err error) {
	// 用户id不存在，赋值登陆的用户id
	value, exists := ctx.Get("User_Id")
	if !exists {
		err = fmt.Errorf("User_Id 不存在")
		return
	}

	var ok bool
	User_Id, ok = value.(uint)
	if !ok {
		err = fmt.Errorf("User_Id 未知类型")
	}
	return
}

func Web() {

	bind := fmt.Sprintf("%s:%d", Init.Config.API.Ip, Init.Config.API.Post)

	// gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	// 注册中间件
	r.Use(allowAllCors())              // 跨域问题
	r.Use(Response_Use(), token_use()) // 全局启用token验证、全局注册标准相应

	// r.Use(static.ServeRoot("/", "../vue"))
	// r.Use(static.ServeRoot("/assets", "../vue/assets"))

	log.Print("INFO ", "api", bind)
	gui_api(r)
	sdk_api(r)
	// 前端接口

	// time.Sleep(3 * time.Second)

	r.Run(bind)

}
