/*
* 日期: 2026.11.19
* 作用: 极简 RBAC 判定核心（纯函数、无数据库依赖）。
*       给定资源/动作/占位符上下文与主体的有效规则集合，按优先级返回允许或拒绝。
*
* 判定流程：
*   1) 候选规则 = 主题命中(先解析占位符，见 Match) 且 动作命中(rule.Action==action 或 rule.Action=="*")；
*   2) 候选按优先级排序（高→低）：deny 先于 allow（拒绝优先拦截）
*      → 精确 > 单层 +/* > 尾部 # > 整串正则（ specificity）→ match_sort 升序（数字小者优先）→ 权限 id 升序；
*   3) 取排名第一的规则，其 effect 即为最终结果；
*   4) 无任何候选 → 默认拒绝（兜底拦截）。
 */

package rbac

import "sort"

// Rule 单条有效权限规则（由数据层从 t_permission 关联加载而来）。
type Rule struct {
	PermId    uint64 `json:"perm_id"`
	PermCode  string `json:"perm_code"`
	TopicKey  string `json:"topic_key"`  // 类 MQTT 模式，可含 + / 尾部 # / {占位符}
	Action    string `json:"action"`     // view/edit/delete/export/*
	MatchSort int    `json:"match_sort"` // 数字越小优先级越高
	Allow     bool   `json:"allow"`      // effect：true=allow false=deny
	Role      string `json:"role"`       // 授予该规则的来源角色编码（直接授权为 "direct"）
}

// Decision 判定结果，附带命中信息便于日志与排障。
type Decision struct {
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason"`
	PermCode  string `json:"perm_code"`  // 命中的权限编码（默认拒绝时为空）
	HitId     uint64 `json:"hit_id"`     // 命中的权限字典 id
	TopicKey  string `json:"topic_key"`  // 命中规则的资源主题模式串
	MatchSort int    `json:"match_sort"` // 命中规则的优先级
	Role      string `json:"role"`       // 命中规则的来源角色编码
}

// specificity 计算模式具体度分层：精确(0) > 单层 +/*(1) > 尾部 #(2) > 整串正则(3)，值越小越具体。
func specificity(pattern string) int {
	// 整条被反引号包裹的整串正则最宽，具体度最低
	if _, isRe := regexBody(pattern); isRe {
		return 3
	}
	segs := Split(pattern)
	if len(segs) == 0 {
		return 0
	}
	if segs[len(segs)-1] == "#" {
		return 2
	}
	for _, s := range segs {
		if s == "+" || s == "*" {
			return 1
		}
	}
	return 0
}

// actionMatch 判断规则动作是否覆盖请求动作。
func actionMatch(ruleAction, action string) bool {
	return ruleAction == "*" || ruleAction == action
}

// Decide 对 (resource, action, params) 在 rules 中做出允许/拒绝判定。
// params 用于解析主题模式里的 {orgId}/{userId}/{clientId} 等占位符。
func Decide(resource, action string, params map[string]string, rules []Rule) Decision {
	type cand struct {
		rule     Rule
		denyRank int // deny=0 排在前（优先拦截），allow=1
		tier     int // 具体度分层
	}
	cands := make([]cand, 0, len(rules))
	for _, r := range rules {
		if !actionMatch(r.Action, action) {
			continue
		}
		if !Match(r.TopicKey, resource, params) { // 内含占位符解析；未解析成功即判不命中
			continue
		}
		denyRank := 1
		if !r.Allow {
			denyRank = 0
		}
		cands = append(cands, cand{rule: r, denyRank: denyRank, tier: specificity(r.TopicKey)})
	}

	// 无候选 → 默认拒绝兜底
	if len(cands) == 0 {
		return Decision{Allowed: false, Reason: "no_match_default_deny"}
	}

	sort.SliceStable(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.denyRank != b.denyRank {
			return a.denyRank < b.denyRank // deny 优先
		}
		if a.tier != b.tier {
			return a.tier < b.tier // 精确 > + > #
		}
		if a.rule.MatchSort != b.rule.MatchSort {
			return a.rule.MatchSort < b.rule.MatchSort // match_sort 小者优先
		}
		return a.rule.PermId < b.rule.PermId // id 升序保证稳定
	})

	win := cands[0].rule
	reason := "matched_allow"
	if !win.Allow {
		reason = "matched_deny_intercept"
	}
	return Decision{
		Allowed:   win.Allow,
		Reason:    reason,
		PermCode:  win.PermCode,
		HitId:     win.PermId,
		TopicKey:  win.TopicKey,
		MatchSort: win.MatchSort,
		Role:      win.Role,
	}
}
