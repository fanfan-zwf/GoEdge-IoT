/*
* 日期: 2026.8.26
* 作者: 范范zwf
* 作用: Siemens S7 组包 —— 按 Area+DBNumber 分组，合并连续字节地址
 */

package siemenss7

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	s7MaxItemsPerRead = 20  // S7 协议 AGReadMulti 单次最大 item 数
	s7DefaultPDUSize  = 480 // 默认 PDU 长度（字节） Smart200=240 300=240 400=960 1200=480 1500=960
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
// 1:bool 2:int8 3:uint8 4:int16 5:uint16 6:int32 7:uint32 8:int64 9:uint64 10:int 11:uint 12:float32 13:float64 14:float 15:string
func ValueTypeToByteLen(vt int) int {
	switch vt {
	case 1: // bool
		return 1
	case 2, 3: // int8, uint8
		return 1
	case 4, 5: // int16, uint16
		return 2
	case 6, 7: // int32, uint32
		return 4
	case 8, 9: // int64, uint64
		return 8
	case 10, 11: // int, uint
		return 8
	case 12, 14: // float32, float
		return 4
	case 13: // float64
		return 8
	case 15: // string（变长，默认按4字节处理）
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

		// 合并连续/重叠地址
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

					if newLen <= maxPacketLen {
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
