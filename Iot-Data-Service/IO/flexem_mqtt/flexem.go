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
	points_config_RWMu sync.RWMutex
	points_config_map  map[string]int // 点位配置下标

	push_finally_time_RWMu    sync.RWMutex
	push_finally_time_map     map[string]time.Time // 点位状态
	Push_finally_time_timeout time.Duration

	Push_External_Mappings func([]fullConfig.Value_type) error
}

// 定义接口
type Connect_interface interface {
	New() error
}

func (c *Flexem_Mqtt) New() (err error) {
	if c.Push_finally_time_timeout == 0 {
		c.Push_finally_time_timeout = 10 * time.Second
	}

	c.points_config_map = make(map[string]int)
	c.push_finally_time_map = make(map[string]time.Time)
	for i, v := range c.Config.Points {
		c.points_config_map_W(v.Tag, i)
		c.push_finally_time_map_W(v.Tag, time.Now())
	}
	c.timer = timer.NewTimerTask()
	c.timer.Start(c.Push_finally_time_timeout, c.timer_msg)
	return
}

// 关闭连接
func (c *Flexem_Mqtt) Close() {
	c.timer.Stop()
}

func (c *Flexem_Mqtt) timer_msg(callTime time.Time) {
	var value_list []fullConfig.Value_type
	for _, cfg := range c.Config.Points {
		var msg string
		t, ok := c.push_finally_time_map_R(cfg.Tag)
		if !ok {
			msg = "不存在的点位"
		} else if t.IsZero() {
			msg = "无首次最后时间"
		} else if time.Since(t) >= c.Push_finally_time_timeout {
			msg = fmt.Sprintf("超时 最后时间:%s", t)
		} else {
			fmt.Printf("mqtt 点位【%s】正常在线 \n", cfg.Tag)
			continue
		}
		fmt.Printf("mqtt 点位【%s】异常【%s】\n", cfg.Tag, msg)

		value_list = append(value_list, fullConfig.Value_type{
			Tag:   cfg.Tag,        // 点位名称
			Value: nil,            // 点位值
			Type:  cfg.Value_Type, // 输出类型
			Msg:   msg,            // 状态信息
			Time:  callTime,       // 读取时间
		})
	}

	if c.Push_External_Mappings == nil {
		c.timer.Stop()
		return
	}

	if len(value_list) != 0 {
		c.Push_External_Mappings(value_list)
	}
}

func (c *Flexem_Mqtt) points_config_map_W(tag string, i int) {
	c.points_config_RWMu.Lock()
	defer c.points_config_RWMu.Unlock()
	c.points_config_map[tag] = i
}
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

func (c *Flexem_Mqtt) push_finally_time_map_W(tag string, t time.Time) {
	c.push_finally_time_RWMu.Lock()
	defer c.push_finally_time_RWMu.Unlock()
	c.push_finally_time_map[tag] = t
}
func (c *Flexem_Mqtt) push_finally_time_map_R(tag string) (time.Time, bool) {
	c.push_finally_time_RWMu.RLock()
	defer c.push_finally_time_RWMu.RUnlock()
	t, ok := c.push_finally_time_map[tag]
	return t, ok
}

// 推送
func (c *Flexem_Mqtt) Push() (err error) {
	mqttbase.Subscribe(c.Config.Drive.Example_IDentifier, c.Config.Drive.Topic_Push, func(broker, topic string, dataByte []byte) {
		if len(dataByte) == 0 {
			return
		}

		if c.Config.Drive.Example_IDentifier != broker {
			return
		}

		if c.Config.Drive.Topic_Push != topic {
			return
		}

		compress, ok := cloud.GetKVValue(c.Config.Drive.Config, "压缩")
		if ok && compress != "" {
			dataByte, err = cloud.Decompress(dataByte)
			if err != nil {
				log.Printf("ERROR mqtt实例名称【%s】 配置压缩【%s】算法解压错误", c.Config.Drive.Name, compress)
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
		var Time time.Time
		t, ok := data_map["flexem_timestamp"]
		if ok {
			var unix int64
			err = json.Unmarshal(t, &unix)
			if err != nil {
				log.Printf("ERROR flexem_timestamp响应解析失败,以及转化当前时间: %v", err)
				Time = time.Now()
			} else {
				Time = time.Unix(unix, 0)
			}
		} else {
			Time = time.Now()
		}

		var value_list []fullConfig.Value_type
		for map_key, map_value := range data_map {
			if map_key == "flexem_timestamp" {
				continue
			}

			if string(map_value) == "null" {
				continue
			}

			cfg, ok := c.points_config_map_R(map_key)
			if !ok {
				continue
			}

			if cfg.RW_Cancel != "R" && cfg.RW_Cancel != "R/W" {
				continue
			}

			var realValue any
			switch cfg.Value_Type {
			case "bool":
				var v any
				err = json.Unmarshal(map_value, &v)
				realValue, _ = byte_util.ConvBool(v, byte_util.ValueType(v))
			case "int":
				var v int
				err = json.Unmarshal(map_value, &v)
				realValue = v
			case "uint":
				var v uint
				err = json.Unmarshal(map_value, &v)
				realValue = v
			case "float":
				var v float64
				err = json.Unmarshal(map_value, &v)
				realValue = v
			case "string":
				var v string
				err = json.Unmarshal(map_value, &v)
				realValue = v
			default:
				continue
			}

			if err != nil {
				continue
			}

			c.push_finally_time_map_W(cfg.Tag, Time)

			value_list = append(value_list, fullConfig.Value_type{
				Tag:   cfg.Tag,        // 点位名称
				Value: realValue,      // 点位值
				Type:  cfg.Value_Type, // 输出类型
				Msg:   "ok",           // 状态信息
				Time:  Time,           // 读取时间
			})
		}

		if c.Push_External_Mappings != nil {
			c.Push_External_Mappings(value_list)
		}
	})
	return
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
			return fmt.Errorf("没有[%s]点位", cfg.Tag)
		}

		if cfg.RW_Cancel != "W" && cfg.RW_Cancel != "R/W" {
			return fmt.Errorf("[%s]不是一个可以写的", cfg.Tag)
		}

		if cfg.Value_Type != value.Type {
			return fmt.Errorf("点位[%s]错误,请求类型[%s],实际类型[%s]", cfg.Tag, value.Type, cfg.Value_Type)
		}
		t := time.Until(value.Time).Abs()
		if t > (5 * time.Second) {
			return fmt.Errorf("[%s]点位超时[%s]", cfg.Tag, t)
		}

		r[value.Tag] = value.Value
	}

	dataByte, err := json.Marshal(r)
	if err != nil {
		return err
	}

	return mqttbase.Send(c.Config.Drive.Example_IDentifier, c.Config.Drive.Topic_Down, dataByte)
}
