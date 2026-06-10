package byte_util

import (
	"fmt"
	"strconv"
)

func ConvBool(v any, to string) (bool, bool) {
	switch to {
	case "bool":
		b, ok := v.(bool)
		return b, ok
	case "uint8", "byte":
		b, ok := v.(uint8)
		return b != 0, ok
	case "int8":
		b, ok := v.(int8)
		return b != 0, ok
	case "uint16":
		b, ok := v.(uint16)
		return b != 0, ok
	case "int16":
		b, ok := v.(int16)
		return b != 0, ok
	case "uint32":
		b, ok := v.(uint32)
		return b != 0, ok
	case "int32":
		b, ok := v.(int32)
		return b != 0, ok
	case "uint64":
		b, ok := v.(uint64)
		return b != 0, ok
	case "int64":
		b, ok := v.(int64)
		return b != 0, ok
	case "uint":
		b, ok := v.(uint)
		return b != 0, ok
	case "int":
		b, ok := v.(int)
		return b != 0, ok
	case "float32":
		b, ok := v.(float32)
		return b != 0, ok
	case "float64", "float":
		b, ok := v.(float64)
		return b != 0, ok
	case "string":
		s, ok := v.(string)
		if !ok {
			return false, false
		}
		return s == "true" || s == "1", true
	default:
		return false, false
	}
}

