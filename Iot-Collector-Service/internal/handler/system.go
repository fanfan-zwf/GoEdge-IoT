/*
* 日期: 2026.11.20
* 作者: 范范zwf
* 作用: 系统管理 API（合并自原 system_config / system_info）：
*       一、config.yaml 的读取/写入/重载（写读只记指纹不记正文，含口令）；
*       二、系统性能信息，按功能逐一路由 /api/v1.0/system/info/<功能>，直接返回 pkg/sysinfo 的 gopsutil 原始结果。
 */
package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"

	"collector/internal/config"
	"collector/internal/repo/mysql"
	"collector/pkg/sysinfo"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// ============================================================
// 系统配置管理（config.yaml 读写/重载）
// ============================================================

// SystemConfigFileRequest 系统配置文件写入请求
type SystemConfigFileRequest struct {
	Content string `json:"content"` // YAML 文件原始内容
}

// api_system_config_read 读取系统配置文件（返回原始 YAML 内容）
func api_system_config_read(ctx *gin.Context) {
	// 直接读取配置文件内容
	data, err := os.ReadFile(config.ConfigFilePath)
	if err != nil {
		ctx.Set("Response", []any{500, "读取配置文件失败: " + err.Error()})
		return
	}

	// 返回原始 YAML 文本（内容含账号口令，只记“读了配置文件”，不把正文写进日志）
	logOperate(ctx, mysql.Log_Type_System, mysql.Log_Operate_Query, gin.H{"action": "read", "path": config.ConfigFilePath, "bytes": len(data)})
	ctx.Set("Response", []any{200, "ok", string(data)})
}

// api_system_config_write 写入系统配置文件（接收原始 YAML 内容）
func api_system_config_write(ctx *gin.Context) {
	var req SystemConfigFileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if req.Content == "" {
		ctx.Set("Response", []any{417, "配置内容不能为空"})
		return
	}

	// 验证 YAML 格式是否正确
	var testConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(req.Content), &testConfig); err != nil {
		ctx.Set("Response", []any{417, "YAML 格式错误: " + err.Error()})
		return
	}

	// 写入配置文件
	if err := os.WriteFile(config.ConfigFilePath, []byte(req.Content), 0644); err != nil {
		ctx.Set("Response", []any{500, "写入配置文件失败: " + err.Error()})
		return
	}

	// config.yaml 里有 MQTT 账号口令，日志只存指纹与长度，可比对变更不存正文
	sum := sha256.Sum256([]byte(req.Content))
	logOperate(ctx, mysql.Log_Type_System, mysql.Log_Operate_Update, gin.H{
		"action": "write",
		"path":   config.ConfigFilePath,
		"bytes":  len(req.Content),
		"sha256": hex.EncodeToString(sum[:8]),
	})
	ctx.Set("Response", []any{200, "配置保存成功"})
}

// api_system_config_reload 重新加载配置文件（用于外部修改后刷新）
func api_system_config_reload(ctx *gin.Context) {
	if err := config.ReloadConfig(); err != nil {
		ctx.Set("Response", []any{500, err.Error()})
		return
	}

	logOperate(ctx, mysql.Log_Type_System, mysql.Log_Operate_Update, gin.H{"action": "reload", "path": config.ConfigFilePath})
	ctx.Set("Response", []any{200, "配置重新加载成功"})
}

// system_config_api_register 注册系统配置管理路由
func system_config_api_register(r *gin.Engine) {
	r.POST("/api/v1.0/system/config/read", api_system_config_read)
	r.POST("/api/v1.0/system/config/write", api_system_config_write)
	r.POST("/api/v1.0/system/config/reload", api_system_config_reload)
}

// ============================================================
// 系统性能信息（通配 <功能> 子路径，直返 gopsutil 原始结果）
// ============================================================

// queryBool 解析查询参数为 bool，缺省或无法识别时返回 def。
func queryBool(ctx interface{ Query(string) string }, key string, def bool) bool {
	switch strings.ToLower(ctx.Query(key)) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return def
	}
}

// api_system_info 按子路径 <功能> 分发到对应的采集函数，直接返回原始结果。
// 形如 /api/v1.0/system/info/cpu/percent、/api/v1.0/system/info/mem/swap 等。
func api_system_info(ctx *gin.Context) {
	c := ctx.Request.Context()
	// 去掉通配参数的前导斜杠，得到功能路径，如 "cpu/percent"
	fn := strings.TrimPrefix(ctx.Param("path"), "/")

	var (
		data any
		err  error
	)

	switch fn {
	// ---- Host / OS ----
	case "host":
		data, err = sysinfo.CollectHostInfo(c)
	case "host/users":
		data, err = sysinfo.CollectHostUsers(c)
	case "host/virtualization":
		var system, role string
		system, role, err = sysinfo.CollectVirtualization(c)
		data = map[string]string{"system": system, "role": role}
	case "sensors":
		data, err = sysinfo.CollectSensors(c)

	// ---- CPU ----
	case "cpu":
		data, err = sysinfo.CollectCPUInfo(c)
	case "cpu/cores":
		data, err = sysinfo.CollectCPUCounts(c, queryBool(ctx, "logical", true))
	case "cpu/percent":
		data, err = sysinfo.CollectCPUPercent(c, sysinfo.DefaultCPUInterval, queryBool(ctx, "percpu", false))
	case "cpu/times":
		data, err = sysinfo.CollectCPUTimes(c, queryBool(ctx, "percpu", false))

	// ---- Mem ----
	case "mem":
		data, err = sysinfo.CollectVirtualMemory(c)
	case "mem/swap":
		data, err = sysinfo.CollectSwapMemory(c)
	case "mem/swapdevices":
		data, err = sysinfo.CollectSwapDevices(c)

	// ---- Disk ----
	case "disk":
		data, err = sysinfo.CollectDiskPartitions(c, queryBool(ctx, "all", false))
	case "disk/usage":
		path := ctx.Query("path")
		if path == "" {
			path = "/"
		}
		data, err = sysinfo.CollectDiskUsage(c, path)
	case "disk/io":
		data, err = sysinfo.CollectDiskIOCounters(c)

	// ---- Load ----
	case "load":
		data, err = sysinfo.CollectLoadAvg(c)
	case "load/misc":
		data, err = sysinfo.CollectLoadMisc(c)

	// ---- Net ----
	case "net":
		data, err = sysinfo.CollectNetIOCounters(c, queryBool(ctx, "pernic", true))
	case "net/interfaces":
		data, err = sysinfo.CollectNetInterfaces(c)
	case "net/protocols":
		var protocols []string
		if q := ctx.Query("protocols"); q != "" {
			protocols = strings.Split(q, ",")
		}
		data, err = sysinfo.CollectNetProtoCounters(c, protocols)
	case "net/connections":
		kind := ctx.Query("kind")
		if kind == "" {
			kind = "tcp"
		}
		data, err = sysinfo.CollectNetConnections(c, kind)

	// ---- Process ----
	case "process":
		data, err = sysinfo.CollectProcesses(c)
	case "process/pids":
		data, err = sysinfo.CollectPids(c)

	default:
		ctx.Set("Response", []any{417, "未知的系统信息功能: " + fn})
		return
	}

	if err != nil {
		ctx.Set("Response", []any{500, "采集失败: " + err.Error()})
		return
	}

	ctx.Set("Response", []any{200, "ok", data})
}

// system_info_api_register 注册系统信息相关路由（通配 <功能> 子路径）。
func system_info_api_register(r *gin.Engine) {
	r.GET("/api/v1.0/system/info/*path", api_system_info)
	r.POST("/api/v1.0/system/info/*path", api_system_info)
}
