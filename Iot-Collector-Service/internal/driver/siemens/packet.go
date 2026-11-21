/*
* 日期: 2026.8.26
* 作者: 范范zwf
* 作用: Siemens S7 组包 —— 按 Area+DBNumber 分组，合并连续字节地址
 */

package siemens

import (
	"collector/pkg/conv"
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	s7MaxItemsPerRead = 20  // snap7 Cli_ReadMultiVars/WriteMultiVars 单次最大 item 数（MaxVars 硬限制）
	s7DefaultPDUSize  = 480 // 默认 PDU 长度（字节） Smart200=240 300=240 400=960 1200=480 1500=960
	s7RequestPDU      = 960 // 连接时向 PLC 请求的 PDU 上限；协商后取 PLC 实际支持值（各型号上限 240/480/960）
	s7ItemOverhead    = 17  // 每个 DataItem 的协议开销（请求侧 12 字节 item 规格 + 响应侧 4 字节返回头，取上界并留裕量）
	s7PduBaseOverhead = 18  // S7 报文固定开销（请求头 10+参数计数 1，响应头 12+参数 2，取较大值并留裕量）；PDU 是整包大小含头，数据区须先扣除
)

// S7GroupKey S7 组包分组键：Area + DBNumber
type S7GroupKey struct {
	Area     int
	DBNumber int
}

// S7PackPoint S7 组包输入点位
type S7PackPoint struct {
	Id        uint // 点位id
	Area      int  // 存储区：0x84 = DB块
	DBNumber  int  // DB号
	Start     int  // 字节偏移
	ByteLen   int  // 数据类型的字节长度（由 Value_Type 决定）
	ValueType int  // 输出类型（透传，用于后续解析）
}

// S7PackResultAddr 中
// 你定义的结构体（完全保留）
type PackAddressPackages_Point_type struct {
	Id        uint   // 点位id
	StartAddr uint16 // 点位开始值
	DataLen   uint16 // 点位类型长度
	EndAddr   uint16 // 内部计算用
}

// 组包结果结构
type PackageResult struct {
	StartAddr uint16
	DataLen   uint16
	Id        []uint // 改成 uint 匹配你的结构
}

// PackAddressPackages 终极修复版 → 连续地址1、2、3一定会合并！
func PackAddressPackages(addrList []PackAddressPackages_Point_type, maxPackageLen uint16) ([]PackageResult, error) {
	if len(addrList) == 0 {
		return nil, errors.New("输入点位列表不能为空")
	}

	// 自动计算 EndAddr + 过滤有效点位
	var validPoints []PackAddressPackages_Point_type
	for _, p := range addrList {
		if p.DataLen <= 0 {
			fmt.Printf("id=%d\n", p.Id)
			continue
		}
		p.EndAddr = p.StartAddr + p.DataLen - 1
		validPoints = append(validPoints, p)
	}

	// 按起始地址排序
	sort.Slice(validPoints, func(i, j int) bool {
		return validPoints[i].StartAddr < validPoints[j].StartAddr
	})

	var packages []PackageResult

	// 核心合并逻辑（连续地址必合并）
	for _, point := range validPoints {
		merged := false

		// 尝试合并到最后一个包
		if len(packages) > 0 {
			last := &packages[len(packages)-1]
			lastEnd := last.StartAddr + last.DataLen - 1

			// ==============================
			// 关键：只要 连续 / 重叠 就合并
			// ==============================
			if point.StartAddr <= lastEnd+1 { // +1 兼容连续地址
				newEnd := max(lastEnd, point.EndAddr)
				newLen := newEnd - last.StartAddr + 1

				// 不超限才合并
				if maxPackageLen == 0 || newLen <= maxPackageLen {
					last.DataLen = newLen
					last.Id = append(last.Id, point.Id)
					merged = true
				}
			}
		}

		// 不能合并 → 新建包
		if !merged {
			packages = append(packages, PackageResult{
				StartAddr: point.StartAddr,
				DataLen:   point.DataLen,
				Id:        []uint{point.Id},
			})
		}
	}

	return packages, nil
}

// IsTimePassed
// a: 前面的时间
// b: 后面的时间
// c: 需要判断是否过去的时长
// return: true = a在b前面 + 间隔时间 >= c；false = 不满足任一条件
func IsTimePassed(a, b time.Time, c time.Duration) bool {
	// 1. 必须满足：a 在前面，b 在后面
	if !a.Before(b) {
		return false
	}

	// 2. 计算两个时间的间隔
	duration := b.Sub(a)

	// 3. 判断间隔是否 >= 指定时长
	return duration > c
}

type S7PackResultAddr struct {
	Start int    // 起始字节偏移
	End   int    // 结束字节偏移（含）
	Ids   []uint // 包含的点位id
	Types []int  // 对应的 ValueType
}

