package siemenss7

import (
	"main/cloud"
	"time"
)

const (
	address_index       = 0 // 地址
	Retry_timeout_index = 1 // 重试间隔
	delayBetween_index  = 2 // 轮询间隔（可选，默认20ms）

	rack_index = 3 // 机架号
	slot_index = 4 // 槽位号

)

// Drive_Config_Switch 解析驱动配置字符串 192.168.1.1;0;2;20ms
func Drive_Config_Switch(configStr string) (c Config_type, err error) {

	c.Address, err = cloud.GetSplitStr(configStr, address_index, "IP")
	if err != nil {
		return
	}

	// 重试间隔（可选，默认3s）
	c.Retry_timeout, err = cloud.GetSplitDuration(configStr, Retry_timeout_index, "重试间隔")
	if err != nil {
		c.Retry_timeout = 3 * time.Second
		err = nil // 可选字段，不报错
	}

	// 轮询间隔（可选，默认 20ms）
	c.Delay_between_polls, err = cloud.GetSplitDuration(configStr, delayBetween_index, "轮询间隔")
	if err != nil {
		c.Delay_between_polls = 20 * time.Millisecond
		err = nil // 可选字段，不报错
	}

	c.Rack, err = cloud.GetSplitInt(configStr, rack_index, "机架号")
	if err != nil {
		return
	}

	c.Slot, err = cloud.GetSplitInt(configStr, slot_index, "槽位号")
	if err != nil {
		return
	}
	return
}

func Point_Config_Switch(configStr string) {}

const (
	tcpDevice = "127.0.0.1"
	rack      = 0
	slot      = 2
)
