package modbus

import (
	"collector/pkg/conv"
	"encoding/json/v2"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Drive_Config_Switch 解析驱动配置 JSON 字符串
// JSON 格式示例: {"Address":"192.168.1.1:502","Retry_timeout":"3s","Connect_timeout":"10s","Response_timeout":"10s","Delay_between_polls":"0s","Packet_max":120,"Write_Coils_Function":15,"Write_Register_Function":16}
func Drive_Config_Switch(configStr string) (c Config_type, err error) {
	if err = json.Unmarshal([]byte(configStr), &c); err != nil {
		err = fmt.Errorf("Modbus配置JSON解析失败: %w", err)
		return
	}

	// IP地址（必填）
	if c.Address == "" {
		err = fmt.Errorf("Modbus配置缺少必填字段: Address")
		return
	}

	// 重试间隔（可选，默认3s）
	if c.Retry_timeout <= 0 {
		c.Retry_timeout = conv.Duration(3 * time.Second)
	}

	// 连接超时（可选，默认10s）
	if c.Connect_timeout <= 0 {
		c.Connect_timeout = conv.Duration(10 * time.Second)
	}

	// 响应超时（可选，默认10s）
	if c.Response_timeout <= 0 {
		c.Response_timeout = conv.Duration(10 * time.Second)
	}

	// 组包最大长度（可选，默认120）
	if c.Packet_max <= 0 {
		c.Packet_max = 120
	}

	// Delay_between_polls 默认0，无需额外处理

	return
}

// Modbus点位配置正则表达式常量
// 匹配格式: 从机地址:功能码:寄存器地址[.子地址]:字节序:数据类型
// 示例: 1:3:100:ABCD:float32 或 1:1:100.5:ABCD:bool
const Regex_Modbus_Point = `^(\d+):([1-4]):(\d+)(?:\.(\d+))?:(\w+):(\w+)$`

var modbusPointRegex = regexp.MustCompile(Regex_Modbus_Point)

// Point_Config_Switch 解析 Modbus 点位配置字符串
// 格式: 从机地址:功能码:寄存器地址[.子地址]:字节序:数据类型
// 示例: 1:3:100:ABCD:float32, 1:1:100.5:ABCD:bool
// 返回: (协议参数, 数据类型编码, 错误)
func Point_Config_Switch(s string) (point Points_type, valueType int, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return point, 0, fmt.Errorf("点位配置不能为空")
	}

	matches := modbusPointRegex.FindStringSubmatch(s)
	if matches == nil {
		return point, 0, fmt.Errorf("点位配置格式错误: %s, 期望格式: 从机地址:功能码:寄存器地址[.子地址]:字节序:数据类型", s)
	}

	// 解析从机地址 (0-255)
	slaveID, _ := strconv.Atoi(matches[1])
	if slaveID < 0 || slaveID > 255 {
		return point, 0, fmt.Errorf("从机地址超出范围(0-255): %d", slaveID)
	}

	// 解析功能码 (1-4)
	function, _ := strconv.Atoi(matches[2])

	// 解析寄存器地址 (0-65535)
	address, _ := strconv.Atoi(matches[3])
	if address < 0 || address > 65535 {
		return point, 0, fmt.Errorf("寄存器地址超出范围(0-65535): %d", address)
	}

	// 解析子地址（可选）
	var childAddress int
	if matches[4] != "" {
		childAddress, _ = strconv.Atoi(matches[4])
		if childAddress < 0 || childAddress > 15 {
			return point, 0, fmt.Errorf("子地址超出范围(0-15): %d", childAddress)
		}
	}

	// 解析字节序
	byteOrderStr := strings.ToUpper(matches[5])
	byteOrder, exists := conv.Byte_Value[byteOrderStr]
	if !exists {
		return point, 0, fmt.Errorf("无效的字节序: %s", byteOrderStr)
	}

	// 解析数据类型
	typeStr := strings.ToLower(matches[6])
	valueType, exists = conv.ValueTypeStringToInt(typeStr)
	if !exists {
		return point, 0, fmt.Errorf("无效的数据类型: %s", typeStr)
	}

	point = Points_type{
		SlaveID:       slaveID,      // 从机地址
		Function:      function,     // Modbus功能码
		Address:       address,      // 寄存器地址
		Child_Address: childAddress, // 子地址
		Byte_Order:    byteOrder,    // 字节序
	}

	return point, valueType, nil
}

// func Point_Config_Switch_List(configs []string) (points []Points_type, err error)

// func Read_Callback_v(v []Read_Value_type) {
// 	fmt.Printf("Read_Callback >>>>>>>> \n%+v \n", v)
// }
