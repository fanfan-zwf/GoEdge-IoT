package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-yaml/yaml"
)

const (
	Version = "v1.0.0"

	RFC_FAN = "2006-01-02 15:04:05.000000"

	User_Permissions = 100

	// 正则表达式值
	Regex_Name            = `^.{0,6}$`
	Regex_Passwd          = `^(?=.*\d)(?=.*[!@#$%&])[A-Za-z\d!@#$%&]{8}$`
	Regex_Passwd_sha3_256 = `^[a-f0-9]{64}$`
	Regex_Phone           = `^1[3-9]\d{9}$`
	Regex_Email           = `^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`
	Regex_URL             = `https?:\/\/[^\s]+`
)

type Config_type struct {
	APP struct {
		Version string `yaml:"version"` // 版本号
		SN      string `yaml:"sn"`      // 设备id
		Passwd  string `yaml:"passwd"`
		Label   string `yaml:"label"` // 标识符
	} `yaml:"APP"` // 程序主要参数

	Config_Service struct {
		Enable  bool   `yaml:"enable"`  // 配置服务是否启用
		Address string `yaml:"address"` // 配置服务地址

		SetTimeout       time.Duration `yaml:"setTimeout"`       // 设置超时时间
		SetRetryCount    int           `yaml:"setRetryCount"`    // 401 最多重试 1 次，避免死循环
		SetRetryWaitTime time.Duration `yaml:"setRetryWaitTime"` // 设置重试等待时间
	} `yaml:"Config_Service"`

	API struct {
		Enable bool   `yaml:"enable"`
		Ip     string `yaml:"ip"`
		Post   uint16 `yaml:"post"`
		Header []struct {
			Key   string `yaml:"key"`
			Value string `yaml:"value"`
		} `yaml:"header"` // 请求头
	} `yaml:"API"` // 接口服务

	MYSQL struct {
		Dsn string `yaml:"dsn"`
	} `yaml:"MYSQL"` // 数据库

	REDIS struct {
		Ip       string `yaml:"ip"`
		Post     uint16 `yaml:"post"`
		Passwd   string `yaml:"passwd"`
		Database int    `yaml:"database"`
	} `yaml:"REDIS"` // 数据库

	Influxdb struct {
		Enable        bool   `yaml:"enable"`
		Url           string `yaml:"url"`
		Token         string `yaml:"token"`
		Org           string `yaml:"org"`
		Bucket        string `yaml:"bucket"`
		Write_Timeout uint   `yaml:"write_timeout"` // 单次 HTTP 请求超时 单位：毫秒

		// 以下批量/重传参数均作用于 InfluxDB 客户端自带的写入缓冲区（攒够数量或到时间就发送一次）
		BufferSize     uint          `yaml:"buffer_size"`    // 单批点数阈值
		FlushInterval  time.Duration `yaml:"flush_interval"` // 缓冲区刷新间隔
		Max_Retries    uint          `yaml:"max_retries"`    // 写入失败最大重试次数，0=使用默认值
		Retry_Interval time.Duration `yaml:"retry_interval"` // 重试等待间隔
		Retry_Buffer   uint          `yaml:"retry_buffer"`   // 重试缓冲区最大点数，0=按单批点数自动计算
		Use_GZip       bool          `yaml:"use_gzip"`       // 写入请求 GZip 压缩，降低带宽占用

		Write_Quantity_Tag string `yaml:"write_quantity_tag"` // 「本批写入点数」统计点的点位标识 Tag（为空时使用默认值 write_quantity）

	} `yaml:"Influxdb"` // 时序数据库

	LOG struct {
		Enable   bool   `yaml:"enable"`
		Path     string `yaml:"path"`
		CacheTTL uint   `yaml:"cacheTTL"`
		Flags    string `yaml:"flags"`
	} `yaml:"LOG"`

	Internal_Mqtt struct {
		Enable bool `yaml:"enable"` // 是否启用内部MQTT功能

		// 独立 MQTT 连接参数（单独建立一条专用连接）
		Broker         string        `yaml:"broker"`   // broker 地址，如 tcp://127.0.0.1:1883
		Username       string        `yaml:"username"` // 用户名
		Password       string        `yaml:"password"` // 密码
		ClientID       string        `yaml:"client_id"`
		CleanSession   bool          `yaml:"clean_session"`   // 清洁会话（重启不接收离线消息）
		AutoReconnect  bool          `yaml:"auto_reconnect"`  // 自动重连
		ConnectTimeout time.Duration `yaml:"connect_timeout"` // 连接超时
		WriteTimeout   time.Duration `yaml:"write_timeout"`   // 写超时
		KeepAlive      time.Duration `yaml:"keep_alive"`      // 心跳保活

		TopicPrefix string `yaml:"topic_prefix"` // 主题前缀，实际主题 = topic_prefix + "/" + APP.label + 后缀

		// 订阅（下行）后缀，均可自定义；留空使用内置默认值
		SubWrite        string `yaml:"sub_write"`         // 写数据
		SubRestart      string `yaml:"sub_restart"`       // 重启驱动/服务
		SubToken        string `yaml:"sub_token"`         // 增加临时 handler token（长/短）
		SubConfigUpdate string `yaml:"sub_config_update"` // 更新所有配置（从http地址下载表格导入）

		// 发布（上行）后缀，均可自定义；留空使用内置默认值
		PubData         string `yaml:"pub_data"`          // 读数据（点位实时值）
		PubAlarm        string `yaml:"pub_alarm"`         // 报警事件（配置字段 + 报警值/时间）
		PubStatusPrefix string `yaml:"pub_status_prefix"` // 当前监控状态前缀（每个功能一个子主题）
		PubResult       string `yaml:"pub_result"`        // 操作结果回执（重启/令牌/配置更新）

		StatusInterval   time.Duration `yaml:"status_interval"`    // 监控状态发布周期
		DataFullInterval time.Duration `yaml:"data_full_interval"` // 点位数据全量推送周期
	} `yaml:"Internal_Mqtt"` // 内部MQTT功能（对外提供采集器读写/监控/运维主题）

	ALARM struct {
		Config_CacheTTL time.Duration `yaml:"Config_CacheTTL"` // 报警配置缓存TTL，0s=永久保存
	} `yaml:"ALARM"` // 报警配置

	// RBAC 极简权限系统（主体复用 user_account=业务用户、api_key=API 客户端）
	// 说明：无启用/拦截开关——RBAC 恒开启并强制拦截，无权限一律 403；无超管白名单，
	//       管理员靠 super_admin 角色走真实权限链路（见 rbac_admin_role_seed.sql）。
	RBAC struct {
		TTL time.Duration `yaml:"cache_ttl"` // 有效权限规则 Redis 缓存 TTL，过期后回源重建
	} `yaml:"RBAC"`
}

