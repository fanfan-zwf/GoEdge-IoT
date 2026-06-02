/*
* 日期: 2025.5.29 PM10:17
* 作者: 范范zwf
* 作用: modbus tcp组包
 */

package Modbus_Tcp

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// 你定义的结构体（完全保留）
type PackAddressPackages_Point_type struct {
	Tag       string // 点位名称
	StartAddr uint16 // 点位开始值
	DataLen   uint16 // 点位类型长度
	EndAddr   uint16 // 内部计算用
}

// 组包结果结构
type PackageResult struct {
	StartAddr uint16
	DataLen   uint16
	Tags      []string // 改成 uint 匹配你的结构
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
			fmt.Printf("跳过无效点位：Tag=%s\n", p.Tag)
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
					last.Tags = append(last.Tags, point.Tag)
					merged = true
				}
			}
		}

		// 不能合并 → 新建包
		if !merged {
			packages = append(packages, PackageResult{
				StartAddr: point.StartAddr,
				DataLen:   point.DataLen,
				Tags:      []string{point.Tag},
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
