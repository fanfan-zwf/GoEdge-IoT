/*
* 日期: 2025.5.29 PM10:17
* 作者: 范范zwf
* 作用: 字节转换
 */

package byte_convert

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

/*
* 功能: byte转换bool
* 顺序: AB
 */
func Byte_Convert_1byte_8bool(b byte) ([8]bool, error) {
	var bools [8]bool
	for i := 0; i < 8; i++ {
		bools[i] = (b & (1 << i)) != 0
	}
	return bools, nil
}

/*
* 功能: 8bool转换1byte
* 顺序: AB
 */
func Byte_Convert_8bool_1byte(boolList [8]bool) ([1]byte, error) {
	var resultByte [1]byte
	for i, bit := range boolList {
		if bit {
			// 如果该位为 true，则通过位或操作设置到字节的相应位置上
			resultByte[0] |= 1 << (7 - i) // 注意位的位置，可根据需求调整顺序
		}
	}
	return resultByte, nil
}

/*
* 功能: [2]byte转换bool
* 顺序: AB
 */
func Byte_Convert_2byte_16bool(bytes [2]byte, byte_order string) ([16]bool, error) {

	switch byte_order {
	case "AB":
		var bools [16]bool
		for i := 0; i < 2; i++ {
			for j := 0; j < 8; j++ {
				// 计算比特位：从高位（MSB）到低位（LSB）依次提取
				bit := (bytes[i] >> (7 - j)) & 1
				bools[i*8+j] = bit != 0
			}
		}
		return bools, nil
	case "BA":
		var bools [16]bool
		for i := 0; i < 2; i++ {
			for j := 0; j < 8; j++ {
				// 从低位(LSB)到高位(MSB)提取比特位
				bit := (bytes[i] >> j) & 1
				bools[i*8+j] = bit != 0
			}
		}
		return bools, nil
	}

	return [16]bool{}, errors.New("顺序输入错误")
}

/*
* 功能: uint16转换[2]byte
* 顺序: AB
 */
func Byte_Convert_uint16_byte(n uint16, byte_order string) ([2]byte, error) {
	switch byte_order {
	case "AB":
		return [2]byte{
			byte(n >> 8), // 高8位
			byte(n),      // 低8位
		}, nil
	case "BA":
		return [2]byte{
			byte(n),      // 低8位
			byte(n >> 8), // 高8位
		}, nil
	default:
		return [2]byte{}, errors.New("顺序输入错误")
	}
}

/*
* 功能: uint16转换bool
* 顺序: AB
 */
func Byte_Convert_uint16_bool(num uint16, byte_order string) ([16]bool, error) {
	var re [16]bool
	var i int8
	func_num, _ := Byte_Convert_uint16_byte(num, byte_order)
	for _, v := range func_num {
		func_v, _ := Byte_Convert_1byte_8bool(v)
		for _, b := range func_v {
			re[i] = b
			i += 1
		}
	}
	return re, nil
}

/*
* 功能: byte 转换 uint16
* 顺序: AB,BA
 */
func Byte_Convert_byte_uint16(b [2]byte, byte_order string) (uint16, error) {
	switch byte_order {
	case "AB":
		return uint16(b[0])<<8 | uint16(b[1]), nil
	case "BA":
		return uint16(b[0]) | uint16(b[1])<<8, nil
	}
	return 0, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, b)
}

/*
* 功能: byte 转换 int16
* 顺序: AB,BA
 */
func Byte_Convert_byte_int16(b [2]byte, byte_order string) (int16, error) {
	switch byte_order {
	case "AB":
		return int16(b[0])<<8 | int16(b[1]), nil
	case "BA":
		return int16(b[0]) | int16(b[1])<<8, nil
	}
	return 0, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, b)
}

/*
* 功能: byte 转换 uint32
* 顺序: AB,BA
 */