var Config Config_type

// 配置文件路径
const ConfigFilePath = "configs/config.yaml"

func init() {
	loadConfig()
	init_log()
}

// loadConfig 加载配置文件
func loadConfig() {
	data, err := os.ReadFile(ConfigFilePath)
	if err != nil {
		log.Panic("ERR", "读取配置文件错误", err)
		return
	}

	// Config_Service 默认值
	Config.Config_Service.SetTimeout = 10 * time.Second
	Config.Config_Service.SetRetryCount = 1
	Config.Config_Service.SetRetryWaitTime = 3 * time.Second

	// API 默认值
	Config.API.Ip = "0.0.0.0"
	Config.API.Post = 8102

	// ALARM 默认值
	Config.ALARM.Config_CacheTTL = 1 * time.Hour

	// RBAC 默认值：有效权限规则 Redis 缓存 30 分钟
	Config.RBAC.TTL = 30 * time.Minute

	// log
	Config.LOG.Path = "./log"
	Config.LOG.CacheTTL = 30
	Config.LOG.Flags = "Ldate Lmicroseconds Llongfile"

	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		log.Panic("ERR", "写入结构体错误", err)
		return
	}

	if err := checkAppVersion(Config.APP.Version); err != nil {
		log.Panic("ERR", err)
		return
	}

	applyMqttDefaults()
}

