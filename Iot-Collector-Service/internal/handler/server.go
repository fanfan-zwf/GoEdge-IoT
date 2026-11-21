package handler

import (
	"collector/internal/config"

	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
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
		for _, v := range config.Config.API.Header {
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

// guardGoroutine 异步 goroutine 里 defer 它：子 goroutine 的 panic 会直接带走整个采集进程，
// 兜住后降级成一条带堆栈的 ERROR 日志，接口调用方与采集任务不受影响。
func guardGoroutine(what string) {
	if r := recover(); r != nil {
		log.Printf("ERROR 异步任务 %s 发生 panic: %v\n%s", what, r, debug.Stack())
	}
}

func Web() error {
	if !config.Config.API.Enable {
		log.Print("INFO ", "api", "接口服务已禁用")
		return nil
	}

	bind := fmt.Sprintf("%s:%d", config.Config.API.Ip, config.Config.API.Post)

	// gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	// 注册中间件
	r.Use(allowAllCors())    // 跨域问题
	r.Use(Response_Use())    // 全局注册标准响应（在鉴权之后，保证鉴权失败也走标准 JSON）
	r.Use(Auth_Middleware()) // 统一鉴权中间件（Bearer token / HMAC 签名双通路，按路径豁免登录/刷新/内部同步）
	r.Use(Rbac_Middleware()) // RBAC 权限校验（须在鉴权之后，读取主体身份；enable=false 整体跳过，enforce=false 仅观测不拦截）

	gui_api(r)
	// r.Use(static.ServeRoot("/", "../vue"))
	// r.Use(static.ServeRoot("/assets", "../vue/assets"))

	log.Print("INFO ", "api", bind)

	// 不用 r.Run()：它内部直接 http.ListenAndServe，拿不到 server 句柄，
	// 退出时无法 graceful shutdown（在途请求被硬切，也不会主动关监听）。
	webSrv = &http.Server{Addr: bind, Handler: r}

	serveErr := make(chan error, 1)
	go func() {
		err := webSrv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		close(serveErr)
	}()

	// 短暂等待，把端口占用/无权绑定等致命错误同步回传给调用方（旧写法用 log.Panic 直接炸进程）
	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("api 监听失败(%s): %w", bind, err)
		}
	case <-time.After(200 * time.Millisecond):
	}

	return nil
}

// webSrv HTTP 服务句柄，由 Web 建立、WebShutdown 优雅关闭；接口未启用时为 nil。
var webSrv *http.Server

// WebShutdown 优雅停止 HTTP 服务：不再接受新连接，等在途请求完成或 ctx 到期。
func WebShutdown(ctx context.Context) error {
	if webSrv == nil {
		return nil
	}
	return webSrv.Shutdown(ctx)
}

// parseXLSXFromStream 内存解析XLSX文件流（逻辑和之前一致，仅调整返回值适配Gin）
func parseXLSXFromStream(fileStream io.Reader) ([][]string, error) {
	// 直接从io.Reader读取文件流，不落地硬盘
	f, err := excelize.OpenReader(fileStream)
	if err != nil {
		return nil, fmt.Errorf("解析XLSX流失败：%w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("关闭文档失败：%v", err)
		}
	}()

	// 读取Sheet1的所有行数据（返回给前端）
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, fmt.Errorf("获取工作表行数据失败：%w", err)
	}

	return rows, nil
}