func Byte_Convert_byte_uint32(b [4]byte, byte_order string) (uint32, error) {
	switch byte_order {
	case "ABCD": //标准大端序
		return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), nil
	case "CDAB": //标准大端序
		return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
	case "BADC": //标准大端序
		high := uint32(b[0])<<8 | uint32(b[1])
		low := uint32(b[2])<<8 | uint32(b[3])
		return low<<16 | high, nil
	case "DCBA": //标准大端序
		low := uint32(b[0]) | uint32(b[1])<<8
		high := uint32(b[2]) | uint32(b[3])<<8
		return high<<16 | low, nil
	default:
		return 0, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, b)
	}
}

/*
* 功能: byte 转换 uint32
* 顺序: AB,BA
 */
func Byte_Convert_byte_int32(b [4]byte, byte_order string) (int32, error) {
	switch byte_order {
	case "ABCD": //标准大端序
		return int32(b[0])<<24 | int32(b[1])<<16 | int32(b[2])<<8 | int32(b[3]), nil
	case "CDAB": //标准大端序
		return int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16 | int32(b[3])<<24, nil
	case "BADC": //标准大端序
		high := int32(b[0])<<8 | int32(b[1])
		low := int32(b[2])<<8 | int32(b[3])
		return low<<16 | high, nil
	case "DCBA": //标准大端序
		low := int32(b[0]) | int32(b[1])<<8
		high := int32(b[2]) | int32(b[3])<<8
		return high<<16 | low, nil
	default:
		return 0, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, b)
	}
}

/*
* 功能: byte 转换 float32
* 顺序: AB,BA
 */
func Byte_Convert_byte_float32(b [4]byte, byte_order string) (float32, error) {
	var bits uint32
	switch byte_order {
	case "ABCD": // 大端序
		bits = uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	case "CDAB": // 字交换序
		bits = uint32(b[2])<<24 | uint32(b[3])<<16 | uint32(b[0])<<8 | uint32(b[1])
	case "BADC": // 字节交换序
		bits = uint32(b[1])<<24 | uint32(b[0])<<16 | uint32(b[3])<<8 | uint32(b[2])
	case "DCBA": // 小端序
		bits = uint32(b[3])<<24 | uint32(b[2])<<16 | uint32(b[1])<<8 | uint32(b[0])
	default:
		return 0.0, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, b)
	}
	return math.Float32frombits(bits), nil
}

/*
* 功能: uint16 转换 int32
* 顺序: AB,BA
 */
func Byte_Convert_uint16_int32(uint_16 [2]uint16, byte_order string) (int32, error) {
	b0, err := Byte_Convert_uint16_byte(uint_16[0], "AB")
	if err != nil {
		return 0, err
	}
	b1, err := Byte_Convert_uint16_byte(uint_16[1], "AB")
	if err != nil {
		return 0, err
	}
	swapped := [4]byte{b0[0], b0[1], b1[0], b1[1]}
	return Byte_Convert_byte_int32(swapped, byte_order)
}

/*
* 功能: uint16 转换 uint32
* 顺序: AB,BA
 */
func Byte_Convert_uint16_uint32(uint_16 [2]uint16, byte_order string) (uint32, error) {
	b0, err := Byte_Convert_uint16_byte(uint_16[0], "AB")
	if err != nil {
		return 0, err
	}
	b1, err := Byte_Convert_uint16_byte(uint_16[1], "AB")
	if err != nil {
		return 0, err
	}
	swapped := [4]byte{b0[0], b0[1], b1[0], b1[1]}
	return Byte_Convert_byte_uint32(swapped, byte_order)
}

/*
* 功能: uint16 转换 float32
* 顺序: AB,BA
 */
func Byte_Convert_uint16_float32(uint_16 [2]uint16, byte_order string) (float32, error) {
	b0, err := Byte_Convert_uint16_byte(uint_16[0], "AB")
	if err != nil {
		return 0, err
	}
	b1, err := Byte_Convert_uint16_byte(uint_16[1], "AB")
	if err != nil {
		return 0, err
	}
	swapped := [4]byte{b0[0], b0[1], b1[0], b1[1]}
	return Byte_Convert_byte_float32(swapped, byte_order)
}

