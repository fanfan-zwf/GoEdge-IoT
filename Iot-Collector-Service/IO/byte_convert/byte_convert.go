/*
* 日期: 2026.4.16 AM11:38
* 作者: 范范zwf
* 作用: 字节转换
 */

package byte_convert

import (
	"math"
)

// ==============================
// 字节序（完整）
// ==============================
const (
	// 2字节
	AB = iota
	BA

	// 4字节
	ABCD
	ABDC
	BACD
	DCBA

	// 8字节（8种完整顺序）
	ABCDEFGH
	ABCDEHGF
	ABFEGHCD
	ABGHFEDC
	BACDFEGH
	BADCFEHG
	HGFEDCBA
	HGFECDAB
)

// ==============================
// bool → 所有
// ==============================
func Convert_bool_byte(in []bool) []byte { return Convert_bool_uint8(in) }
func Convert_bool_uint8(in []bool) []uint8 {
	out := make([]uint8, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_int8(in []bool) []int8 {
	out := make([]int8, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_uint16(in []bool) []uint16 {
	out := make([]uint16, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_int16(in []bool) []int16 {
	out := make([]int16, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_uint32(in []bool) []uint32 {
	out := make([]uint32, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_int32(in []bool) []int32 {
	out := make([]int32, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_uint64(in []bool) []uint64 {
	out := make([]uint64, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_int64(in []bool) []int64 {
	out := make([]int64, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_float32(in []bool) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}
func Convert_bool_float64(in []bool) []float64 {
	out := make([]float64, len(in))
	for i, v := range in {
		if v {
			out[i] = 1
		}
	}
	return out
}

// ==============================
// uint8 → 所有
// ==============================
func Convert_uint8_bool(in []uint8) []bool {
	out := make([]bool, len(in))
	for i, v := range in {
		out[i] = v != 0
	}
	return out
}
func Convert_uint8_int8(in []uint8) []int8 {
	out := make([]int8, len(in))
	for i := range in {
		out[i] = int8(in[i])
	}
	return out
}
func Convert_uint8_uint16(in []uint8, order int) []uint16 {
	n := len(in) / 2
	out := make([]uint16, n)
	for i := 0; i < n; i++ {
		b := in[i*2 : i*2+2]
		switch order {
		case AB:
			out[i] = uint16(b[0]) | uint16(b[1])<<8
		case BA:
			out[i] = uint16(b[1]) | uint16(b[0])<<8
		}
	}
	return out
}
func Convert_uint8_int16(in []uint8, order int) []int16 {
	tmp := Convert_uint8_uint16(in, order)
	out := make([]int16, len(tmp))
	for i := range tmp {
		out[i] = int16(tmp[i])
	}
	return out
}
func Convert_uint8_uint32(in []uint8, order int) []uint32 {
	n := len(in) / 4
	out := make([]uint32, n)
	for i := 0; i < n; i++ {
		b := in[i*4 : i*4+4]
		switch order {
		case ABCD:
			out[i] = uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
		case ABDC:
			out[i] = uint32(b[0]) | uint32(b[1])<<8 | uint32(b[3])<<16 | uint32(b[2])<<24
		case BACD:
			out[i] = uint32(b[1]) | uint32(b[0])<<8 | uint32(b[3])<<16 | uint32(b[2])<<24
		case DCBA:
			out[i] = uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
		}
	}
	return out
}
func Convert_uint8_int32(in []uint8, order int) []int32 {
	tmp := Convert_uint8_uint32(in, order)
	out := make([]int32, len(tmp))
	for i := range tmp {
		out[i] = int32(tmp[i])
	}
	return out
}
func Convert_uint8_uint64(in []uint8, order int) []uint64 {
	n := len(in) / 8
	out := make([]uint64, n)
	for i := 0; i < n; i++ {
		b := in[i*8 : i*8+8]
		switch order {
		case ABCD:
			out[i] = uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
				uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
		default:
			out[i] = uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
				uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
		}
	}
	return out
}
func Convert_uint8_int64(in []uint8, order int) []int64 {
	tmp := Convert_uint8_uint64(in, order)
	out := make([]int64, len(tmp))
	for i := range tmp {
		out[i] = int64(tmp[i])
	}
	return out
}
func Convert_uint8_float32(in []uint8, order int) []float32 {
	u := Convert_uint8_uint32(in, order)
	out := make([]float32, len(u))
	for i := range u {
		out[i] = math.Float32frombits(u[i])
	}
	return out
}
func Convert_uint8_float64(in []uint8, order int) []float64 {
	u := Convert_uint8_uint64(in, order)
	out := make([]float64, len(u))
	for i := range u {
		out[i] = math.Float64frombits(u[i])
	}
	return out
}

// ==============================
// int8 → 所有
// ==============================
func Convert_int8_bool(in []int8) []bool {
	out := make([]bool, len(in))
	for i, v := range in {
		out[i] = v != 0
	}
	return out
}
func Convert_int8_uint8(in []int8) []uint8 {
	out := make([]uint8, len(in))
	for i := range in {
		out[i] = uint8(in[i])
	}
	return out
}
func Convert_int8_uint16(in []int8, order int) []uint16 {
	return Convert_uint8_uint16(Convert_int8_uint8(in), order)
}
func Convert_int8_int16(in []int8, order int) []int16 {
	return Convert_uint8_int16(Convert_int8_uint8(in), order)
}
func Convert_int8_uint32(in []int8, order int) []uint32 {
	return Convert_uint8_uint32(Convert_int8_uint8(in), order)
}
func Convert_int8_int32(in []int8, order int) []int32 {
	return Convert_uint8_int32(Convert_int8_uint8(in), order)
}
func Convert_int8_uint64(in []int8, order int) []uint64 {
	return Convert_uint8_uint64(Convert_int8_uint8(in), order)
}
func Convert_int8_int64(in []int8, order int) []int64 {
	return Convert_uint8_int64(Convert_int8_uint8(in), order)
}

// ==============================
// uint16 → 所有
// ==============================
func Convert_uint16_uint8(in []uint16, order int) []uint8 {
	out := make([]uint8, len(in)*2)
	for i := range in {
		v := in[i]
		switch order {
		case AB:
			out[i*2] = uint8(v)
			out[i*2+1] = uint8(v >> 8)
		case BA:
			out[i*2] = uint8(v >> 8)
			out[i*2+1] = uint8(v)
		}
	}
	return out
}
func Convert_uint16_int8(in []uint16, order int) []int8 {
	tmp := Convert_uint16_uint8(in, order)
	out := make([]int8, len(tmp))
	for i := range tmp {
		out[i] = int8(tmp[i])
	}
	return out
}

// ==============================
// int16 → 所有
// ==============================
func Convert_int16_uint8(in []int16, order int) []uint8 {
	tmp := make([]uint16, len(in))
	for i := range in {
		tmp[i] = uint16(in[i])
	}
	return Convert_uint16_uint8(tmp, order)
}
func Convert_int16_int8(in []int16, order int) []int8 {
	tmp := Convert_int16_uint8(in, order)
	out := make([]int8, len(tmp))
	for i := range tmp {
		out[i] = int8(tmp[i])
	}
	return out
}

// ==============================
// uint32 → 所有
// ==============================
func Convert_uint32_uint8(in []uint32, order int) []uint8 {
	out := make([]uint8, len(in)*4)
	for i := range in {
		v := in[i]
		switch order {
		case ABCD:
			out[i*4] = uint8(v)
			out[i*4+1] = uint8(v >> 8)
			out[i*4+2] = uint8(v >> 16)
			out[i*4+3] = uint8(v >> 24)
		case ABDC:
			out[i*4] = uint8(v)
			out[i*4+1] = uint8(v >> 8)
			out[i*4+2] = uint8(v >> 24)
			out[i*4+3] = uint8(v >> 16)
		case BACD:
			out[i*4] = uint8(v >> 8)
			out[i*4+1] = uint8(v)
			out[i*4+2] = uint8(v >> 24)
			out[i*4+3] = uint8(v >> 16)
		case DCBA:
			out[i*4] = uint8(v >> 24)
			out[i*4+1] = uint8(v >> 16)
			out[i*4+2] = uint8(v >> 8)
			out[i*4+3] = uint8(v)
		}
	}
	return out
}

// ==============================
// int32 → 所有
// ==============================
func Convert_int32_uint8(in []int32, order int) []uint8 {
	tmp := make([]uint32, len(in))
	for i := range in {
		tmp[i] = uint32(in[i])
	}
	return Convert_uint32_uint8(tmp, order)
}

// ==============================
// uint64 → 所有
// ==============================
func Convert_uint64_uint8(in []uint64, order int) []uint8 {
	out := make([]uint8, len(in)*8)
	for i := range in {
		v := in[i]
		switch order {
		case ABCD:
			out[i*8] = uint8(v)
			out[i*8+1] = uint8(v >> 8)
			out[i*8+2] = uint8(v >> 16)
			out[i*8+3] = uint8(v >> 24)
			out[i*8+4] = uint8(v >> 32)
			out[i*8+5] = uint8(v >> 40)
			out[i*8+6] = uint8(v >> 48)
			out[i*8+7] = uint8(v >> 56)
		default:
			out[i*8] = uint8(v >> 56)
			out[i*8+1] = uint8(v >> 48)
			out[i*8+2] = uint8(v >> 40)
			out[i*8+3] = uint8(v >> 32)
			out[i*8+4] = uint8(v >> 24)
			out[i*8+5] = uint8(v >> 16)
			out[i*8+6] = uint8(v >> 8)
			out[i*8+7] = uint8(v)
		}
	}
	return out
}

// ==============================
// int64 → 所有
// ==============================
func Convert_int64_uint8(in []int64, order int) []uint8 {
	tmp := make([]uint64, len(in))
	for i := range in {
		tmp[i] = uint64(in[i])
	}
	return Convert_uint64_uint8(tmp, order)
}

// ==============================
// float32 → uint8
// ==============================
func Convert_float32_uint8(in []float32, order int) []uint8 {
	u := make([]uint32, len(in))
	for i := range in {
		u[i] = math.Float32bits(in[i])
	}
	return Convert_uint32_uint8(u, order)
}

// ==============================
// float64 → uint8
// ==============================
func Convert_float64_uint8(in []float64, order int) []uint8 {
	u := make([]uint64, len(in))
	for i := range in {
		u[i] = math.Float64bits(in[i])
	}
	return Convert_uint64_uint8(u, order)
}
