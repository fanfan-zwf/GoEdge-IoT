package web

import (
	"main/Init"

	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
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
			"Timestamp": time.Now().Format(time.DateTime),
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

// 生成token
// 传参 长度 uint
// 返回 token string
func create_token(Token_length uint) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, Token_length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		b[i] = letters[num.Int64()]
	}

	return fmt.Sprintf("%d%s", time.Now().UnixNano(), string(b)), nil
}

// 全局启用token
// 这里需要注意以下，还需要改正优化
func token_use() gin.HandlerFunc {
	return func(ctx *gin.Context) {

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

func Web() error {

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
	// 前端接口

	// time.Sleep(3 * time.Second)

	go func() {
		err := r.Run(bind)
		if err != nil {
			log.Panic(err.Error())
		}
	}()

	return nil
}

// parseXLSXFromStream 内存解析XLSX文件流（逻辑和之前一致，仅调整返回值适配Gin）
func parseXLSXFromStream(fileStream io.Reader) ([][]string, error) {
	// 直接从io.Reader读取文件流，不落地硬盘
	f, err := excelize.OpenReader(fileStream)
	if err != nil {
		return nil, fmt.Errorf("解析XLSX流失败：%v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("关闭文档失败：%v", err)
		}
	}()

	// 读取Sheet1的所有行数据（返回给前端）
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, fmt.Errorf("获取工作表行数据失败：%v", err)
	}

	return rows, nil
}