func Byte_Convert_bool_uint16(b [16]bool, byte_order string) (uint16, error) {
	var value uint16
	switch byte_order {
	case "AB":
		for i := 0; i < 16; i++ {
			if b[i] {
				value |= 1 << i
			}
		}
		return value, nil
	case "BA":
		for i := 0; i < 16; i++ {
			if b[i] {
				value |= 1 << (15 - i) // 第i个布尔值放在第(15-i)位（从高到低）
			}
		}
		return value, nil
	default:
		return 0, fmt.Errorf("%s是一个不能用的顺序 %v", byte_order, b)
	}
}

/*
* 功能: [*]bool 转换 uint16
* 顺序: AB,BA
* value 新值; child_address 子地址; current 当前16为值; byte_order字节顺序
 */
func Byte_Convert_child_bool_uint16(value bool, child_address uint8, current uint16, byte_order string) (uint16, error) {
	b, err := Byte_Convert_uint16_bool(current, byte_order)
	if err != nil {
		return 0, err
	}
	fmt.Print(b, value, child_address, "\n")
	b[child_address] = value
	return Byte_Convert_bool_uint16(b, byte_order)
}

/*
* 功能: int16 转换 [2]byte
* 顺序: AB,BA
 */
func Byte_Convert_int16_byte(n int16, byte_order string) ([2]byte, error) {
	switch byte_order {
	case "AB":
		return [2]byte{
			byte(n >> 8), // 高8位
			byte(n),      // 低8位
		}, nil
	case "BA":
		return [2]byte{
			byte(n),      // 低8位
			byte(n >> 8), // 高8位
		}, nil
	}
	return [2]byte{}, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, n)
}

/*
* 功能: int16 转换 uint16
* 顺序: AB,BA
 */
func Byte_Convert_int16_uint16(num int16, byte_order string) (uint16, error) {
	b, err := Byte_Convert_int16_byte(num, "AB")
	if err != nil {
		return 0, err
	}
	return Byte_Convert_byte_uint16(b, byte_order)
}

/*
* 功能: uint32 转换 [4]byte
* 顺序: AB,BA
 */
func Byte_Convert_uint32_byte(num uint32, byte_order string) ([4]byte, error) {
	swapped := [4]byte{}
	b := []byte{byte(num >> 24), byte(num >> 16), byte(num >> 8), byte(num)}
	switch byte_order {
	case "ABCD": //标准大端序
		swapped = [4]byte{b[0], b[1], b[2], b[3]}
	case "CDAB": //标准大端序
		swapped = [4]byte{b[2], b[3], b[0], b[1]}
	case "BADC": //标准大端序
		swapped = [4]byte{b[1], b[0], b[3], b[2]}
	case "DCBA": //标准大端序
		swapped = [4]byte{b[3], b[2], b[1], b[0]}
	default:
		return swapped, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, b)
	}
	return swapped, nil
}

/*
* 功能: int32 转换 [4]byte
* 顺序: AB,BA
 */
func Byte_Convert_int32_byte(num int32, byte_order string) ([4]byte, error) {
	swapped := [4]byte{}
	b := []byte{byte(num >> 24), byte(num >> 16), byte(num >> 8), byte(num)}
	switch byte_order {
	case "ABCD": //标准大端序
		swapped = [4]byte{b[0], b[1], b[2], b[3]}
	case "CDAB": //标准大端序
		swapped = [4]byte{b[2], b[3], b[0], b[1]}
	case "BADC": //标准大端序
		swapped = [4]byte{b[1], b[0], b[3], b[2]}
	case "DCBA": //标准大端序
		swapped = [4]byte{b[3], b[2], b[1], b[0]}
	default:
		return swapped, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, b)
	}
	return swapped, nil
}

/*
* 功能: float32 转换 [4]byte
* 顺序: AB,BA
 */
