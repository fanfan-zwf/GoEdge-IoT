/*
* 日期: 2026.4.16
* 作者: 范范zwf
* 作用: 字节转换工具（严格支持所有字节序 + 位操作 + 浮点数）
* 修复: 修复8字节序HGFECDAB错误、越界panic、位序定义、代码规范
 */

package byte_util

import (
	"math"
	"reflect"
)

// ==============================
// 字节序定义（完整 + 标准命名）
// 2字节：AB小端(低字节在前)  BA大端(高字节在前)
// 4/8字节：字母顺序 = 内存字节排列顺序
// ==============================
const (
	// 2字节
	AB = iota // 小端：低字节在前
	BA        // 大端：高字节在前（Modbus 默认）

	// 4字节
	ABCD // 大端（标准）
	BADC // 常用小端
	CDAB // 中端
	DCBA // 全反

	// 8字节
	ABCDEFGH
	BADCFEHG
	CDABGHEF
	DCBAHGFE
)

// 类型字节数量输出
var Byte_Value = map[string]int{

	// 2字节
	"AB": AB,
	"BA": BA,

	// 4字节
	"ABCD": ABCD,
	"BADC": BADC,
	"CDAB": CDAB,
	"DCBA": DCBA,

	// 8字节（8种完整顺序）
	"ABCDEFGH": ABCDEFGH,
	"BADCFEHG": BADCFEHG,
	"CDABGHEF": CDABGHEF,
	"DCBAHGFE": DCBAHGFE,
}

// ==============================
// 1. bool 批量互转
// ==============================
func BoolToBytes(in []bool) []byte {
	length := (len(in) + 7) / 8
	out := make([]byte, length)
	for i := 0; i < len(in); i++ {
		if in[i] {
			out[i/8] |= 1 << (i % 8)
		}
	}
	return out
}

func BytesToBool(in []byte) []bool {
	out := make([]bool, len(in)*8)
	for i := 0; i < len(out); i++ {
		out[i] = (in[i/8] & (1 << (i % 8))) != 0
	}
	return out
}

// ==============================
// 2. uint16 批量互转
// ==============================
func Uint16ToBytes(v []uint16, order int) []byte {
	out := make([]byte, len(v)*2)
	for i, n := range v {
		switch order {
		case BA:
			out[i*2] = byte(n >> 8)
			out[i*2+1] = byte(n)
		case AB:
			out[i*2] = byte(n)
			out[i*2+1] = byte(n >> 8)
		}
	}
	return out
}

func BytesToUint16(b []byte, order int) []uint16 {
	size := len(b) / 2
	out := make([]uint16, size)
	for i := 0; i < size; i++ {
		switch order {
		case BA:
			out[i] = uint16(b[i*2])<<8 | uint16(b[i*2+1])
		case AB:
			out[i] = uint16(b[i*2+1])<<8 | uint16(b[i*2])
		}
	}
	return out
}

// ==============================
// 3. int16 批量互转
// ==============================
func Int16ToBytes(v []int16, order int) []byte {
	out := make([]byte, len(v)*2)
	for i, n := range v {
		switch order {
		case BA:
			out[i*2] = byte(n >> 8)
			out[i*2+1] = byte(n)
		case AB:
			out[i*2] = byte(n)
			out[i*2+1] = byte(n >> 8)
		}
	}
	return out
}

func BytesToInt16(b []byte, order int) []int16 {
	size := len(b) / 2
	out := make([]int16, size)
	for i := 0; i < size; i++ {
		switch order {
		case BA:
			out[i] = int16(b[i*2])<<8 | int16(b[i*2+1])
		case AB:
			out[i] = int16(b[i*2+1])<<8 | int16(b[i*2])
		}
	}
	return out
}

// ==============================
// 4. uint32 批量互转
// ==============================
func Uint32ToBytes(v []uint32, order int) []byte {
	out := make([]byte, len(v)*4)
	for i, n := range v {
		switch order {
		case ABCD:
			out[i*4] = byte(n >> 24)
			out[i*4+1] = byte(n >> 16)
			out[i*4+2] = byte(n >> 8)
			out[i*4+3] = byte(n)
		case BADC:
			out[i*4] = byte(n >> 8)
			out[i*4+1] = byte(n >> 24)
			out[i*4+2] = byte(n >> 16)
			out[i*4+3] = byte(n)
		case CDAB:
			out[i*4] = byte(n >> 16)
			out[i*4+1] = byte(n >> 8)
			out[i*4+2] = byte(n >> 24)
			out[i*4+3] = byte(n)
		case DCBA:
			out[i*4] = byte(n)
			out[i*4+1] = byte(n >> 8)
			out[i*4+2] = byte(n >> 16)
			out[i*4+3] = byte(n >> 24)
		}
	}
	return out
}

