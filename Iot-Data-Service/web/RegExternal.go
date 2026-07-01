package web

import (
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HTTP 方法常量定义
const (
	Method_GET    = "GET"    // GET 请求方法
	Method_POST   = "POST"   // POST 请求方法
	Method_PUT    = "PUT"    // PUT 请求方法
	Method_DELETE = "DELETE" // DELETE 请求方法
)

// ==================== 完整的API注册系统（支持动态注册/注销） ====================

// APIRegister API 注册信息结构体
// 用于存储每个 API 接口的元数据和状态信息
type APIRegister struct {
	Method   string          // HTTP 请求方法（GET/POST/PUT/DELETE）
	Handler  gin.HandlerFunc // Gin 框架的请求处理函数
	IsActive bool            // 接口激活状态标记，true=激活，false=禁用（返回404）
}

var (
	// apiRegistries API 注册表
	// key: API 路径（如 "/api/user"）
	// value: APIRegister 指针，使用指针避免结构体拷贝导致的并发安全问题
	apiRegistries = make(map[string]*APIRegister)
	
	// registryMutex 读写锁，保护 apiRegistries 的并发访问
	// RLock: 读操作（查询、获取handler）可以并发执行
	// Lock: 写操作（注册、注销、更新）需要独占访问
	registryMutex sync.RWMutex
	
	// globalRouter 全局 Gin 引擎引用，用于动态注册路由
	// 在 Web() 函数中通过 SetGlobalRouter() 设置
	globalRouter *gin.Engine
)

// SetGlobalRouter 设置全局 Gin 引擎引用
// 
// 参数:
//   - router: Gin 引擎实例
//
// 功能:
//   - 保存 Gin 引擎引用，供动态路由注册使用
//   - 必须在 Web() 函数中调用，在 ExecuteRegistrations() 之前
//
// 示例:
//   SetGlobalRouter(R)
//   ExecuteRegistrations(R)
func SetGlobalRouter(router *gin.Engine) {
	globalRouter = router
	log.Printf("INFO 全局 Gin 引擎已设置")
}

// notFoundHandler 404 未找到处理器
// 当接口被注销、禁用或不存在时调用
// 通过 ctx.Set("Response", ...) 设置响应数据，由中间件统一处理响应格式
func notFoundHandler(ctx *gin.Context) {
	ctx.Set("Response", []any{404, "接口不存在或已禁用"})
}

// defaultHandler 默认处理器（501 未实现）
// 当注册的 handler 为 nil 时使用此默认处理器
// 通过 ctx.Set("Response", ...) 设置响应数据，由中间件统一处理响应格式
func defaultHandler(ctx *gin.Context) {
	ctx.Set("Response", []any{500, "接口未实现"})
}

// RegisterAPI 注册 API 接口（支持动态注册）
// 
// 参数:
//   - method: HTTP 请求方法（GET/POST/PUT/DELETE），使用 Method_* 常量
//   - url: API 路径（如 "/api/user/list"）
//   - handler: Gin 请求处理函数，如果为 nil 则自动使用 defaultHandler
//
// 功能:
//   - 将 API 信息保存到全局注册表 apiRegistries
//   - 默认设置为激活状态（IsActive=true）
//   - 线程安全，使用互斥锁保护
//   - **关键特性**: 如果 globalRouter 已设置，会立即注册到 Gin 引擎，实现真正的动态注册
//
// 示例:
//   RegisterAPI(Method_POST, "/api/data", func(c *gin.Context) { ... })
func RegisterAPI(method, url string, handler gin.HandlerFunc) {
	registryMutex.Lock()       // 加写锁，独占访问
	defer registryMutex.Unlock() // 函数退出时释放锁

	// 安全检查：如果 handler 为 nil，使用默认处理器
	if handler == nil {
		log.Printf("WARN 接口 [%s] %s 的handler为nil，使用默认handler", method, url)
		handler = defaultHandler
	}

	// 创建 API 注册信息并保存到注册表
	// 使用指针 &APIRegister{} 避免后续修改时的结构体拷贝问题
	apiRegistries[url] = &APIRegister{
		Method:   method,
		Handler:  handler,
		IsActive: true, // 新注册的接口默认为激活状态
	}

	// 记录注册日志，包含 handler 的内存地址用于调试
	log.Printf("INFO 注册接口: [%s] %s (handler地址: %p)", method, url, handler)

	// 如果全局 Gin 引擎已设置，立即注册路由到引擎（支持真正的动态注册）
	if globalRouter != nil {
		registerRouteToEngine(globalRouter, method, url, handler)
		log.Printf("INFO 动态路由已注册到 Gin 引擎: [%s] %s", method, url)
	}
}

// registerRouteToEngine 将路由注册到 Gin 引擎
// 
// 参数:
//   - router: Gin 引擎实例
//   - method: HTTP 方法
//   - url: API 路径
//   - handler: 处理函数
//
// 注意:
//   - 此函数必须在持有 registryMutex 写锁的情况下调用
//   - Gin 不支持重复注册同一路由，如果路由已存在会被覆盖
func registerRouteToEngine(router *gin.Engine, method, url string, handler gin.HandlerFunc) {
	switch method {
	case Method_GET:
		router.GET(url, handler)
	case Method_POST:
		router.POST(url, handler)
	case Method_PUT:
		router.PUT(url, handler)
	case Method_DELETE:
		router.DELETE(url, handler)
	default:
		router.POST(url, handler)
	}
}

// UnregisterAPI 取消注册 API 接口（动态注销，返回 404）
//
// 参数:
//   - url: 要注销的 API 路径
//
// 返回值:
//   - bool: true=注销成功，false=接口不存在
//
// 功能:
//   - 将接口标记为非激活状态（IsActive=false）
//   - 将 handler 替换为 notFoundHandler，确保返回 404
//   - 保留注册信息，便于后续重新启用或查询
//
// 注意:
//   - 注销后该接口仍然存在于注册表中，只是返回 404
//   - 如需完全删除，需手动从 map 中移除（当前未提供此功能）
//
// 示例:
//   success := UnregisterAPI("/api/old-endpoint")
func UnregisterAPI(url string) bool {
	registryMutex.Lock()       // 加写锁
	defer registryMutex.Unlock()

	reg, exists := apiRegistries[url]
	if !exists {
		log.Printf("WARN 接口不存在，无法注销: %s", url)
		return false
	}

	// 标记为非激活状态
	reg.IsActive = false
	// 将 handler 替换为 404 处理器，确保所有请求都返回 404
	reg.Handler = notFoundHandler
	apiRegistries[url] = reg

	log.Printf("INFO 注销接口: %s (返回404)", url)
	return true
}

// DisableAPI 临时禁用 API 接口（返回 404，但保留原始 handler）
//
// 参数:
//   - url: 要禁用的 API 路径
//
// 返回值:
//   - bool: true=禁用成功，false=接口不存在
//
// 功能:
//   - 仅将 IsActive 标记为 false
//   - 保留原始 handler，便于后续通过 EnableAPI 快速恢复
//   - 与 UnregisterAPI 的区别：DisableAPI 不替换 handler
//
// 使用场景:
//   - 临时维护期间禁用某个接口
//   - 灰度发布时暂时关闭旧接口
//   - 需要频繁切换激活/禁用状态的接口
//
// 示例:
//   DisableAPI("/api/maintenance-endpoint")
func DisableAPI(url string) bool {
	registryMutex.Lock()       // 加写锁
	defer registryMutex.Unlock()

	reg, exists := apiRegistries[url]
	if !exists {
		log.Printf("WARN 接口不存在，无法禁用: %s", url)
		return false
	}

	// 仅标记为非激活，不修改 handler
	reg.IsActive = false
	apiRegistries[url] = reg

	log.Printf("INFO 禁用接口: %s", url)
	return true
}

// EnableAPI 重新启用已禁用的 API 接口
//
// 参数:
//   - url: 要启用的 API 路径
//
// 返回值:
//   - bool: true=启用成功，false=接口不存在
//
// 功能:
//   - 将 IsActive 标记为 true
//   - 恢复使用原始 handler（如果是通过 DisableAPI 禁用的）
//   - 如果是通过 UnregisterAPI 注销的，需要配合 UpdateHandler 重新设置 handler
//
// 使用场景:
//   - 维护结束后恢复接口
//   - 灰度发布完成后启用新接口
//
// 示例:
//   EnableAPI("/api/maintenance-endpoint")
func EnableAPI(url string) bool {
	registryMutex.Lock()       // 加写锁
	defer registryMutex.Unlock()

	reg, exists := apiRegistries[url]
	if !exists {
		log.Printf("WARN 接口不存在，无法启用: %s", url)
		return false
	}

	// 重新激活接口
	reg.IsActive = true
	apiRegistries[url] = reg

	log.Printf("INFO 启用接口: %s", url)
	return true
}

// GetHandler 安全获取 API 的处理函数（每次调用时实时检查状态）
//
// 参数:
//   - method: HTTP 请求方法
//   - url: API 路径
//
// 返回值:
//   - gin.HandlerFunc: 对应的处理函数，如果不存在或已禁用则返回 notFoundHandler
//
// 功能:
//   - 从注册表中查找 API
//   - 检查接口是否存在
//   - 检查接口是否激活
//   - 检查 handler 是否为 nil
//   - 任何一项检查失败都返回 notFoundHandler
//
// 注意:
//   - 此函数在每次 HTTP 请求时都会被调用，确保获取最新的 handler
//   - 使用读锁，允许多个请求并发执行
//
// 示例:
//   handler := GetHandler(Method_GET, "/api/user/list")
//   handler(c) // 执行处理函数
func GetHandler(method, url string) gin.HandlerFunc {
	registryMutex.RLock()       // 加读锁，允许多个读者并发
	defer registryMutex.RUnlock()

	reg, exists := apiRegistries[url]

	// 检查 1: 接口是否存在
	if !exists {
		log.Printf("WARN 接口不存在: %s %s", method, url)
		return notFoundHandler
	}

	// 检查 2: 接口是否激活
	if !reg.IsActive {
		return notFoundHandler
	}

	// 检查 3: handler 是否为 nil（防御性编程）
	if reg.Handler == nil {
		log.Printf("WARN 接口 %s %s 的handler为nil，使用404 handler", method, url)
		return notFoundHandler
	}

	return reg.Handler
}

// ExecuteRegistrations 执行路由注册（将注册表中的 API 注册到 Gin 路由器）
//
// 参数:
//   - router: Gin 引擎实例
//
// 功能:
//   - 遍历 apiRegistries 中的所有 API
//   - 根据 HTTP 方法注册到 Gin 路由器
//   - 使用闭包捕获 URL，每次请求时动态获取最新的 handler
//   - 支持运行时动态更新 handler（无需重启服务）
//
// 重要特性:
//   - 闭包内调用 GetHandler() 确保每次请求都获取最新配置
//   - 即使注册后修改了 handler，也会立即生效
//   - 已禁用的接口会自动返回 404
//
// 注意:
//   - 此函数通常在服务启动时调用一次
//   - 如果需要热更新路由，可再次调用此函数（Gin 会覆盖已有路由）
//
// 示例:
//   router := gin.Default()
//   ExecuteRegistrations(router)
//   router.Run(":8080")
func ExecuteRegistrations(router *gin.Engine) {
	registryMutex.RLock()       // 加读锁
	defer registryMutex.RUnlock()

	for url, reg := range apiRegistries {
		// 获取当前 handler（可能是正常 handler 或 404 handler）
		handler := reg.Handler
		if handler == nil || !reg.IsActive {
			handler = notFoundHandler
		}

		// 根据 HTTP 方法注册路由
		// 使用闭包捕获 url，确保每次请求都能正确匹配
		switch reg.Method {
		case Method_GET:
			router.GET(url, func(c *gin.Context) {
				// 每次请求都重新获取 handler，支持动态更新
				h := GetHandler(Method_GET, c.FullPath())
				h(c)
			})
		case Method_POST:
			router.POST(url, func(c *gin.Context) {
				h := GetHandler(Method_POST, c.FullPath())
				h(c)
			})
		case Method_PUT:
			router.PUT(url, func(c *gin.Context) {
				h := GetHandler(Method_PUT, c.FullPath())
				h(c)
			})
		case Method_DELETE:
			router.DELETE(url, func(c *gin.Context) {
				h := GetHandler(Method_DELETE, c.FullPath())
				h(c)
			})
		default:
			// 未知方法默认使用 POST
			router.POST(url, func(c *gin.Context) {
				h := GetHandler(Method_POST, c.FullPath())
				h(c)
			})
		}

		// 记录路由注册状态
		status := "激活"
		if !reg.IsActive {
			status = "禁用"
		}
		log.Printf("INFO 注册路由: [%s] %s (状态: %s)", reg.Method, url, status)
	}
}

// ==================== 快捷注册函数 ====================

// RegisterGET 快捷注册 GET 接口
// 等价于: RegisterAPI(Method_GET, url, handler)
func RegisterGET(url string, handler gin.HandlerFunc) {
	RegisterAPI(Method_GET, url, handler)
}

// RegisterPOST 快捷注册 POST 接口
// 等价于: RegisterAPI(Method_POST, url, handler)
func RegisterPOST(url string, handler gin.HandlerFunc) {
	RegisterAPI(Method_POST, url, handler)
}

// RegisterPUT 快捷注册 PUT 接口
// 等价于: RegisterAPI(Method_PUT, url, handler)
func RegisterPUT(url string, handler gin.HandlerFunc) {
	RegisterAPI(Method_PUT, url, handler)
}

// RegisterDELETE 快捷注册 DELETE 接口
// 等价于: RegisterAPI(Method_DELETE, url, handler)
func RegisterDELETE(url string, handler gin.HandlerFunc) {
	RegisterAPI(Method_DELETE, url, handler)
}

// ==================== 快捷注销函数 ====================

// UnregisterGET 快捷注销 GET 接口
// 等价于: UnregisterAPI(url)
func UnregisterGET(url string) bool {
	return UnregisterAPI(url)
}

// UnregisterPOST 快捷注销 POST 接口
// 等价于: UnregisterAPI(url)
func UnregisterPOST(url string) bool {
	return UnregisterAPI(url)
}

// UnregisterPUT 快捷注销 PUT 接口
// 等价于: UnregisterAPI(url)
func UnregisterPUT(url string) bool {
	return UnregisterAPI(url)
}

// UnregisterDELETE 快捷注销 DELETE 接口
// 等价于: UnregisterAPI(url)
func UnregisterDELETE(url string) bool {
	return UnregisterAPI(url)
}

// ==================== 更新 handler ====================

// UpdateHandler 更新 API 的处理函数
//
// 参数:
//   - method: HTTP 请求方法
//   - url: API 路径
//   - newHandler: 新的处理函数，如果为 nil 则使用 defaultHandler
//
// 返回值:
//   - bool: true=更新成功，false=接口不存在
//
// 功能:
//   - 替换指定 API 的 handler
//   - 自动将接口设置为激活状态（IsActive=true）
//   - 支持运行时热更新接口逻辑，无需重启服务
//
// 使用场景:
//   - 修复线上 bug 时替换 handler
//   - A/B 测试时切换不同版本的 handler
//   - 灰度发布时逐步替换接口实现
//
// 注意:
//   - 更新后立即生效，正在处理的请求不受影响
//   - 新请求会使用更新后的 handler
//
// 示例:
//   UpdateHandler(Method_POST, "/api/data", newHandler)
func UpdateHandler(method, url string, newHandler gin.HandlerFunc) bool {
	registryMutex.Lock()       // 加写锁
	defer registryMutex.Unlock()

	reg, exists := apiRegistries[url]
	if !exists {
		log.Printf("WARN 接口不存在，无法更新: %s %s", method, url)
		return false
	}

	// 安全检查：如果新 handler 为 nil，使用默认处理器
	if newHandler == nil {
		log.Printf("WARN 更新handler为nil，使用默认handler")
		newHandler = defaultHandler
	}

	// 更新 handler 并自动激活
	reg.Handler = newHandler
	reg.IsActive = true // 更新时自动激活，确保接口可用
	apiRegistries[url] = reg

	log.Printf("INFO 更新接口handler: [%s] %s (新handler地址: %p)", method, url, newHandler)
	return true
}

// ==================== 查询接口状态 ====================

// IsAPIActive 检查 API 是否处于激活状态
//
// 参数:
//   - url: API 路径
//
// 返回值:
//   - bool: true=激活，false=禁用或不存在
//
// 功能:
//   - 快速检查接口的激活状态
//   - 用于监控、管理界面展示
//
// 示例:
//   if IsAPIActive("/api/user") {
//       fmt.Println("用户接口正常运行")
//   }
func IsAPIActive(url string) bool {
	registryMutex.RLock()       // 加读锁
	defer registryMutex.RUnlock()

	reg, exists := apiRegistries[url]
	if !exists {
		return false
	}
	return reg.IsActive
}

// GetAPIStatus 获取 API 的详细状态信息
//
// 参数:
//   - url: API 路径
//
// 返回值:
//   - method: HTTP 请求方法（GET/POST/PUT/DELETE）
//   - isActive: 是否激活
//   - exists: 是否存在于注册表中
//
// 功能:
//   - 一次性获取接口的完整状态信息
//   - 用于调试、监控、管理界面
//
// 示例:
//   method, active, exists := GetAPIStatus("/api/user")
//   if exists {
//       fmt.Printf("方法: %s, 激活: %v\n", method, active)
//   }
func GetAPIStatus(url string) (method string, isActive bool, exists bool) {
	registryMutex.RLock()       // 加读锁
	defer registryMutex.RUnlock()

	reg, exists := apiRegistries[url]
	if !exists {
		return "", false, false
	}
	return reg.Method, reg.IsActive, true
}

// ListAllAPIs 列出所有已注册的 API 接口
//
// 返回值:
//   - map[string]*APIRegister: key=API路径, value=API注册信息副本
//
// 功能:
//   - 返回注册表中所有 API 的快照
//   - 返回的是副本，不会暴露内部数据结构
//   - 用于管理界面、监控系统、调试工具
//
// 注意:
//   - 返回的数据是某一时刻的快照，可能与实际状态有延迟
//   - 修改返回的副本不会影响实际注册表
//
// 示例:
//   apis := ListAllAPIs()
//   for url, reg := range apis {
//       fmt.Printf("%s [%s] 激活:%v\n", url, reg.Method, reg.IsActive)
//   }
func ListAllAPIs() map[string]*APIRegister {
	registryMutex.RLock()       // 加读锁
	defer registryMutex.RUnlock()

	// 创建副本，避免并发问题和数据泄露
	result := make(map[string]*APIRegister)
	for url, reg := range apiRegistries {
		result[url] = &APIRegister{
			Method:   reg.Method,
			Handler:  reg.Handler,
			IsActive: reg.IsActive,
		}
	}
	return result
}

// ==================== 定时检查任务（可选功能） ====================

// StartCheckTask 启动定时健康检查任务
//
// 功能:
//   - 每 5 分钟检查一次所有注册的 API
//   - 发现 handler 为 nil 的接口自动重置为 404 handler
//   - 防止因 GC 或其他原因导致 handler 丢失
//   - 在后台 goroutine 中运行，不阻塞主线程
//
// 注意:
//   - 此函数是可选的，根据需要决定是否启动
//   - 建议在服务启动时调用一次
//   - 使用 defer ticker.Stop() 确保资源正确释放
//
// 示例:
//   func main() {
//       StartCheckTask() // 启动健康检查
//       // ... 其他初始化代码
//       router.Run(":8080")
//   }
func StartCheckTask() {
	go func() {
		// 创建定时器，每 5 分钟触发一次
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop() // 确保 goroutine 退出时停止定时器

		// 循环等待定时器触发
		for range ticker.C {
			registryMutex.RLock()
			for url, reg := range apiRegistries {
				// 检查 handler 是否意外变为 nil
				if reg.Handler == nil {
					log.Printf("WARN 发现handler为nil的接口: %s，重置为404 handler", url)
					// 重置为 404 handler 并禁用接口
					reg.Handler = notFoundHandler
					reg.IsActive = false
					apiRegistries[url] = reg
				}
			}
			registryMutex.RUnlock()
		}
	}()
}
