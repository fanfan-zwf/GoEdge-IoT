package db_point

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// 根据Value_Type转换Value为对应类型的核心函数
// 返回转换后的值和错误（转换失败时）
func ConvertValueByType(rawValue any, valueType string) (any, error) {
	// 专门处理 nil 值：返回 0.0，无错误

	if rawValue == nil && valueType == "bool" {
		return false, nil
	}

	if rawValue == nil && valueType == "float" {
		return 0.0, nil
	}

	if rawValue == nil && (valueType == "int8" ||
		valueType == "uint8" ||
		valueType == "int16" ||
		valueType == "uint16" ||
		valueType == "int32" ||
		valueType == "uint32" ||
		valueType == "int64" ||
		valueType == "uint64" ||
		valueType == "int" ||
		valueType == "uint") {
		return 0, nil
	}

	switch valueType {
	// 处理整数类型
	case "int":
		switch v := rawValue.(type) {
		case float64: // JSON数字反序列化默认是float64
			if v == math.Trunc(v) { // 检查是否是整数（如25.0，而非36.5）
				return int(v), nil
			}
			return nil, fmt.Errorf("值 %.2f 不是整数，无法转换为int类型", v)
		case string: // 兼容JSON中值是字符串形式的数字（如"100"）
			intVal, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("字符串 %s 转换int失败: %v", v, err)
			}
			return intVal, nil
		case int: // 极端情况：原始值已是int
			return v, nil
		default:
			return nil, fmt.Errorf("类型 %T 无法转换为int", rawValue)
		}

	// 处理浮点数类型
	case "float":
		switch v := rawValue.(type) {
		case float64: // JSON浮点数/整数反序列化后都是float64
			return v, nil
		case int:
			return float64(v), nil
		case string: // 兼容字符串形式的浮点数（如"36.5"）
			floatVal, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("字符串 %s 转换float失败: %v 希望类型:%s", v, err, valueType)
			}
			return floatVal, nil
		default:
			return nil, fmt.Errorf("类型 %T 无法转换为float,希望类型:%s", rawValue, valueType)
		}

	// 处理字符串类型
	case "string":
		// 无论原始类型是什么，都转为字符串（如数字25→"25"，bool→"true"）
		return fmt.Sprintf("%v", rawValue), nil

	// 处理布尔类型
	case "bool":
		switch v := rawValue.(type) {
		case bool:
			return v, nil
		case string: // 兼容字符串形式的bool（如"true"/"false"）
			boolVal, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("字符串 %s 转换bool失败: %v", v, err)
			}
			return boolVal, nil
		default:
			return nil, fmt.Errorf("类型 %T 无法转换为bool", rawValue)
		}

	// 未定义的类型
	default:
		return nil, fmt.Errorf("不支持的Value_Type: %s", valueType)
	}
}

// json数组转化更新数组结构体
func Update_Json_Type_List(jsonData string) (Change_Value_List []Db_Value_type, err error) {

	// 2. 结构体数组转JSON（序列化）
	// jsonData, err := json.MarshalIndent(originalData, "", "  ") // 带缩进，便于查看
	// if err != nil {
	// 	log.Fatalf("序列化失败: %v", err)
	// }
	// fmt.Println("=== 序列化后的JSON ===")
	// fmt.Println(string(jsonData))

	// 3. JSON转结构体数组（反序列化）

	err = json.Unmarshal([]byte(jsonData), &Change_Value_List)
	if err != nil {
		err = fmt.Errorf("反序列化失败: %v", err)
		return
	}
	for idx, point := range Change_Value_List {

		var (
			Change_Value any
		)
		Change_Value, err = ConvertValueByType(point.Value, point.Type)
		if err != nil {
			Change_Value_List[idx].Msg = fmt.Sprintf("缓存转换失败：%v\n", err)
		} else {
			Change_Value_List[idx].Value = Change_Value
		}

	}
	return
}

// json转化更新结构体
func Update_Json_Type(jsonData string) (Db_Value Db_Value_type, err error) {

	// 2. 结构体数组转JSON（序列化）
	// jsonData, err := json.MarshalIndent(originalData, "", "  ") // 带缩进，便于查看
	// if err != nil {
	// 	log.Fatalf("序列化失败: %v", err)
	// }
	// fmt.Println("=== 序列化后的JSON ===")
	// fmt.Println(string(jsonData))

	// 3. JSON转结构体数组（反序列化）

	err = json.Unmarshal([]byte(jsonData), &Db_Value)
	if err != nil {
		err = fmt.Errorf("反序列化失败: %v", err)
		return
	}

	converted_Value, err := ConvertValueByType(Db_Value.Value, Db_Value.Type)
	if err != nil {
		Db_Value.Msg = fmt.Sprintf("缓存转换失败：%v\n", err)
	} else {
		Db_Value.Value = converted_Value
	}
	return
}