// leadingInt 取出字符串开头的十进制数字，遇到非数字即结束，
// 因此允许修订号/预发布后缀（如 "0"、"12-rc1"、"3.1 "）；无开头数字或溢出返回 ok=false。
func leadingInt(s string) (int, bool) {
	s = strings.TrimSpace(s)

	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}

	n, err := strconv.Atoi(s[:end])
	if err != nil { // 位数过多导致溢出
		return 0, false
	}
	return n, true
}

// parseMinor 提取版本号的前两段（主版本.次版本）。
//   - 段数不设上限：两段 "v1.0"、三段 "v1.0.0"、四段及以上 "v1.0.0.1.2.3" 均可，
//     第三段往后的修订号/构建号/预发布标记一律忽略；
//   - 但至少要有两段，且每段非空（连续点、首尾点如 "v1.0." 均判非法）；
//   - 前两段还须以数字开头，否则同样判非法。
func parseMinor(v string) (major, minor int, ok bool) {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")

	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return 0, 0, false
	}
	for _, p := range parts {
		if strings.TrimSpace(p) == "" { // 空段：如 "v1..0"、"v1.0."、"v.1.0"
			return 0, 0, false
		}
	}

	major, ok = leadingInt(parts[0])
	if !ok {
		return 0, 0, false
	}
	minor, ok = leadingInt(parts[1])
	if !ok {
		return 0, 0, false
	}
	return major, minor, true
}

// checkAppVersion 校验配置文件 APP.version 与软件内置 Version 是否兼容：
//   - 主版本号、次版本号必须完全一致，修订号及其后的所有段不作要求
//     （软件 v1.0.0 可接受配置 v1.0 / v1.0.2 / v1.0.3 / v1.0.0.1，拒绝 v1.1、v1.3、v2.1）；
//   - 两侧版本段数均不限（最少两段），取前两段做数值比较而非字符串比较；
//   - 配置版本为空或格式非法同样拒绝，避免拿错版本的配置文件继续启动。
func checkAppVersion(cfgVersion string) error {
	if strings.TrimSpace(cfgVersion) == "" {
		return fmt.Errorf("配置文件缺少 APP.version（软件版本 %s，请填写与之匹配的配置版本）", Version)
	}

	sMajor, sMinor, ok := parseMinor(Version)
	if !ok {
		return fmt.Errorf("软件版本号格式非法: %s", Version)
	}
	cMajor, cMinor, ok := parseMinor(cfgVersion)
	if !ok {
		return fmt.Errorf("配置文件 APP.version 格式非法: %s（至少两段且以数字开头，如 vX.Y / vX.Y.Z / vX.Y.Z.N）", cfgVersion)
	}

	if sMajor != cMajor || sMinor != cMinor {
		return fmt.Errorf("配置文件版本 %s 与软件版本 %s 不兼容，主/次版本号必须一致（本软件接受 %d.%d.x 的配置）",
			cfgVersion, Version, sMajor, sMinor)
	}

	return nil
}

