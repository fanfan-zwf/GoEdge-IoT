package Init

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/google/uuid"
)

const (
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

type MqttItem struct {
	Enable            bool          `yaml:"enable"`
	Broker            string        `yaml:"broker"`
	Username          string        `yaml:"username"`
	Password          string        `yaml:"password"`
	ClientID          string        `yaml:"client_id"`
	SetCleanSession   bool          `yaml:"clean_session"`   // 清洁会话（重启不接收离线消息）
	SetAutoReconnect  bool          `yaml:"auto_reconnect"`  // 自动重连（必须开）
	SetConnectTimeout time.Duration `yaml:"connect_timeout"` // 连接超时
	SetWriteTimeout   time.Duration `yaml:"write_timeout"`   // 写超时
	SetKeepAlive      time.Duration `yaml:"keep_alive"`      // 心跳保活
}

type Config_type struct {
	APP struct {
		Version   string `yaml:"version"` // 版本号
		SN        string `yaml:"sn"`      // 设备id
		Uuid      string `yaml:"uuid"`
		AesPasswd string `yaml:"aes_passwd"`
	} `yaml:"APP"` // 程序主要参数

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
		Enable        bool          `yaml:"enable"`
		Url           string        `yaml:"url"`
		Token         string        `yaml:"token"`
		Org           string        `yaml:"org"`
		Bucket        string        `yaml:"bucket"`
		Write_Timeout uint          `yaml:"write_timeout"`  // 写入超时 单位：毫秒
		BufferSize    int           `yaml:"buffer_size"`    // 缓冲区大小
		FlushInterval time.Duration `yaml:"flush_interval"` // 刷新间隔

		Write_Quantity_Tag string `yaml:"write_quantity_tag"` // 写入时序数据库的点数量

		Redis_Cache_Enable        bool          `yaml:"redis_cache_enable"`         // 是否启用redis缓存
		Redis_Cache_ufferSize     int           `yaml:"redis_cache_buffer_size"`    // 缓冲区大小
		Redis_Cache_FlushInterval time.Duration `yaml:"redis_cache_flush_interval"` // 刷新间隔
	} `yaml:"Influxdb"` // 时序数据库

	LOG struct {
		Enable   bool   `yaml:"enable"`
		Path     string `yaml:"path"`
		CacheTTL uint   `yaml:"cacheTTL"`
		Flags    string `yaml:"flags"`
	} `yaml:"LOG"`

	User_Service struct {
		Url    string `yaml:"url"`
		ApiKey string `yaml:"apikey"`
		Secret string `yaml:"secret"`
	} `yaml:"User_Service"` // 用户服务

	Mqtt_Rpc struct {
		Example            string        `yaml:"example"`
		Enable             bool          `yaml:"enable"`
		BusinessTimeout    time.Duration `yaml:"business_timeout"`
		ListenTopic        string        `yaml:"listen_topic"`
		ConfigServiceTopic string        `yaml:"config_service_topic"`

		Point_Push_Value  string `yaml:"point_push_value"`  // 点更新值
		Point_Down_value  string `yaml:"point_down_value"`  // 点下发值
		Point_Alarm_Value string `yaml:"point_alarm_value"` // 点更新值
	} `yaml:"Mqtt_Rpc"` // mqtt版的rpc通信

	Mqtt map[string]MqttItem `yaml:"Mqtt"` // mqtt版的rpc通信
}

var Config Config_type

func init() {
	// // 获取可执行文件的路径
	// exePath, err := os.Executable()
	// if err != nil {
	// 	fmt.Printf("获取执行文件路径失败: %v\n", err)
	// 	return
	// }

	// // 得到可执行文件所在目录
	// execDir := filepath.Dir(exePath)

	// // 构建配置文件的绝对路径
	// configPath := filepath.Join(execDir, "config.yaml")
	// fmt.Printf("尝试从执行目录加载配置: %s\n", configPath)

	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Panic("ERR", "读取配置文件错误", err)
		return
	}

	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		log.Panic("ERR", "写入结构体错误", err)
		return
	}

	if Config.APP.Uuid == "" {
		Config.APP.Uuid = uuid.New().String()

		update_data, err := yaml.Marshal(&Config)
		if err != nil {
			panic("结构体转化字符串错误 " + err.Error())
		}

		err = os.WriteFile("config.yaml", update_data, 0755)
		if err != nil {
			panic("uuid写入配置文件错误 " + err.Error())
		}
	}
	init_log()
}
