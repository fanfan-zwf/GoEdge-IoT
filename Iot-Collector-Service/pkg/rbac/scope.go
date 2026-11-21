/*
* 日期: 2026.11.20
* 作用: 实例级（数据级）授权的纯函数核心。主题形如 /config/drive/query/{id}，
*       通配 /config/drive/query/* 代表"全部可见/可操作"。服务层解析出规则后调用本文件计算 id 集合。
*       约定：真实行 id 从 1 起，哨兵 id "0" 只可能被 * / # / 正则等宽规则命中，用以探测"全量"。
 */

package rbac

import (
	"strconv"
)

// Can 单资源主题判定；action 因规则集 Action 恒为 "*" 实际不参与匹配，仅为可读性传入。
func Can(rules []Rule, params map[string]string, topic, action string) bool {
	return Decide(topic, action, params, rules).Allowed
}

// ScopeAll 用哨兵 id "0" 探测 base 下是否"全量"：仅 * / # / 正则等宽规则会命中 "0"。
func ScopeAll(rules []Rule, params map[string]string, base, action string) bool {
	return Can(rules, params, base+"/0", action)
}

// FilterIds 逐 id 判定，返回候选 ids 中被允许的有序子集（用于列表可见集合 / 批量可操作子集）。
func FilterIds(rules []Rule, params map[string]string, base, action string, ids []uint) []uint {
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if Can(rules, params, base+"/"+strconv.FormatUint(uint64(id), 10), action) {
			out = append(out, id)
		}
	}
	return out
}