func ConvUint8(v any, to string) (uint8, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return val, ok
	case "int8":
		val, ok := v.(int8)
		return uint8(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return uint8(val), ok
	case "int16":
		val, ok := v.(int16)
		return uint8(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return uint8(val), ok
	case "int32":
		val, ok := v.(int32)
		return uint8(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return uint8(val), ok
	case "int64":
		val, ok := v.(int64)
		return uint8(val), ok
	case "uint":
		val, ok := v.(uint)
		return uint8(val), ok
	case "int":
		val, ok := v.(int)
		return uint8(val), ok
	case "float32":
		val, ok := v.(float32)
		return uint8(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return uint8(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseUint(str, 10, 8)
		if err != nil {
			return 0, false
		}
		return uint8(num), true
	default:
		return 0, false
	}
}

func ConvInt8(v any, to string) (int8, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return int8(val), ok
	case "int8":
		val, ok := v.(int8)
		return val, ok
	case "uint16":
		val, ok := v.(uint16)
		return int8(val), ok
	case "int16":
		val, ok := v.(int16)
		return int8(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return int8(val), ok
	case "int32":
		val, ok := v.(int32)
		return int8(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return int8(val), ok
	case "int64":
		val, ok := v.(int64)
		return int8(val), ok
	case "uint":
		val, ok := v.(uint)
		return int8(val), ok
	case "int":
		val, ok := v.(int)
		return int8(val), ok
	case "float32":
		val, ok := v.(float32)
		return int8(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return int8(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseInt(str, 10, 8)
		if err != nil {
			return 0, false
		}
		return int8(num), true
	default:
		return 0, false
	}
}

func ConvUint16(v any, to string) (uint16, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return uint16(val), ok
	case "int8":
		val, ok := v.(int8)
		return uint16(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return val, ok
	case "int16":
		val, ok := v.(int16)
		return uint16(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return uint16(val), ok
	case "int32":
		val, ok := v.(int32)
		return uint16(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return uint16(val), ok
	case "int64":
		val, ok := v.(int64)
		return uint16(val), ok
	case "uint":
		val, ok := v.(uint)
		return uint16(val), ok
	case "int":
		val, ok := v.(int)
		return uint16(val), ok
	case "float32":
		val, ok := v.(float32)
		return uint16(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return uint16(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseUint(str, 10, 16)
		if err != nil {
			return 0, false
		}
		return uint16(num), true
	default:
		return 0, false
	}
}

func ConvInt16(v any, to string) (int16, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return int16(val), ok
	case "int8":
		val, ok := v.(int8)
		return int16(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return int16(val), ok
	case "int16":
		val, ok := v.(int16)
		return val, ok
	case "uint32":
		val, ok := v.(uint32)
		return int16(val), ok
	case "int32":
		val, ok := v.(int32)
		return int16(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return int16(val), ok
	case "int64":
		val, ok := v.(int64)
		return int16(val), ok
	case "uint":
		val, ok := v.(uint)
		return int16(val), ok
	case "int":
		val, ok := v.(int)
		return int16(val), ok
	case "float32":
		val, ok := v.(float32)
		return int16(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return int16(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseInt(str, 10, 16)
		if err != nil {
			return 0, false
		}
		return int16(num), true
	default:
		return 0, false
	}
}

func ConvUint32(v any, to string) (uint32, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return uint32(val), ok
	case "int8":
		val, ok := v.(int8)
		return uint32(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return uint32(val), ok
	case "int16":
		val, ok := v.(int16)
		return uint32(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return val, ok
	case "int32":
		val, ok := v.(int32)
		return uint32(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return uint32(val), ok
	case "int64":
		val, ok := v.(int64)
		return uint32(val), ok
	case "uint":
		val, ok := v.(uint)
		return uint32(val), ok
	case "int":
		val, ok := v.(int)
		return uint32(val), ok
	case "float32":
		val, ok := v.(float32)
		return uint32(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return uint32(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseUint(str, 10, 32)
		if err != nil {
			return 0, false
		}
		return uint32(num), true
	default:
		return 0, false
	}
}

func ConvInt32(v any, to string) (int32, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return int32(val), ok
	case "int8":
		val, ok := v.(int8)
		return int32(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return int32(val), ok
	case "int16":
		val, ok := v.(int16)
		return int32(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return int32(val), ok
	case "int32":
		val, ok := v.(int32)
		return val, ok
	case "uint64":
		val, ok := v.(uint64)
		return int32(val), ok
	case "int64":
		val, ok := v.(int64)
		return int32(val), ok
	case "uint":
		val, ok := v.(uint)
		return int32(val), ok
	case "int":
		val, ok := v.(int)
		return int32(val), ok
	case "float32":
		val, ok := v.(float32)
		return int32(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return int32(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseInt(str, 10, 32)
		if err != nil {
			return 0, false
		}
		return int32(num), true
	default:
		return 0, false
	}
}

func ConvUint64(v any, to string) (uint64, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return uint64(val), ok
	case "int8":
		val, ok := v.(int8)
		return uint64(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return uint64(val), ok
	case "int16":
		val, ok := v.(int16)
		return uint64(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return uint64(val), ok
	case "int32":
		val, ok := v.(int32)
		return uint64(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return val, ok
	case "int64":
		val, ok := v.(int64)
		return uint64(val), ok
	case "uint":
		val, ok := v.(uint)
		return uint64(val), ok
	case "int":
		val, ok := v.(int)
		return uint64(val), ok
	case "float32":
		val, ok := v.(float32)
		return uint64(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return uint64(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return 0, false
		}
		return uint64(num), true
	default:
		return 0, false
	}
}

func ConvInt64(v any, to string) (int64, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return int64(val), ok
	case "int8":
		val, ok := v.(int8)
		return int64(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return int64(val), ok
	case "int16":
		val, ok := v.(int16)
		return int64(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return int64(val), ok
	case "int32":
		val, ok := v.(int32)
		return int64(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return int64(val), ok
	case "int64":
		val, ok := v.(int64)
		return val, ok
	case "uint":
		val, ok := v.(uint)
		return int64(val), ok
	case "int":
		val, ok := v.(int)
		return int64(val), ok
	case "float32":
		val, ok := v.(float32)
		return int64(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return int64(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return 0, false
		}
		return int64(num), true
	default:
		return 0, false
	}
}

func ConvUint(v any, to string) (uint, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return uint(val), ok
	case "int8":
		val, ok := v.(int8)
		return uint(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return uint(val), ok
	case "int16":
		val, ok := v.(int16)
		return uint(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return uint(val), ok
	case "int32":
		val, ok := v.(int32)
		return uint(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return uint(val), ok
	case "int64":
		val, ok := v.(int64)
		return uint(val), ok
	case "uint":
		val, ok := v.(uint)
		return val, ok
	case "int":
		val, ok := v.(int)
		return uint(val), ok
	case "float32":
		val, ok := v.(float32)
		return uint(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return uint(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return 0, false
		}
		return uint(num), true
	default:
		return 0, false
	}
}

func ConvInt(v any, to string) (int, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1, true
		}
		return 0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return int(val), ok
	case "int8":
		val, ok := v.(int8)
		return int(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return int(val), ok
	case "int16":
		val, ok := v.(int16)
		return int(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return int(val), ok
	case "int32":
		val, ok := v.(int32)
		return int(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return int(val), ok
	case "int64":
		val, ok := v.(int64)
		return int(val), ok
	case "uint":
		val, ok := v.(uint)
		return int(val), ok
	case "int":
		val, ok := v.(int)
		return val, ok
	case "float32":
		val, ok := v.(float32)
		return int(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return int(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return 0, false
		}
		return int(num), true
	default:
		return 0, false
	}
}

func ConvFloat32(v any, to string) (float32, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1.0, true
		}
		return 0.0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return float32(val), ok
	case "int8":
		val, ok := v.(int8)
		return float32(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return float32(val), ok
	case "int16":
		val, ok := v.(int16)
		return float32(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return float32(val), ok
	case "int32":
		val, ok := v.(int32)
		return float32(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return float32(val), ok
	case "int64":
		val, ok := v.(int64)
		return float32(val), ok
	case "uint":
		val, ok := v.(uint)
		return float32(val), ok
	case "int":
		val, ok := v.(int)
		return float32(val), ok
	case "float32":
		val, ok := v.(float32)
		return val, ok
	case "float64", "float":
		val, ok := v.(float64)
		return float32(val), ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseFloat(str, 32)
		if err != nil {
			return 0, false
		}
		return float32(num), true
	default:
		return 0, false
	}
}

func ConvFloat64(v any, to string) (float64, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return 0, false
		}
		if val {
			return 1.0, true
		}
		return 0.0, true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return float64(val), ok
	case "int8":
		val, ok := v.(int8)
		return float64(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return float64(val), ok
	case "int16":
		val, ok := v.(int16)
		return float64(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return float64(val), ok
	case "int32":
		val, ok := v.(int32)
		return float64(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return float64(val), ok
	case "int64":
		val, ok := v.(int64)
		return float64(val), ok
	case "uint":
		val, ok := v.(uint)
		return float64(val), ok
	case "int":
		val, ok := v.(int)
		return float64(val), ok
	case "float32":
		val, ok := v.(float32)
		return float64(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return val, ok
	case "string":
		str, ok := v.(string)
		if !ok {
			return 0, false
		}
		num, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return 0, false
		}
		return num, true
	default:
		return 0, false
	}
}

func ConvString(v any, to string) (string, bool) {
	switch to {
	case "bool":
		val, ok := v.(bool)
		if !ok {
			return "", false
		}
		if val {
			return "true", true
		}
		return "false", true
	case "uint8", "byte":
		val, ok := v.(uint8)
		return fmt.Sprint(val), ok
	case "int8":
		val, ok := v.(int8)
		return fmt.Sprint(val), ok
	case "uint16":
		val, ok := v.(uint16)
		return fmt.Sprint(val), ok
	case "int16":
		val, ok := v.(int16)
		return fmt.Sprint(val), ok
	case "uint32":
		val, ok := v.(uint32)
		return fmt.Sprint(val), ok
	case "int32":
		val, ok := v.(int32)
		return fmt.Sprint(val), ok
	case "uint64":
		val, ok := v.(uint64)
		return fmt.Sprint(val), ok
	case "int64":
		val, ok := v.(int64)
		return fmt.Sprint(val), ok
	case "uint":
		val, ok := v.(uint)
		return fmt.Sprint(val), ok
	case "int":
		val, ok := v.(int)
		return fmt.Sprint(val), ok
	case "float32":
		val, ok := v.(float32)
		return fmt.Sprint(val), ok
	case "float64", "float":
		val, ok := v.(float64)
		return fmt.Sprint(val), ok
	case "string":
		str, ok := v.(string)
		return str, ok
	default:
		return "", false
	}
}

func ConvertType(value any, fromType, toType string) (any, bool) {
	switch toType {
	case "bool":
		return ConvBool(value, fromType)
	case "uint8", "byte":
		return ConvUint8(value, fromType)
	case "int8":
		return ConvInt8(value, fromType)
	case "uint16":
		return ConvUint16(value, fromType)
	case "int16":
		return ConvInt16(value, fromType)
	case "uint32":
		return ConvUint32(value, fromType)
	case "int32":
		return ConvInt32(value, fromType)
	case "uint64":
		return ConvUint64(value, fromType)
	case "int64":
		return ConvInt64(value, fromType)
	case "uint":
		return ConvUint(value, fromType)
	case "int":
		return ConvInt(value, fromType)
	case "float32":
		return ConvFloat32(value, fromType)
	case "float64", "float":
		return ConvFloat64(value, fromType)
	case "string":
		return ConvString(value, fromType)
	default:
		return nil, false
	}
}
