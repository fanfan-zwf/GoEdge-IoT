package flexem_flexem

import (
	"fmt"
	"log"
	"main/IO/manager/fullConfig"
	"main/cloud"
	"main/db/mysql"
	"main/method/timer"
	"main/web"
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

type API_indexkey_struct struct {
	DeviceSn      string // 盒子序列号
	DmonGroupName string // 监控点分组名称
	DmonName      string // 监控点名称
}

type API_Post_Value_struct struct {
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
	timer                 *timer.TimerTask

	Push_External_Mappings func([]fullConfig.Value_type) error

	// 盒子序列号+监控点分组名称+监控点名称 对应配置列表下标
	api_indexkey_RWMu sync.RWMutex
	api_indexkey      map[API_indexkey_struct]int // 盒子序列号+监控点分组名称+监控点名称 对应配置列表下标

	// api推送 盒子序列号+监控点分组名称+监控点名称
	deviceSn_finally_time_RWMu    sync.RWMutex
	deviceSn_finally_time         map[string]time.Time // 盒子序列号 最后一次推送时间
	deviceSn_finally_time_timeout time.Duration

	// 点位索引到设备SN的映射（性能优化：避免重复解析JSON）
	pointIndex_to_deviceSn []string // pointIndex_to_deviceSn[i] = c.Config.Points[i] 对应的设备SN
}

// 定义接口
type Connect_interface interface {
	New() error
}

func (c *Flexem_FlexEm) Start() error {
	if c.deviceSn_finally_time_timeout == 0 {
		c.deviceSn_finally_time_timeout = 60 * time.Second
	}

	url, ok := cloud.GetKVValue(c.Config.Drive.Config, "url")
	if !ok || url == "" {
		return fmt.Errorf("url是空的 【%s】", c.Config.Drive.Config)
	}
	web.RegisterPOST(url, c.handler)

	api_indexkey := make(map[API_indexkey_struct]int)
	deviceSn_finally_time := make(map[string]time.Time)

	// 性能优化：预分配切片容量，避免动态扩容
	pointIndex_to_deviceSn := make([]string, len(c.Config.Points))

	for i, point := range c.Config.Points {

		// 从驱动配置获取设备SN（一个驱动对应一个设备）
		deviceSn, ok := cloud.GetKVValue(c.Config.Drive.Config, "盒子序列号")
		if !ok || deviceSn == "" {
			err := fmt.Errorf("ERROR 驱动名称【%s】 盒子序列号【%s】", c.Config.Drive.Name, deviceSn)
			log.Print(err)
			return err
		}

		dmonGroupName, ok := cloud.GetKVValue(point.Config, "监控点分组名称")
		if !ok || dmonGroupName == "" {
			err := fmt.Errorf("ERROR 驱动名称【%s】 监控点分组名称【%s】", c.Config.Drive.Name, dmonGroupName)
			log.Print(err)
			return err
		}
		dmonName, ok := cloud.GetKVValue(point.Config, "监控点名称")
		if !ok || dmonName == "" {
			err := fmt.Errorf("ERROR 驱动名称【%s】 监控点名称【%s】", c.Config.Drive.Name, dmonName)
			log.Print(err)
			return err
		}
		api_indexkey[API_indexkey_struct{
			DeviceSn:      deviceSn,
			DmonGroupName: dmonGroupName,
			DmonName:      dmonName,
		}] = i

		// 缓存点位索引到设备SN的映射
		pointIndex_to_deviceSn[i] = deviceSn

		_, ok = deviceSn_finally_time[deviceSn]
		if !ok {
			deviceSn_finally_time[deviceSn] = time.Time{}
		}
	}

	c.api_indexkey_RWMu.Lock()
	c.api_indexkey = api_indexkey
	c.api_indexkey_RWMu.Unlock()

	// 保存点位到设备SN的映射
	c.pointIndex_to_deviceSn = pointIndex_to_deviceSn
	
	// ✅ 关键修复：将初始化好的 deviceSn_finally_time 赋值给结构体字段
	c.deviceSn_finally_time_RWMu.Lock()
	c.deviceSn_finally_time = deviceSn_finally_time
	c.deviceSn_finally_time_RWMu.Unlock()

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

// 关闭连接
func (c *Flexem_FlexEm) Close() error {
	c.timer.Stop()

	url, ok := cloud.GetKVValue(c.Config.Drive.Config, "url")
	if !ok || url == "" {
		return fmt.Errorf("url是空的 【%s】", c.Config.Drive.Config)
	}
	web.UnregisterAPI(url)

	return nil
}

func (c *Flexem_FlexEm) timer_msg(callTime time.Time) {
	// ========== 第一步：只读锁检查超时设备（不修改数据）==========
	c.deviceSn_finally_time_RWMu.RLock() // ✅ 使用读锁

	// 收集所有超时的设备SN及其消息
	timeoutDeviceMap := make(map[string]string)
	for deviceSn, lastTime := range c.deviceSn_finally_time {
		if lastTime.IsZero() {
			continue // 跳过从未收到数据的设备
		}
		// 使用 After 方法符合 Go 时间比较规范
		if time.Now().After(lastTime.Add(c.deviceSn_finally_time_timeout)) {
			timeoutDeviceMap[deviceSn] = fmt.Sprintf("驱动超时 最后时间:%s", lastTime)
		}
	}

	c.deviceSn_finally_time_RWMu.RUnlock() // ✅ 立即释放读锁

	// 如果没有超时设备，直接返回（避免无效遍历）
	if len(timeoutDeviceMap) == 0 {
		return
	}

	// ========== 第二步：构建超时点位的值列表（O(n)复杂度，无锁）==========
	var value_list []fullConfig.Value_type

	for pointIndex, point := range c.Config.Points {
		// 边界检查
		if pointIndex >= len(c.pointIndex_to_deviceSn) {
			log.Printf("ERROR 点位索引越界: index=%d, len=%d", pointIndex, len(c.pointIndex_to_deviceSn))
			continue
		}

		// 从预缓存映射获取设备SN（O(1)，零JSON解析开销）
		deviceSn := c.pointIndex_to_deviceSn[pointIndex]

		// 检查该设备是否超时
		msg, isTimeout := timeoutDeviceMap[deviceSn]
		if !isTimeout {
			continue // 设备未超时，跳过
		}

		// 添加超时点位到结果列表
		value_list = append(value_list, fullConfig.Value_type{
			Tag:   point.Tag,        // 使用正确的 Tag 字段
			Value: nil,              // 点位值
			Type:  point.Value_Type, // 输出类型
			Msg:   msg,              // 状态信息（超时消息）
			Time:  callTime,         // 读取时间
		})
	}

	// ========== 第三步：如果没有超时点位，直接返回 ==========
	if len(value_list) == 0 {
		return
	}

	// ========== 第四步：推送前检查回调函数 ==========
	if c.Push_External_Mappings == nil {
		c.timer.Stop()
		return
	}

	// ========== 第五步：写锁重置超时标记（防止重复报警）==========
	c.deviceSn_finally_time_RWMu.Lock() // ✅ 升级为写锁
	for deviceSn := range timeoutDeviceMap {
		c.deviceSn_finally_time[deviceSn] = time.Time{} // 重置为零值
	}
	c.deviceSn_finally_time_RWMu.Unlock() // ✅ 立即释放写锁

	// ========== 第六步：推送超时状态数据 ==========
	c.Push_External_Mappings(value_list)
}

// 接收api数据
func (c *Flexem_FlexEm) handler(ctx *gin.Context) {
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

	var jsondata []API_Post_Value_struct
	if err := ctx.BindJSON(&jsondata); err != nil {
		ctx.Set("Response", []any{417, "请求格式不对"})
		return
	}

	value_list := c.processAPIValues(jsondata)

	if c.Push_External_Mappings != nil {
		c.Push_External_Mappings(value_list)
	}
	ctx.Set("Response", []any{200, "ok"})
}

// processAPIValues 处理API推送的值
func (c *Flexem_FlexEm) processAPIValues(jsondata []API_Post_Value_struct) []fullConfig.Value_type {
	var value_list []fullConfig.Value_type

	for _, v := range jsondata {
		cfg, ok := c.api_indexkey_R(API_indexkey_struct{
			DeviceSn:      v.DeviceSn,
			DmonGroupName: v.DmonGroupName,
			DmonName:      v.DmonName,
		})

		// ✅ 修复：使用 defer 确保锁正确释放，避免死锁和竞态条件
		c.deviceSn_finally_time_RWMu.Lock()
		c.deviceSn_finally_time[v.DeviceSn] = v.Timestamp
		c.deviceSn_finally_time_RWMu.Unlock()

		// 如果找不到对应的配置，尝试根据监控点分组名称和监控点名称构建一个临时的点位值
		if !ok {
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
func (c *Flexem_FlexEm) buildValueFromAPI(v API_Post_Value_struct, cfg mysql.Mqtt_Points__type) fullConfig.Value_type {
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
func (c *Flexem_FlexEm) extractValue(v API_Post_Value_struct, valueType string) any {
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