func BytesToUint32(b []byte, order int) []uint32 {
	size := len(b) / 4
	out := make([]uint32, size)
	for i := 0; i < size; i++ {
		switch order {
		case ABCD:
			out[i] = uint32(b[i*4])<<24 | uint32(b[i*4+1])<<16 | uint32(b[i*4+2])<<8 | uint32(b[i*4+3])
		case BADC:
			out[i] = uint32(b[i*4+1])<<24 | uint32(b[i*4])<<16 | uint32(b[i*4+3])<<8 | uint32(b[i*4+2])
		case CDAB:
			out[i] = uint32(b[i*4+2])<<24 | uint32(b[i*4+3])<<16 | uint32(b[i*4])<<8 | uint32(b[i*4+1])
		case DCBA:
			out[i] = uint32(b[i*4+3])<<24 | uint32(b[i*4+2])<<16 | uint32(b[i*4+1])<<8 | uint32(b[i*4])
		}
	}
	return out
}

// ==============================
// 5. int32 批量互转
// ==============================
func Int32ToBytes(v []int32, order int) []byte {
	return Uint32ToBytes(uint32Slice(v), order)
}

func BytesToInt32(b []byte, order int) []int32 {
	return int32Slice(BytesToUint32(b, order))
}

// ==============================
// 6. float32 批量互转
// ==============================
func Float32ToBytes(v []float32, order int) []byte {
	ui32 := make([]uint32, len(v))
	for i, f := range v {
		ui32[i] = math.Float32bits(f)
	}
	return Uint32ToBytes(ui32, order)
}

func BytesToFloat32(b []byte, order int) []float32 {
	ui32 := BytesToUint32(b, order)
	out := make([]float32, len(ui32))
	for i, u := range ui32 {
		out[i] = math.Float32frombits(u)
	}
	return out
}

// ==============================
// 7. uint64 批量互转
// ==============================
func Uint64ToBytes(v []uint64, order int) []byte {
	out := make([]byte, len(v)*8)
	for i, n := range v {
		switch order {
		case ABCDEFGH:
			out[i*8] = byte(n >> 56)
			out[i*8+1] = byte(n >> 48)
			out[i*8+2] = byte(n >> 40)
			out[i*8+3] = byte(n >> 32)
			out[i*8+4] = byte(n >> 24)
			out[i*8+5] = byte(n >> 16)
			out[i*8+6] = byte(n >> 8)
			out[i*8+7] = byte(n)
		case BADCFEHG:
			out[i*8] = byte(n >> 8)
			out[i*8+1] = byte(n >> 56)
			out[i*8+2] = byte(n >> 48)
			out[i*8+3] = byte(n >> 40)
			out[i*8+4] = byte(n >> 24)
			out[i*8+5] = byte(n >> 32)
			out[i*8+6] = byte(n >> 16)
			out[i*8+7] = byte(n)
		case DCBAHGFE:
			out[i*8] = byte(n)
			out[i*8+1] = byte(n >> 8)
			out[i*8+2] = byte(n >> 16)
			out[i*8+3] = byte(n >> 24)
			out[i*8+4] = byte(n >> 32)
			out[i*8+5] = byte(n >> 40)
			out[i*8+6] = byte(n >> 48)
			out[i*8+7] = byte(n >> 56)
		}
	}
	return out
}

func BytesToUint64(b []byte, order int) []uint64 {
	size := len(b) / 8
	out := make([]uint64, size)
	for i := 0; i < size; i++ {
		switch order {
		case ABCDEFGH:
			out[i] = uint64(b[i*8])<<56 | uint64(b[i*8+1])<<48 | uint64(b[i*8+2])<<40 | uint64(b[i*8+3])<<32 |
				uint64(b[i*8+4])<<24 | uint64(b[i*8+5])<<16 | uint64(b[i*8+6])<<8 | uint64(b[i*8+7])
		case BADCFEHG:
			out[i] = uint64(b[i*8+1])<<56 | uint64(b[i*8+2])<<48 | uint64(b[i*8+3])<<40 | uint64(b[i*8])<<32 |
				uint64(b[i*8+5])<<24 | uint64(b[i*8+6])<<16 | uint64(b[i*8+4])<<8 | uint64(b[i*8+7])
		case DCBAHGFE:
			out[i] = uint64(b[i*8+7])<<56 | uint64(b[i*8+6])<<48 | uint64(b[i*8+5])<<40 | uint64(b[i*8+4])<<32 |
				uint64(b[i*8+3])<<24 | uint64(b[i*8+2])<<16 | uint64(b[i*8+1])<<8 | uint64(b[i*8])
		}
	}
	return out
}