// S7PackResult S7 组包最终结果
type S7PackResult struct {
	Area     int    // 存储区
	DBNumber int    // DB号
	Start    int    // 起始字节偏移
	ByteLen  int    // 总字节长度（End - Start + 1）
	Ids      []uint // 包含的点位id
	Types    []int  // 对应的 ValueType
}

// ValueTypeToByteLen 将 Value_Type 整数编码转换为字节长度
func ValueTypeToByteLen(vt int) int {
	switch vt {
	case conv.TYPE_BOOL: // bool
		return 1
	case conv.TYPE_INT8, conv.TYPE_UINT8: // int8, uint8
		return 1
	case conv.TYPE_INT16, conv.TYPE_UINT16: // int16, uint16
		return 2
	case conv.TYPE_INT32, conv.TYPE_UINT32: // int32, uint32
		return 4
	case conv.TYPE_INT64, conv.TYPE_UINT64: // int64, uint64
		return 8
	case conv.TYPE_INT, conv.TYPE_UINT: // int, uint
		return 8
	case conv.TYPE_FLOAT32, conv.TYPE_FLOAT: // float32, float
		return 4
	case conv.TYPE_FLOAT64: // float64
		return 8
	case conv.TYPE_STRING: // string（变长，默认按4字节处理）
		return 4
	default:
		return 0
	}
}

// PackS7AddressPackages 按 Area+DBNumber 分组后，合并连续/重叠字节地址
//   - points: 需要组包的点位列表
//   - maxPacketLen: 单个包最大字节长度，0 表示使用默认 PDU 大小
//
// 返回: 组包结果列表，每个结果对应一次 S7 读取请求
func PackS7AddressPackages(points []S7PackPoint, maxPacketLen int) ([]S7PackResult, error) {
	if len(points) == 0 {
		return nil, errors.New("输入点位列表不能为空")
	}

	if maxPacketLen <= 0 {
		maxPacketLen = s7DefaultPDUSize
	}

	// PDU 是整个报文大小（含 S7 头 + 每 item 协议开销）；单个合并包 = 一个 ReadMultiVars item，
	// 其数据区上限需先扣除基础头与单 item 开销，否则响应报文会超过协商 PDU 被 PLC 拒绝
	maxData := maxPacketLen - s7PduBaseOverhead - s7ItemOverhead
	if maxData <= 0 {
		maxData = 1
	}

	// 1. 按 Area + DBNumber 分组，同时过滤掉 ByteLen <= 0 的无效点位
	groups := make(map[S7GroupKey][]S7PackPoint)
	for _, p := range points {
		if p.ByteLen <= 0 {
			fmt.Printf("S7 组包: 跳过无效点位 id=%d (ByteLen=%d)\n", p.Id, p.ByteLen)
			continue
		}
		key := S7GroupKey{Area: p.Area, DBNumber: p.DBNumber}
		groups[key] = append(groups[key], p)
	}

	var results []S7PackResult

	// 2. 对每个分组独立排序 + 合并
	for key, groupPoints := range groups {
		// 按字节偏移排序
		sort.Slice(groupPoints, func(i, j int) bool {
			return groupPoints[i].Start < groupPoints[j].Start
		})

		// Timer(0x1C) 和 Counter(0x1D) 区域不合并，每个点位单独成包
		// 因为 Timer/Counter 的 Start 是编号而非字节偏移，ByteLen 固定为 2
		if key.Area == S7AreaCT || key.Area == S7AreaTM {
			for _, point := range groupPoints {
				results = append(results, S7PackResult{
					Area:     key.Area,
					DBNumber: key.DBNumber,
					Start:    point.Start,
					ByteLen:  2, // 每个 Timer/Counter 固定 2 字节
					Ids:      []uint{point.Id},
					Types:    []int{point.ValueType},
				})
			}
			continue
		}

		// 合并连续/重叠地址（仅适用于普通区域：PE/PA/MK/DB）
		var merged []S7PackResultAddr

		for _, point := range groupPoints {
			pointEnd := point.Start + point.ByteLen - 1
			mergedFlag := false

			if len(merged) > 0 {
				last := &merged[len(merged)-1]

				// 连续或重叠 → 尝试合并
				if point.Start <= last.End+1 {
					newEnd := max(last.End, pointEnd)
					newLen := newEnd - last.Start + 1

					if newLen <= maxData {
						last.End = newEnd
						last.Ids = append(last.Ids, point.Id)
						last.Types = append(last.Types, point.ValueType)
						mergedFlag = true
					}
				}
			}

			if !mergedFlag {
				merged = append(merged, S7PackResultAddr{
					Start: point.Start,
					End:   pointEnd,
					Ids:   []uint{point.Id},
					Types: []int{point.ValueType},
				})
			}
		}

		// 3. 转换为最终结果
		for _, m := range merged {
			results = append(results, S7PackResult{
				Area:     key.Area,
				DBNumber: key.DBNumber,
				Start:    m.Start,
				ByteLen:  m.End - m.Start + 1,
				Ids:      m.Ids,
				Types:    m.Types,
			})
		}
	}

	return results, nil
}
