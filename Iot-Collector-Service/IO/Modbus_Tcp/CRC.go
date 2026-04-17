package Modbus_Tcp

// modbusCRCTable Modbus CRC-16查表（预计算）
var modbusCRCTable []uint16

func init() {
	// 初始化Modbus CRC查表
	modbusCRCTable = make([]uint16, 256)
	for i := 0; i < 256; i++ {
		crc := uint16(i)
		for j := 0; j < 8; j++ {
			if crc&1 == 1 {
				crc = (crc >> 1) ^ 0xA001 // 0x8005的位反射形式
			} else {
				crc = crc >> 1
			}
		}
		modbusCRCTable[i] = crc
	}
}

// ModbusCRC16 计算Modbus CRC-16校验值
func ModbusCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)

	for _, b := range data {
		crc = (crc >> 8) ^ modbusCRCTable[byte(crc)^b]
	}

	return crc
}

// AddModbusCRC 为数据添加Modbus CRC校验码（小端序）
func AddModbusCRC(data []byte) []byte {
	crc := ModbusCRC16(data)
	return append(data, byte(crc), byte(crc>>8))
}

// VerifyModbusCRC 验证数据的Modbus CRC校验码
func VerifyModbusCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	// 提取接收到的CRC值（小端序）
	receivedCRC := uint16(data[len(data)-1])<<8 | uint16(data[len(data)-2])

	// 计算数据的CRC（不包括最后两个字节）
	calculatedCRC := ModbusCRC16(data[:len(data)-2])

	return receivedCRC == calculatedCRC
}

// ModbusCRC16Slow 慢速但易于理解的Modbus CRC-16实现
func ModbusCRC16Slow(data []byte) uint16 {
	crc := uint16(0xFFFF)

	for _, b := range data {
		crc ^= uint16(b)

		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc = crc >> 1
			}
		}
	}

	return crc
}

// func main() {
// 	// Modbus请求帧（不包含CRC）
// 	request := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x01}

// 	// 计算CRC
// 	crc := ModbusCRC16(request)
// 	fmt.Printf("Modbus CRC-16: 0x%04X\n", crc)

// 	// 添加CRC到数据
// 	frame := AddModbusCRC(request)
// 	fmt.Printf("完整Modbus帧: % X\n", frame)

// 	// 验证CRC
// 	isValid := VerifyModbusCRC(frame)
// 	fmt.Printf("CRC验证结果: %v\n", isValid)

// 	// 比较快速实现和慢速实现的结果
// 	crcSlow := ModbusCRC16Slow(request)
// 	fmt.Printf("慢速实现结果: 0x%04X, 结果一致: %v\n", crcSlow, crc == crcSlow)
// }