func Byte_Convert_float32_4byte(f float32, byte_order string) ([4]byte, error) {
	swapped := [4]byte{}
	num := math.Float32bits(f)
	b := []byte{byte(num >> 24), byte(num >> 16), byte(num >> 8), byte(num)}
	switch byte_order {
	case "ABCD": //标准大端序
		swapped = [4]byte{b[0], b[1], b[2], b[3]}
	case "CDAB": //标准大端序
		swapped = [4]byte{b[2], b[3], b[0], b[1]}
	case "BADC": //标准大端序
		swapped = [4]byte{b[1], b[0], b[3], b[2]}
	case "DCBA": //标准大端序
		swapped = [4]byte{b[3], b[2], b[1], b[0]}
	default:
		return swapped, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, b)
	}
	return swapped, nil
}

func Round_Float32(val float32, precision uint) float32 {
	ratio := math.Pow10(int(precision))
	return float32(math.Round(float64(val)*ratio) / ratio)

}

/*
* 功能: 将任意长度的 []bool转换为 []byte的实现，不足 8 个的部分自动补 0：
* 顺序：小端序转换实现
 */
func BoolsToBytesLittleEndian(bools []bool) []byte {
	// 计算需要的字节数（每8个bool用1个byte）
	byteCount := (len(bools) + 7) / 8
	bytes := make([]byte, byteCount)

	for i := 0; i < len(bools); i++ {
		if bools[i] {
			// 小端序：第一个bool在最低位(bit 0)，第二个在bit 1，依此类推
			byteIndex := i / 8
			bitPos := i % 8
			bytes[byteIndex] |= 1 << bitPos
		}
	}

	return bytes
}

/*
* 功能: 将任意长度的 [2]byte转换为 hex(int)的实现
* 顺序：小端序转换实现
 */
func Byte_Convert_2byte_hex_int(v [2]byte, byte_order string) (int, error) {
	var hex_string string
	switch byte_order {
	case "AB":
		hex_string = fmt.Sprintf("%d%d", v[0], v[1])
	case "BA":
		hex_string = fmt.Sprintf("%d%d", v[1], v[0])
	default:
		return 0, fmt.Errorf("%s是一个不能用的顺序 %x", byte_order, v)
	}
	num, err := strconv.Atoi(hex_string)
	if err != nil {
		return 0, err
	}

	return num, nil
}

/*
* 功能: 将任意长度的 [2]byte转换为 hex(int)的实现
* 顺序：小端序转换实现
 */
func Byte_Convert_16bool_2byte(bools [16]bool, byte_order string) ([2]byte, error) {
	var result [2]byte
	switch byte_order {
	case "AB":

		// 处理高字节（前8位）
		for i := 0; i < 8; i++ {
			if bools[i] {
				result[0] |= 1 << (7 - i) // 设置高位字节的对应位
			}
		}
		return result, nil
	case "BA":
		// 处理低字节（后8位）
		for i := 0; i < 8; i++ {
			if bools[i+8] {
				result[1] |= 1 << (7 - i) // 设置低位字节的对应位
			}
		}
		return result, nil
	default:
		return result, fmt.Errorf("%s是一个不能用的顺序 %v", byte_order, bools)
	}
}

// 浮点数四舍五入
func Round_float64(num float64, precision int) float64 {
	factor := math.Pow(10, float64(precision))
	// 通过 math.Copysign 正确处理正负数
	return math.Round(num*factor+math.Copysign(0.5, num)) / factor
}

// RemoveDuplicates 通用数组去重函数（以 int 数组为例）
func RemoveDuplicates[T comparable](arr []T) []T {
	// 创建一个空 map，键类型与数组元素一致，值为 bool 仅作标记
	seen := make(map[T]bool)
	// 用于存储去重后的结果
	result := []T{}

	// 遍历原数组
	for _, item := range arr {
		// 如果 map 中不存在该元素，则加入结果集并标记为已存在
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
