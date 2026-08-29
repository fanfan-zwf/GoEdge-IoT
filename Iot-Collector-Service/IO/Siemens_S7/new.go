package siemenss7

import (
	"main/cloud"
	"time"
)

const (
	address_index         = 0 // 地址
	rack_index            = 1 // 机架号
	slot_index            = 2 // 槽位号
	retryTimeout_index    = 3 // 重试间隔（可选，默认3s）
	delayBetween_index    = 4 // 轮询间隔（可选，默认0）
	connectTimeout_index  = 5 // 连接超时（可选，默认10s）
	responseTimeout_index = 6 // 响应超时（可选，默认10s）
	maxPacketLen_index    = 7 // 组包最大长度（可选，默认480） Smart200=240 300=240 400=960 1200=480 1500=960
	connectionType_index  = 8 // 连接类型（可选，默认2） 1=PG 2=OP 3=Basic
)

// Drive_Config_Switch 解析驱动配置字符串 192.168.1.1;0;2;3s;0;10s;10s;480;2
func Drive_Config_Switch(configStr string) (c Config_type, err error) {

	c.Address, err = cloud.GetSplitStr(configStr, address_index, "IP")
	if err != nil {
		return
	}

	c.Rack, err = cloud.GetSplitInt(configStr, rack_index, "机架号")
	if err != nil {
		return
	}

	c.Slot, err = cloud.GetSplitInt(configStr, slot_index, "槽位号")
	if err != nil {
		return
	}

	// 重试间隔（可选，默认3s）
	c.Retry_timeout, err = cloud.GetSplitDuration(configStr, retryTimeout_index, "重试间隔")
	if err != nil {
		c.Retry_timeout = 3 * time.Second
		err = nil
	}

	// 轮询间隔（可选，默认0）
	c.Delay_between_polls, err = cloud.GetSplitDuration(configStr, delayBetween_index, "轮询间隔")
	if err != nil {
		c.Delay_between_polls = 0
		err = nil
	}

	// 连接超时（可选，默认10s）
	c.Connect_timeout, err = cloud.GetSplitDuration(configStr, connectTimeout_index, "连接超时")
	if err != nil {
		c.Connect_timeout = 10 * time.Second
		err = nil
	}

	// 响应超时（可选，默认10s）
	c.Response_timeout, err = cloud.GetSplitDuration(configStr, responseTimeout_index, "响应超时")
	if err != nil {
		c.Response_timeout = 10 * time.Second
		err = nil
	}

	// 组包最大长度（可选，默认480）
	c.MaxPacketLen, err = cloud.GetSplitInt(configStr, maxPacketLen_index, "组包最大长度")
	if err != nil {
		c.MaxPacketLen = 0 // 0 表示使用默认 PDU 大小 480
		err = nil
	}

	// 连接类型（可选，默认2） 1=PG编程设备 2=OP操作面板 3=Basic
	c.ConnectionType, err = cloud.GetSplitInt(configStr, connectionType_index, "连接类型")
	if err != nil {
		c.ConnectionType = 2 // 默认 OP 连接
		err = nil
	}

	return
}

// 点位配置 index
const (
	area_index         = 0 // 存储区（默认 132=DB块）
	dbNumber_index     = 1 // DB块号
	start_index        = 2 // 字节偏移
	type_index         = 3 // 采集数据类型
	childAddress_index = 4 // 子地址（可选）
)

// Point_Config_Switch 解析点位配置字符串 132;1;0;12;0
func Point_Config_Switch(configStr string) (point Points_type, err error) {

	// Area（存储区，可选，默认 132=DB块）
	point.Area, err = cloud.GetSplitInt(configStr, area_index, "存储区")
	if err != nil {
		point.Area = 0x84 // 默认 DB块
		err = nil
	}

	// DBNumber（DB块号，必填）
	point.DBNumber, err = cloud.GetSplitInt(configStr, dbNumber_index, "DB块号")
	if err != nil {
		return
	}

	// Start（字节偏移，必填）
	point.Start, err = cloud.GetSplitInt(configStr, start_index, "字节偏移")
	if err != nil {
		return
	}

	// Type（采集数据类型，必填）
	point.Type, err = cloud.GetSplitInt(configStr, type_index, "数据类型")
	if err != nil {
		return
	}

	// Child_Address（子地址，可选）
	point.Child_Address, err = cloud.GetSplitInt(configStr, childAddress_index, "子地址")
	if err != nil {
		point.Child_Address = 0
		err = nil
	}

	return
}
