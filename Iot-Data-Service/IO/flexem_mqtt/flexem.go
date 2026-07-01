package flexem_mqtt

import (
	"fmt"
	"main/IO/byte_util"
	"main/IO/manager/fullConfig"
	"main/app/mqttbase"
	"main/cloud"
	"main/db/mysql"
	"main/method/timer"

	"encoding/json"
	"log"
	"sync"
	"time"
)

// 定义一个结构体
type Flexem_Mqtt struct {
	fullConfig.BaseDriver // 驱动全配置（驱动配置 + 该驱动下的所有点位配置）
	timer                 *timer.TimerTask
	// Drive                 mysql.Mqtt__type        // 通信参数结构体
	// Points                []mysql.Mqtt_Points__type //  // 点位结构体

	// 点位标志与配置下标
	points_config_RWMu sync.RWMutex
	points_config_map  map[string]int // 点位配置下标

	// 推送key值
	push_key_map_RWMu sync.RWMutex
	push_key_map      map[string]int

	// 设备最后一次推送时间
	push_finally_time_RWMu    sync.RWMutex
	push_finally_time         time.Time // 设备最后一次推送时间
	Push_finally_time_timeout time.Duration

	Push_External_Mappings func([]fullConfig.Value_type) error
}

// 定义接口
type Connect_interface interface {
	New() error
}

func (c *Flexem_Mqtt) New() error {
	if c.Push_finally_time_timeout == 0 {
		c.Push_finally_time_timeout = 10 * time.Second
	}

	// 初始化点位配置映射
	c.points_config_map = make(map[string]int)
	c.push_key_map = make(map[string]int)

	for i, v := range c.Config.Points {
		// 构建点位标签到索引的映射
		c.points_config_RWMu.Lock()
		c.points_config_map[v.Tag] = i
		c.points_config_RWMu.Unlock()

		// 构建MQTT变量名称到索引的映射
		mqttName, ok := cloud.GetKVValue(v.Config, "MQTT变量名称")
		if !ok || mqttName == "" {
			log.Printf("ERROR 点位【%s】 MQTT变量名称配置缺失", v.Tag)
			continue
		}
		
		c.push_key_map_RWMu.Lock()
		c.push_key_map[mqttName] = i
		c.push_key_map_RWMu.Unlock()
	}

	// 启动定时器任务
	c.timer = timer.NewTimerTask()
	c.timer.Start(c.Push_finally_time_timeout, c.timer_msg)
	
	return nil
}

// 关闭连接
func (c *Flexem_Mqtt) Close() {
	c.timer.Stop()
}

func (c *Flexem_Mqtt) timer_msg(callTime time.Time) {
	// 检查最后一次推送时间
	c.push_finally_time_RWMu.RLock()
	t := c.push_finally_time
	c.push_finally_time_RWMu.RUnlock()

	var msg string
	if t.IsZero() {
		return // 从未收到过数据，不触发超时
	} else if time.Since(t) >= c.Push_finally_time_timeout {
		msg = fmt.Sprintf("驱动超时 最后时间:%v", t)
	} else {
		return // 未超时
	}

	// 重置推送时间（使用写锁）
	c.push_finally_time_RWMu.Lock()
	c.push_finally_time = time.Time{}
	c.push_finally_time_RWMu.Unlock()

	// 构建超时状态的值列表
	var value_list []fullConfig.Value_type
	for _, cfg := range c.Config.Points {
		value_list = append(value_list, fullConfig.Value_type{
			Tag:   cfg.Tag,
			Value: nil,
			Type:  cfg.Value_Type,
			Msg:   msg,
			Time:  callTime,
		})
	}

	if len(value_list) == 0 || c.Push_External_Mappings == nil {
		return
	}

	c.Push_External_Mappings(value_list)
}

// 获取点位配置下标
func (c *Flexem_Mqtt) points_config_map_R(tag string) (mysql.Mqtt_Points__type, bool) {
	c.points_config_RWMu.RLock()
	defer c.points_config_RWMu.RUnlock()
	index, ok := c.points_config_map[tag]
	if !ok {
		return mysql.Mqtt_Points__type{}, false
	}
	if index < 0 || index >= len(c.Config.Points) {
		return mysql.Mqtt_Points__type{}, false
	}
	return c.Config.Points[index], true
}

// 获取mqtt点位名称的配置
func (c *Flexem_Mqtt) push_key_map_R(mqttName string) (mysql.Mqtt_Points__type, bool) {
	c.push_key_map_RWMu.RLock()
	defer c.push_key_map_RWMu.RUnlock()
	index, ok := c.push_key_map[mqttName]
	if !ok {
		return mysql.Mqtt_Points__type{}, false
	}
	if index < 0 || index >= len(c.Config.Points) {
		return mysql.Mqtt_Points__type{}, false
	}
	return c.Config.Points[index], true
}