// ==============================
// 8. int64 批量互转
// ==============================
func Int64ToBytes(v []int64, order int) []byte {
	return Uint64ToBytes(uint64Slice(v), order)
}

func BytesToInt64(b []byte, order int) []int64 {
	return int64Slice(BytesToUint64(b, order))
}

// ==============================
// 9. float64 批量互转
// ==============================
func Float64ToBytes(v []float64, order int) []byte {
	ui64 := make([]uint64, len(v))
	for i, f := range v {
		ui64[i] = math.Float64bits(f)
	}
	return Uint64ToBytes(ui64, order)
}

func BytesToFloat64(b []byte, order int) []float64 {
	ui64 := BytesToUint64(b, order)
	out := make([]float64, len(ui64))
	for i, u := range ui64 {
		out[i] = math.Float64frombits(u)
	}
	return out
}

// ==============================
// 辅助工具函数
// ==============================
func uint32Slice(i []int32) []uint32 {
	o := make([]uint32, len(i))
	for k, v := range i {
		o[k] = uint32(v)
	}
	return o
}

func int32Slice(u []uint32) []int32 {
	o := make([]int32, len(u))
	for k, v := range u {
		o[k] = int32(v)
	}
	return o
}

func uint64Slice(i []int64) []uint64 {
	o := make([]uint64, len(i))
	for k, v := range i {
		o[k] = uint64(v)
	}
	return o
}

func int64Slice(u []uint64) []int64 {
	o := make([]int64, len(u))
	for k, v := range u {
		o[k] = int64(v)
	}
	return o
}

// GetItemOrDefault 获取数组第 index 个元素
// 越界  返回类型零值
// 不越界 返回真实值
func Get_list_index[T any](arr []T, start int, cnt int) []T {
	// 如果数量 <=0 → 直接返回空数组
	if cnt <= 0 {
		return []T{}
	}

	var result []T
	for i := 0; i < cnt; i++ {
		index := start + i
		if index < 0 || index >= len(arr) {
			var zero T
			result = append(result, zero)
			continue
		}
		result = append(result, arr[index])
	}

	return result
}

// Update_List_Slice
// 泛型函数：从 startIndex 开始，用 values 覆盖 arr 指向的数组
// 规则：
// 1. startIndex 越界 → 不做任何修改
// 2. values 长度超出剩余空间 → 超出部分忽略
// 3. 直接修改原数组（传指针）
func Update_List_Slice[T any](arr *[]T, startIndex int, values []T) {
	// 空指针保护
	if arr == nil {
		return
	}

	arrLen := len(*arr)

	// 开始下标非法（小于0 或 超过数组最大下标+1），直接不修改
	// 允许 startIndex == arrLen（表示从末尾追加）
	if startIndex < 0 || startIndex > arrLen {
		return
	}

	// 需要的总长度 = 开始下标 + 要写入的值数量
	needLen := startIndex + len(values)

	// 如果需要更长 → 自动扩容
	if needLen > arrLen {
		// 扩容到需要的长度
		newArr := make([]T, needLen)
		// 复制原有数据
		copy(newArr, *arr)
		*arr = newArr
	}

	// 开始覆盖赋值
	for i := range values {
		(*arr)[startIndex+i] = values[i]
	}
}

// Is_Type_Match
// 传入：
//
//	value any    要判断的值
//	typeName string  类型字符串："bool"、"uint"、"uint8"、"uint16"、"uint32"、"uint64"、"int"、"int8"、"int32"、"int64"、"float32"、"float64"、"string"
//
// 返回：是否完全匹配
func Is_Type_Match(value any, typeName string) bool {
	if value == nil {
		return false
	}

	// 获取值的真实类型名称
	return reflect.TypeOf(value).Name() == typeName
}
