package Init

import (
	"io/ioutil"
	"log"
	"os"

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

type Config_type struct {
	APP struct {
		Version                 string    `yaml:"version"`                 // 版本号
		SN                      string    `yaml:"sn"`                      // 设备id
		Historical_Storage_Time uint      `yaml:"historical_storage_time"` // 历史缓存时间, 先存入redis再批量写入mysql
		Uuid                    uuid.UUID `yaml:"uuid"`
	} `yaml:"APP"` // 程序主要参数

	API struct {
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

	LOG struct {
		Enable   bool   `yaml:"enable"`
		Path     string `yaml:"path"`
		CacheTTL uint   `yaml:"cacheTTL"`
		Flags    string `yaml:"flags"`
	} `yaml:"LOG"` // GPIO

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

	if Config.APP.Uuid == uuid.Nil {
		Config.APP.Uuid = uuid.New()

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