// 推送
func (c *Flexem_Mqtt) Push() (err error) {
	topic_Push, ok := cloud.GetKVValue(c.Config.Drive.Config, "推送")
	if !ok || topic_Push == "" {
		err := fmt.Errorf("ERROR mqtt实例名称【%s】 推送值错误【%s】", c.Config.Drive.Name, topic_Push)
		log.Print(err)
		return err
	}

	example_IDentifier, ok := cloud.GetKVValue(c.Config.Drive.Config, "MQTT实例")
	if !ok || example_IDentifier == "" {
		err := fmt.Errorf("ERROR mqtt实例名称【%s】 MQTT实例【%s】", c.Config.Drive.Name, example_IDentifier)
		log.Print(err)
		return err
	}

	mqttbase.Subscribe(example_IDentifier, topic_Push, func(broker, topic string, dataByte []byte) {
		if len(dataByte) == 0 {
			return
		}

		if topic_Push != broker {
			return
		}

		if topic_Push != topic {
			return
		}

		compress, ok := cloud.GetKVValue(c.Config.Drive.Config, "压缩")
		if !ok || compress == "" {
			dataByte, err = cloud.Decompress(dataByte)
			if err != nil {
				err := fmt.Errorf("ERROR mqtt实例名称【%s】 配置压缩【%s】算法解压错误", c.Config.Drive.Name, compress)
				log.Print(err)
				return
			}
		}

		data_map := make(map[string]json.RawMessage)

		// 绑定JSON，只解析Type，不解析Value
		err := json.Unmarshal(dataByte, &data_map)
		if err != nil {
			log.Printf("ERROR 响应解析失败: %v", err)
			return
		}

		// 解析时间戳
		var Time time.Time
		if timestampData, ok := data_map["flexem_timestamp"]; ok {
			var unix int64
			if err := json.Unmarshal(timestampData, &unix); err != nil {
				log.Printf("ERROR flexem_timestamp解析失败: %v，使用当前时间", err)
				Time = time.Now()
			} else {
				Time = time.Unix(unix, 0)
			}
		} else {
			Time = time.Now()
		}

		// 更新最后一次推送时间（使用写锁）
		c.push_finally_time_RWMu.Lock()
		c.push_finally_time = Time
		c.push_finally_time_RWMu.Unlock()

		var value_list []fullConfig.Value_type
		for map_key, map_value := range data_map {
			// 跳过时间戳字段
			if map_key == "flexem_timestamp" {
				continue
			}

			// 查找点位配置
			cfg, ok := c.push_key_map_R(map_key)
			if !ok {
				log.Printf("WARN MQTT键【%s】未找到对应点位配置", map_key)
				continue
			}

			// 检查读写权限
			if cfg.RW_Cancel != "R" && cfg.RW_Cancel != "R/W" {
				continue
			}

			// 处理null值
			if string(map_value) == "null" {
				value_list = append(value_list, fullConfig.Value_type{
					Tag:   cfg.Tag,
					Value: nil,
					Type:  cfg.Value_Type,
					Msg:   "无数据",
					Time:  Time,
				})
				continue
			}

			// 根据类型解析值
			realValue, parseErr := c.parseMQTTValue(map_value, cfg.Value_Type)
			if parseErr != nil {
				log.Printf("ERROR 点位【%s】值解析失败: %v", cfg.Tag, parseErr)
				continue
			}

			value_list = append(value_list, fullConfig.Value_type{
				Tag:   cfg.Tag,
				Value: realValue,
				Type:  cfg.Value_Type,
				Msg:   "ok",
				Time:  Time,
			})
		}

		if len(value_list) > 0 && c.Push_External_Mappings != nil {
			c.Push_External_Mappings(value_list)
		}
	})
	return
}

// parseMQTTValue 解析MQTT消息中的值
func (c *Flexem_Mqtt) parseMQTTValue(data json.RawMessage, valueType string) (any, error) {
	switch valueType {
	case "bool":
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("bool解析失败: %w", err)
		}
		result, _ := byte_util.ConvBool(v, byte_util.ValueType(v))
		return result, nil
	case "int":
		var v int
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("int解析失败: %w", err)
		}
		return v, nil
	case "uint":
		var v uint
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("uint解析失败: %w", err)
		}
		return v, nil
	case "float":
		var v float64
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("float解析失败: %w", err)
		}
		return v, nil
	case "string":
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("string解析失败: %w", err)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("不支持的类型: %s", valueType)
	}
}

// 下发
func (c *Flexem_Mqtt) Down(values []fullConfig.Value_type) error {
	if len(values) == 0 {
		return nil
	}

	r := make(map[string]any)
	r["flexem_timestamp"] = time.Now().Unix()

	for _, value := range values {
		cfg, ok := c.points_config_map_R(value.Tag)
		if !ok {
			return fmt.Errorf("没有[%s]点位配置", value.Tag)
		}

		// 检查写权限
		if cfg.RW_Cancel != "W" && cfg.RW_Cancel != "R/W" {
			return fmt.Errorf("[%s]不允许写入，当前权限为[%s]", cfg.Tag, cfg.RW_Cancel)
		}

		// 检查类型匹配
		if cfg.Value_Type != value.Type {
			return fmt.Errorf("点位[%s]类型不匹配: 请求类型[%s], 实际类型[%s]", cfg.Tag, value.Type, cfg.Value_Type)
		}

		// 检查时间有效性
		t := time.Until(value.Time).Abs()
		if t > (3 * time.Second) {
			return fmt.Errorf("[%s]点位超时[%v]", cfg.Tag, t)
		}

		// 获取MQTT变量名称
		mqttName, ok := cloud.GetKVValue(cfg.Config, "MQTT变量名称")
		if !ok || mqttName == "" {
			return fmt.Errorf("点位[%s]的MQTT变量名称配置缺失", cfg.Tag)
		}

		r[mqttName] = value.Value
	}

	// 序列化数据
	dataByte, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

	// 获取推送主题
	topic_Push, ok := cloud.GetKVValue(c.Config.Drive.Config, "推送")
	if !ok || topic_Push == "" {
		return fmt.Errorf("驱动【%s】的推送主题配置缺失", c.Config.Drive.Name)
	}

	// 获取MQTT实例标识
	example_IDentifier, ok := cloud.GetKVValue(c.Config.Drive.Config, "MQTT实例")
	if !ok || example_IDentifier == "" {
		return fmt.Errorf("驱动【%s】的MQTT实例配置缺失", c.Config.Drive.Name)
	}

	return mqttbase.Send(example_IDentifier, topic_Push, dataByte)
}
