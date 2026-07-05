package flexem_flexem

import (
	"main/IO/manager/fullConfig"
	"main/cloud"
	"main/db/mysql"
	"main/web"

	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var ValueType_map = map[int]string{
	1:  "bool",
	2:  "int",
	3:  "int",
	4:  "int",
	5:  "int",
	6:  "int",
	7:  "int",
	8:  "int",
	9:  "int",
	10: "float",
	11: "float",
	12: "string",
}

var status_map = map[int]string{
	0:  "ok",
	1:  "无数据",
	2:  "超时",
	3:  "错误",
	4:  "Socket异常",
	5:  "FDS错误",
	16: "未完成",
}

// 设备状态类型的参数
// type API_equipmentValue_struct struct {
// 	Rssi        int       // 信号值
// 	BoxId       int       // 盒子Id
// 	Timestamp   time.Time // 报警推送的时间戳
// 	BoxSubType  int       // 盒子类型
// 	NetworkType int       // 盒子网络类型
// 	NewState    int       // 盒子状态
// }

type API_indexkey_struct struct {
	DeviceSn      string // 盒子序列号
	DmonGroupName string // 监控点分组名称
	DmonName      string // 监控点名称
}

// 监控点数据结构
type API_dataValue_struct struct {
	DeviceSn      string    // 盒子序列号
	DeviceId      int       // 盒子Id
	Timestamp     time.Time // 数据推送的时间戳
	DeviceType    int       // 盒子类型
	DmonGroupId   int       // 监控点分组Id
	DmonGroupName string    // 监控点分组名称
	DmonName      string    // 监控点名称
	DmonId        int       // 监控点Id
	Error         int       // 监控点状态
	ValueType     int       // 值类型
	BoolValue     bool      // 值类型为1时，此字段接收
	IntValue      int       // 值类型为2,3,4,5时，此字段接收
	UIntValue     uint      // 值类型为6,7,8,9时，此字段接收
	FloatValue    float64   // 值类型为10,11时，此字段接收
	StringValue   string    // 值类型为12时，此字段接收
}

// 定义一个结构体
type Flexem_FlexEm struct {
	fullConfig.BaseDriver // 驱动全配置（驱动配置 + 该驱动下的所有点位配置）

	Callback_Push_External_Mappings func([]fullConfig.Value_type) error // 外部推送函数

	// 盒子序列号+监控点分组名称+监控点名称 对应配置列表下标
	api_indexkey_RWMu sync.RWMutex
	api_indexkey      map[API_indexkey_struct]int // 盒子序列号+监控点分组名称+监控点名称 对应配置列表下标

}

// 定义接口
type Connect_interface interface {
	New() error
}

func (c *Flexem_FlexEm) Start() error {

	dataUrl, ok := cloud.GetKVValue(c.Config.Drive.Config, "数据推送url")
	if !ok || dataUrl == "" {
		return fmt.Errorf("数据推送url是空的 【%s】", c.Config.Drive.Config)
	}

	// 注册api接口
	err := web.RegisterPOST(dataUrl, c.dataHandler)
	if err != nil {
		return err
	}

	api_indexkey := make(map[API_indexkey_struct]int)
	deviceSn_finally_time := make(map[string]time.Time)

	for i, point := range c.Config.Points {
		// 从驱动配置获取设备SN（一个驱动对应一个设备）
		deviceSn, ok := cloud.GetKVValue(point.Config, "盒子序列号")
		if !ok || deviceSn == "" {
			err := fmt.Errorf("ERROR 驱动名称【%s】 缺少盒子序列号【%s】", c.Config.Drive.Name, deviceSn)
			log.Print(err)
			return err
		}

		dmonGroupName, ok := cloud.GetKVValue(point.Config, "监控点分组名称")
		if !ok || dmonGroupName == "" {
			err := fmt.Errorf("ERROR 驱动名称【%s】 缺少监控点分组名称【%s】", c.Config.Drive.Name, dmonGroupName)
			log.Print(err)
			return err
		}

		dmonName, ok := cloud.GetKVValue(point.Config, "监控点名称")
		if !ok || dmonName == "" {
			err := fmt.Errorf("ERROR 驱动名称【%s】 缺少监控点名称【%s】", c.Config.Drive.Name, dmonName)
			log.Print(err)
			return err
		}

		api_indexkey[API_indexkey_struct{
			DeviceSn:      deviceSn,
			DmonGroupName: dmonGroupName,
			DmonName:      dmonName,
		}] = i

		_, ok = deviceSn_finally_time[deviceSn]
		if !ok {
			deviceSn_finally_time[deviceSn] = time.Time{}
		}
	}

	c.api_indexkey_RWMu.Lock()
	defer c.api_indexkey_RWMu.Unlock()
	c.api_indexkey = api_indexkey

	return nil
}

// 获取配置列表下标
func (c *Flexem_FlexEm) api_indexkey_R(v API_indexkey_struct) (mysql.Mqtt_Points__type, bool) {
	if v.DmonGroupName == "" || v.DmonName == "" {
		return mysql.Mqtt_Points__type{}, false
	}

	c.api_indexkey_RWMu.RLock()
	defer c.api_indexkey_RWMu.RUnlock()
	index, ok := c.api_indexkey[v]
	if !ok {
		return mysql.Mqtt_Points__type{}, false
	}
	if index < 0 || index >= len(c.Config.Points) {
		return mysql.Mqtt_Points__type{}, false
	}
	return c.Config.Points[index], true
}

// Stop 优雅关闭驱动（公开接口）
func (c *Flexem_FlexEm) Stop() error {
	log.Printf("INFO 正在关闭 Flexem_FlexEm 驱动 ID:%d", c.Config.Drive.Id)

	// 1. 注销 API 路由
	dataUrl, ok := cloud.GetKVValue(c.Config.Drive.Config, "数据推送url")
	if !ok || dataUrl == "" {
		log.Printf("ERROR 关闭失败 数据推送url是空的 【%s】", c.Config.Drive.Config)
		return fmt.Errorf("关闭失败 数据推送url是空的 【%s】", c.Config.Drive.Config)
	}

	err := web.UnregisterAPI(dataUrl)
	if err != nil {
		log.Printf("WARN 注销API失败: %s", err)
		// 继续执行清理，不返回错误
	}

	// 2. 清理内部 map，防止内存泄漏
	c.api_indexkey_RWMu.Lock()
	defer c.api_indexkey_RWMu.Unlock()
	c.api_indexkey = make(map[API_indexkey_struct]int) // 重新初始化为空 map
	// 3. 清空回调函数引用
	c.Callback_Push_External_Mappings = nil

	log.Printf("INFO Flexem_FlexEm 驱动 ID:%d 已完全关闭并清理资源", c.Config.Drive.Id)
	return nil
}

// 接收api数据
func (c *Flexem_FlexEm) dataHandler(ctx *gin.Context) {
	name, nameOk := cloud.GetKVValue(c.Config.Drive.Config, "用户名")
	passwd, passwdOk := cloud.GetKVValue(c.Config.Drive.Config, "密码")

	if nameOk && passwdOk {
		// 获取 Basic Auth 信息
		username, password, ok := ctx.Request.BasicAuth()

		if !ok {
			// 未提供认证信息或认证失败
			ctx.Header("WWW-Authenticate", "Basic realm=\"Restricted\"")
			ctx.Set("Response", []any{401, "Unauthorized"})
			return
		}

		// 使用获取到的用户名和密码
		if username != name || password != passwd {
			ctx.Set("Response", []any{401, "Forbidden"})
			return
		}
	}

	var jsondata []API_dataValue_struct
	if err := ctx.BindJSON(&jsondata); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	if len(jsondata) == 0 {
		ctx.Set("Response", []any{403, "null"})
		return
	}

	value_list := c.processAPIValues(jsondata)

	if c.Callback_Push_External_Mappings != nil {
		c.Callback_Push_External_Mappings(value_list)
	}
	ctx.Set("Response", []any{200, "ok"})
}

// processAPIValues 处理API推送的值
func (c *Flexem_FlexEm) processAPIValues(jsondata []API_dataValue_struct) []fullConfig.Value_type {
	var value_list []fullConfig.Value_type

	for _, v := range jsondata {
		cfg, ok := c.api_indexkey_R(API_indexkey_struct{
			DeviceSn:      v.DeviceSn,
			DmonGroupName: v.DmonGroupName,
			DmonName:      v.DmonName,
		})

		// 如果找不到对应的配置，尝试根据监控点分组名称和监控点名称构建一个临时的点位值
		if !ok {
			face, ok := cloud.GetKVValue(c.Config.Drive.Config, "临时点")
			if !(ok && face == "True") {
				continue
			}

			valueType, ok := ValueType_map[v.ValueType]
			if !ok {
				continue
			}
			value_list = append(value_list, fullConfig.Value_type{
				Tag:   fmt.Sprintf("//%s//%s/%s/%s", c.Config.Drive.Name, v.DeviceSn, v.DmonGroupName, v.DmonName),
				Value: c.extractValue(v, valueType),
				Type:  valueType,
				Msg:   "ok",
				Time:  v.Timestamp,
			})
			continue
		}
		value := c.buildValueFromAPI(v, cfg)
		if value.Tag == "" {
			continue
		}
		value_list = append(value_list, value)
	}

	return value_list
}

// buildValueFromAPI 根据API数据构建点位值
func (c *Flexem_FlexEm) buildValueFromAPI(v API_dataValue_struct, cfg mysql.Mqtt_Points__type) fullConfig.Value_type {
	// 检查读写权限
	if cfg.RW_Cancel != "R" {
		return fullConfig.Value_type{
			Tag:  cfg.Tag,
			Type: cfg.Value_Type,
			Msg:  fmt.Sprintf("读写方式只能是R，当前为【%s】", cfg.RW_Cancel),
			Time: v.Timestamp,
		}
	}

	// 转换值类型
	valueType, ok := ValueType_map[v.ValueType]
	if !ok {
		return fullConfig.Value_type{
			Tag:  cfg.Tag,
			Type: cfg.Value_Type,
			Msg:  fmt.Sprintf("未知推送值类型，当前为【%d】", v.ValueType),
			Time: v.Timestamp,
		}
	}

	// 检查值类型是否与配置一致
	if valueType != cfg.Value_Type {
		return fullConfig.Value_type{
			Tag:  cfg.Tag,
			Type: cfg.Value_Type,
			Msg:  fmt.Sprintf("推送值类型与配置值类型不一致，推送值类型为【%s】，配置值类型为【%s】", valueType, cfg.Value_Type),
			Time: v.Timestamp,
		}
	}

	// 获取状态信息
	status, ok := status_map[v.Error]
	if !ok {
		return fullConfig.Value_type{
			Tag:  cfg.Tag,
			Type: cfg.Value_Type,
			Msg:  fmt.Sprintf("未知监控点状态，当前为【%d】", v.Error),
			Time: v.Timestamp,
		}
	}

	// 提取实际值
	return fullConfig.Value_type{
		Tag:   cfg.Tag,
		Value: c.extractValue(v, valueType),
		Type:  valueType,
		Msg:   status,
		Time:  v.Timestamp,
	}
}

// extractValue 从API数据结构中提取对应类型的值
func (c *Flexem_FlexEm) extractValue(v API_dataValue_struct, valueType string) any {
	switch valueType {
	case "bool":
		return v.BoolValue
	case "int":
		return v.IntValue
	case "uint":
		return v.UIntValue
	case "float":
		return v.FloatValue
	case "string":
		return v.StringValue
	default:
		return nil
	}
}