// applyMqttDefaults 补全内部 MQTT 功能独立连接的程序内默认值：
//   - client_id 缺省取 APP.label，超时/保活缺省为内置值；
//   - 主题前缀缺省为 /IOT/Collector，各后缀缺省为内置默认值，周期缺省为内置值。
//
// 在 loadConfig / ReloadConfig 解析完成后调用，确保 yaml 未显式配置时也能正常工作。
func applyMqttDefaults() {
	// Internal_Mqtt 独立连接的默认值
	if Config.APP.Label != "" && Config.Internal_Mqtt.ClientID == "" {
		Config.Internal_Mqtt.ClientID = Config.APP.Label
	}
	if Config.Internal_Mqtt.ConnectTimeout <= 0 {
		Config.Internal_Mqtt.ConnectTimeout = 10 * time.Second
	}
	if Config.Internal_Mqtt.WriteTimeout <= 0 {
		Config.Internal_Mqtt.WriteTimeout = 5 * time.Second
	}
	if Config.Internal_Mqtt.KeepAlive <= 0 {
		Config.Internal_Mqtt.KeepAlive = 30 * time.Second
	}
	if Config.Internal_Mqtt.TopicPrefix == "" {
		Config.Internal_Mqtt.TopicPrefix = "/IOT/Collector"
	}
	if Config.Internal_Mqtt.SubWrite == "" {
		Config.Internal_Mqtt.SubWrite = "/write"
	}
	if Config.Internal_Mqtt.SubRestart == "" {
		Config.Internal_Mqtt.SubRestart = "/restart"
	}
	if Config.Internal_Mqtt.SubToken == "" {
		Config.Internal_Mqtt.SubToken = "/token"
	}
	if Config.Internal_Mqtt.SubConfigUpdate == "" {
		Config.Internal_Mqtt.SubConfigUpdate = "/config/update"
	}
	if Config.Internal_Mqtt.PubData == "" {
		Config.Internal_Mqtt.PubData = "/data"
	}
	if Config.Internal_Mqtt.PubAlarm == "" {
		Config.Internal_Mqtt.PubAlarm = "/alarm"
	}
	if Config.Internal_Mqtt.PubStatusPrefix == "" {
		Config.Internal_Mqtt.PubStatusPrefix = "/status"
	}
	if Config.Internal_Mqtt.PubResult == "" {
		Config.Internal_Mqtt.PubResult = "/result"
	}
	if Config.Internal_Mqtt.StatusInterval <= 0 {
		Config.Internal_Mqtt.StatusInterval = 30 * time.Second
	}
	if Config.Internal_Mqtt.DataFullInterval <= 0 {
		Config.Internal_Mqtt.DataFullInterval = 10 * time.Minute
	}
}

// ReloadConfig 重新加载配置文件（用于外部修改后刷新）
func ReloadConfig() error {
	data, err := os.ReadFile(ConfigFilePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 保留需要维持的默认值
	defaultConfig := Config

	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		Config = defaultConfig // 解析失败时恢复原配置
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 版本不兼容时整体回滚，避免半套新版本配置生效
	if err := checkAppVersion(Config.APP.Version); err != nil {
		Config = defaultConfig
		return err
	}

	// 确保默认值不被覆盖
	if Config.Config_Service.SetTimeout == 0 {
		Config.Config_Service.SetTimeout = defaultConfig.Config_Service.SetTimeout
	}
	if Config.Config_Service.SetRetryCount == 0 {
		Config.Config_Service.SetRetryCount = defaultConfig.Config_Service.SetRetryCount
	}
	if Config.Config_Service.SetRetryWaitTime == 0 {
		Config.Config_Service.SetRetryWaitTime = defaultConfig.Config_Service.SetRetryWaitTime
	}
	if Config.API.Ip == "" {
		Config.API.Ip = "0.0.0.0"
	}
	if Config.API.Post == 0 {
		Config.API.Post = 8102
	}
	if Config.ALARM.Config_CacheTTL == 0 {
		Config.ALARM.Config_CacheTTL = 1 * time.Hour
	}
	if Config.RBAC.TTL == 0 {
		Config.RBAC.TTL = defaultConfig.RBAC.TTL
	}
	if Config.LOG.Path == "" {
		Config.LOG.Path = "./log"
	}
	if Config.LOG.CacheTTL == 0 {
		Config.LOG.CacheTTL = 30
	}
	if Config.LOG.Flags == "" {
		Config.LOG.Flags = "Ldate Lmicroseconds Llongfile"
	}

	applyMqttDefaults()

	return nil
}

// SaveConfig 将当前配置保存到配置文件
func SaveConfig() error {
	data, err := yaml.Marshal(&Config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	err = os.WriteFile(ConfigFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}