// json数组转化更新数组结构体
func Change_Json_Type_List(jsonData string) (Change_Value_List []Update_Value_type, err error) {

	// 2. 结构体数组转JSON（序列化）
	// jsonData, err := json.MarshalIndent(originalData, "", "  ") // 带缩进，便于查看
	// if err != nil {
	// 	log.Fatalf("序列化失败: %v", err)
	// }
	// fmt.Println("=== 序列化后的JSON ===")
	// fmt.Println(string(jsonData))

	// 3. JSON转结构体数组（反序列化）
	var parsedData []Update_Value_type
	err = json.Unmarshal([]byte(jsonData), &parsedData)
	if err != nil {
		err = fmt.Errorf("反序列化失败: %v", err)
		return
	}
	for idx, point := range parsedData {

		var (
			Change_Value_List    any
			converted_Value_Last any
		)
		Change_Value_List, err = ConvertValueByType(point.Value, point.Type)
		if err != nil {
			parsedData[idx].Msg = fmt.Sprintf("缓存转换失败：%v\n", err)
		} else {
			parsedData[idx].Value = Change_Value_List
		}

		converted_Value_Last, err = ConvertValueByType(point.Last_Value, point.Type)
		if err != nil {
			parsedData[idx].Msg = fmt.Sprintf("缓存转换失败：%v\n", err)
		} else {
			parsedData[idx].Last_Value = converted_Value_Last
		}
	}
	return
}

// json转化更新结构体
func Change_Json_Type(jsonData string) (Change_Value Update_Value_type, err error) {

	// 2. 结构体数组转JSON（序列化）
	// jsonData, err := json.MarshalIndent(originalData, "", "  ") // 带缩进，便于查看
	// if err != nil {
	// 	log.Fatalf("序列化失败: %v", err)
	// }
	// fmt.Println("=== 序列化后的JSON ===")
	// fmt.Println(string(jsonData))

	// 3. JSON转结构体数组（反序列化）

	err = json.Unmarshal([]byte(jsonData), &Change_Value)
	if err != nil {
		err = fmt.Errorf("反序列化失败: %v", err)
		return
	}

	converted_Value, err := ConvertValueByType(Change_Value.Value, Change_Value.Type)
	if err != nil {
		Change_Value.Msg = fmt.Sprintf("缓存转换失败：%v\n", err)
	} else {
		Change_Value.Value = converted_Value
	}

	converted_Value_Last, err := ConvertValueByType(Change_Value.Last_Value, Change_Value.Type)
	if err != nil {
		Change_Value.Msg = fmt.Sprintf("缓存转换失败：%v\n", err)
	} else {
		Change_Value.Last_Value = converted_Value_Last
	}
	return
}

// ParseDevicePointV1 解析指定格式的设备点位字符串
// 合法格式：//设备ID//任意路径（仅禁止路径中出现//，/数量无限制）
// 示例：//hezi_1//测试modbus_tcp/点位1 、//hezi_1//a/b/c/d/点位2
func ParseDevicePointV1(s string) (deviceID string, point string, err error) {
	// 1. 校验开头必须是//
	if !strings.HasPrefix(s, "//") {
		return "", "", errors.New("字符串开头必须以//开头")
	}

	// 2. 去掉开头的//，拆分设备ID和点位前缀（按//分割且仅分割1次）
	afterPrefix := s[2:]
	parts := strings.SplitN(afterPrefix, "//", 2)

	// 校验设备ID后是否有且仅有1个//
	if len(parts) != 2 {
		return "", "", errors.New("设备ID后必须有且仅有1个//分隔（格式：//设备ID//点位路径）")
	}

	// 3. 提取并校验设备ID（不能为空）
	deviceID = parts[0]
	if deviceID == "" {
		return "", "", errors.New("设备ID不能为空")
	}

	// 4. 提取点位部分并校验：不能包含//（允许任意数量/）
	pointRaw := parts[1]
	if strings.Contains(pointRaw, "//") {
		return "", "", errors.New("点位路径中禁止出现//，仅允许使用单个/分隔")
	}

	// 5. 拼接最终的点位格式（以/开头）
	point = "/" + pointRaw

	return deviceID, point, nil
}
