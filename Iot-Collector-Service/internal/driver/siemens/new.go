package siemens

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
// JSON 格式示例: {"Address":"192.168.1.1","Rack":0,"Slot":2,"Retry_timeout":"3s","Connect_timeout":"10s","Response_timeout":"10s","ConnectionType":2,"Max_reconnect_interval":"30s","Local_TSAP":0,"Remote_TSAP":0}
// 说明: Local_TSAP/Remote_TSAP 为 snap7 原生 uint16 TSAP 覆盖（JSON 中为十进制数字），0=按 ConnectionType+Rack+Slot 自动推导；
// 适用于 CP 343-1 Advanced（6GK7 343-1EX30）等「S7 连接需在 NETPRO 预配置、使用自定义 TSAP」的设备，填上 CP 配置里那条连接的 TSAP（STEP7 显示 0x0102 → 此处填 258）即可匹配握手。
func Drive_Config_Switch(configStr string) (c Config_type, err error) {
	if err = json.Unmarshal([]byte(configStr), &c); err != nil {
		err = fmt.Errorf("Siemens S7配置JSON解析失败: %w", err)
		return
	}

	// IP地址（必填）
	if c.Address == "" {
		err = fmt.Errorf("Siemens S7配置缺少必填字段: Address")
		return
	}

	// 槽位号（可选，默认2）
	if c.Slot <= 0 {
		c.Slot = 2
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

	// 连接类型（可选，默认2）
	if c.ConnectionType == 0 {
		c.ConnectionType = 2
	}

	// 重连最大间隔（可选，默认30s，指数退避封顶）
	if c.Max_reconnect_interval <= 0 {
		c.Max_reconnect_interval = conv.Duration(30 * time.Second)
	}

	return
}

// S7地址解析正则表达式常量
// 匹配格式: DB1.DBX0.0, DB1.DBB2, I0.0, IB2, M10.0, MB10 等
const Regex_S7_Point_Config = `^(?:(DB)(\d+)\.(DB[XBWDL])|([IQM])([XBWDL]?))(\d+)(?:\.(\d+))?$`

var s7AddrRegex = regexp.MustCompile(Regex_S7_Point_Config)

// s7TypeMap TIA Portal 类型名 → 采集类型代码（key 统一小写，查找前 ToLower）
// Byte / Word / DWord / LWord 作为无符号类型的别名一同内建，方便从 TIA 导出地址时直接匹配
var s7TypeMap = map[string]int{
	"bool":   conv.TYPE_BOOL,
	"sint":   conv.TYPE_INT8,
	"usint":  conv.TYPE_UINT8,
	"byte":   conv.TYPE_UINT8,
	"int":    conv.TYPE_INT16,
	"uint":   conv.TYPE_UINT16,
	"word":   conv.TYPE_UINT16,
	"dint":   conv.TYPE_INT32,
	"udint":  conv.TYPE_UINT32,
	"dword":  conv.TYPE_UINT32,
	"lint":   conv.TYPE_INT64,
	"ulint":  conv.TYPE_UINT64,
	"lword":  conv.TYPE_UINT64,
	"real":   conv.TYPE_FLOAT32,
	"lreal":  conv.TYPE_FLOAT64,
	"string": conv.TYPE_STRING,
}

// parseS7Type 将 TIA Portal 类型名称转为采集类型代码（大小写不敏感，与 S7Points_type.Type 注释中映射一一对应）
// 支持的 TIA 名称：Bool / SInt / USInt(Byte) / Int / UInt(Word) / DInt / UDInt(DWord) / LInt / ULInt(LWord)
//
//	/ Real / LReal / String
//
// 不支持：Char / WChar / WString / Time / LTime / Date / TOD / DT（当前驱动未实现字节流解析）
func parseS7Type(name string) (int, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if t, ok := s7TypeMap[key]; ok {
		return t, nil
	}
	return 0, fmt.Errorf("未知的 TIA 类型名 [%s]，支持：Bool / SInt / USInt / Byte / Int / UInt / Word / DInt / UDInt / DWord / LInt / ULInt / LWord / Real / LReal / String", name)
}

// inferS7TypeBySuffix 根据 S7 地址宽度后缀推断默认 TIA 类型（仅当用户未显式写类型时使用）
//
//	X → Bool；B → USInt；W → UInt；D → UDInt；L → ULInt
//
// 无后缀返回 0，由调用方处理（例如回落到 Value_Type）
func inferS7TypeBySuffix(suffix byte) int {
	switch suffix {
	case 'X':
		return conv.TYPE_BOOL
	case 'B':
		return conv.TYPE_UINT8
	case 'W':
		return conv.TYPE_UINT16
	case 'D':
		return conv.TYPE_UINT32
	case 'L':
		return conv.TYPE_UINT64
	default:
		return 0
	}
}

// Point_Config_Switch 解析 S7 点位配置
// 支持两种格式:
//  1. JSON 格式：{"Area":132,"DBNumber":1,"Start":0,"Child_Address":0,"Type":12}
//     Type 为采集类型代码（TIA Portal 声明类型），参见 S7Points_type.Type 注释中的代码值↔TIA类型映射表；
//     缺省或为 0 时，调用方会回落到 Value_Type（保持向后兼容）。
//  2. S7 地址字符串："<地址> [TIA类型名]"（类型名可省略，省略时按地址后缀推断默认类型）
//     例："DB1.DBD0 Real"、"DB1.DBX0.5 Bool"、"M10.0 Byte"、"I0"、"DB1.DBB8 USInt"
//     后缀地址→默认类型对应：X→Bool、B→USInt、W→UInt、D→UDInt、L→ULInt
func Point_Config_Switch(configStr string) (point S7Points_type, err error) {
	configStr = strings.TrimSpace(configStr)
	if configStr == "" {
		return point, fmt.Errorf("配置不能为空")
	}

	// 尝试 JSON 格式解析
	if strings.HasPrefix(configStr, "{") {
		var jsonConfig struct {
			Area          int `json:"Area"`
			DBNumber      int `json:"DBNumber"`
			Start         int `json:"Start"`
			Child_Address int `json:"Child_Address"`
			Type          int `json:"Type"`
		}
		if err := json.Unmarshal([]byte(configStr), &jsonConfig); err != nil {
			return point, fmt.Errorf("JSON 格式解析失败: %w", err)
		}
		point.Area = jsonConfig.Area
		point.DBNumber = jsonConfig.DBNumber
		point.Start = jsonConfig.Start
		point.Child_Address = jsonConfig.Child_Address
		point.Type = jsonConfig.Type // 为 0 时由调用方回落，不在此处报错
		return point, nil
	}

	// S7 地址字符串格式解析：先拆分可选的类型后缀
	addr := configStr
	typeName := ""
	if idx := strings.IndexAny(configStr, " \t"); idx >= 0 {
		addr = strings.TrimSpace(configStr[:idx])
		typeName = strings.TrimSpace(configStr[idx+1:])
	}

	matches := s7AddrRegex.FindStringSubmatch(strings.ToUpper(addr))
	if matches == nil {
		return point, fmt.Errorf("无法解析 S7 地址格式: %s", addr)
	}

	// 判断是 DB 区还是 I/Q/M 区，同时采集宽度后缀（用于默认类型推断）
	var widthSuffix byte
	if matches[1] == "DB" {
		// DB区: matches[1]=DB, matches[2]=块号, matches[3]=DBX/DBB/DBW/DBD/DBL, matches[6]=字节, matches[7]=位
		point.Area = 0x84
		point.DBNumber, _ = strconv.Atoi(matches[2])
		point.Start, _ = strconv.Atoi(matches[6])
		if len(matches[3]) == 4 {
			widthSuffix = matches[3][3]
		}
		if matches[3] == "DBX" && matches[7] != "" {
			point.Child_Address, _ = strconv.Atoi(matches[7])
		}
	} else {
		// I/Q/M区: matches[4]=I/Q/M, matches[5]=X/B/W/D/L, matches[6]=字节, matches[7]=位
		switch matches[4] {
		case "I":
			point.Area = 0x81
		case "Q":
			point.Area = 0x82
		case "M":
			point.Area = 0x83
		}
		point.Start, _ = strconv.Atoi(matches[6])
		if len(matches[5]) == 1 {
			widthSuffix = matches[5][0]
		}
		if matches[5] == "X" || (matches[5] == "" && matches[7] != "") {
			if matches[7] != "" {
				point.Child_Address, _ = strconv.Atoi(matches[7])
			}
		}
	}

	// Type 解析：优先显式类型名（TIA Portal），否则按地址后缀推断；都拿不到时保持 0 让调用方回落
	if typeName != "" {
		t, perr := parseS7Type(typeName)
		if perr != nil {
			return point, perr
		}
		point.Type = t
	} else {
		point.Type = inferS7TypeBySuffix(widthSuffix)
	}

	return point, nil
}
