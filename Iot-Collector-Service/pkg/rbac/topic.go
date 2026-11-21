/*
* 日期: 2026.11.19
* 作用: 极简 RBAC 主题匹配工具（类 MQTT），纯函数、无数据库依赖。两种匹配模式：
*   A. 整串正则：主题被反引号 ` 包裹时，反引号内当一个 Go 正则，对完整路径串做锚定整段匹配；
*      例 `/config/rbac/perm/del/[0-9]+` 命中 /config/rbac/perm/del/13。
*   B. 分段匹配（默认）：按斜杠 / 分段逐段比较（通配符须独占整段）：
*      - + 或 * ：单层通配，匹配恰好一段（二者等价，* 为 + 的同义写法）；
*      - #      ：仅允许出现在最后一段，匹配其后所有段（含零段）；
*      无 # 时，模式与资源段数必须完全相等（故 /config/rbac/+/del 不命中 5 段的 .../del/13）。
*   两种模式均支持 {orgId}/{userId}/{clientId} 占位符，未提供取值时该规则判为不匹配（安全兜底）。
 */

package rbac

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// placeholderRe 匹配形如 {name} 的占位符，name 由字母/数字/下划线组成。
// 仅用于应用层扫描替换，不涉及任何 MySQL 正则。
var placeholderRe = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

// Normalize 归一化主题串：去首尾空白；若有前导 / 只裁掉一个，使 "/a/b" 与 "a/b" 等价，
// 从而让模式与资源走同一套切分规则、段数对齐。
func Normalize(topic string) string {
	t := strings.TrimSpace(topic)
	t = strings.TrimPrefix(t, "/")
	return t
}

// Split 按 / 切分为段。空串返回 nil（视为无段）。
func Split(topic string) []string {
	t := Normalize(topic)
	if t == "" {
		return nil
	}
	return strings.Split(t, "/")
}

// ValidatePattern 校验主题模式合法性：
//   - 若整条主题被反引号 ` 包裹，视为整串正则，反引号内须能编译为合法 Go 正则；
//   - 否则按 / 逐段校验：# 只能作为最后一段且独占整段；+ / * 必须独占整段（禁止 a+、+b、x*y）。
//
// 非法返回 error；供加载权限规则时提前拦截脏数据。
func ValidatePattern(topic string) error {
	if strings.Contains(topic, "`") {
		inner, ok := regexBody(topic)
		if !ok {
			return fmt.Errorf("主题模式 %q 非法：反引号须成对包裹整条主题作为整串正则", topic)
		}
		if inner == "" {
			return fmt.Errorf("主题模式 %q 非法：正则内容为空", topic)
		}
		// 占位符 {name} 在 RE2 里会被当成量词报假错，校验时先替为字面量 0 再试编译（运行时 Match 会先做真实替换）
		probe := placeholderRe.ReplaceAllString(inner, "0")
		if _, err := regexp.Compile(probe); err != nil {
			return fmt.Errorf("主题模式 %q 非法：正则编译失败: %v", topic, err)
		}
		return nil
	}
	segs := Split(topic)
	if len(segs) == 0 {
		return fmt.Errorf("主题模式为空: %q", topic)
	}
	for i, seg := range segs {
		if seg == "#" && i != len(segs)-1 {
			return fmt.Errorf("主题模式 %q 非法：# 只能出现在最后一段", topic)
		}
		// 段内只要包含 # + *，就必须整段恰好等于 # / + / *
		if strings.ContainsAny(seg, "#+*") && seg != "#" && seg != "+" && seg != "*" {
			return fmt.Errorf("主题模式 %q 非法：第 %d 段 %q 含通配符但未独占整段", topic, i+1, seg)
		}
	}
	return nil
}

// regexBody 提取被反引号整条包裹的正则原文；ok=false 表示并非合法的整串正则写法。
func regexBody(topic string) (string, bool) {
	t := strings.TrimSpace(topic)
	if len(t) < 2 || t[0] != '`' || t[len(t)-1] != '`' {
		return "", false
	}
	body := t[1 : len(t)-1]
	if strings.ContainsRune(body, '`') { // 反引号只允许成对出现在首尾
		return "", false
	}
	return body, true
}

// rxCache 缓存“已锚定的整串正则”，键为解析占位符后的正则原文，避免每次 Match 重新编译。
var rxCache sync.Map // string -> *regexp.Regexp（编译失败存 nil）

// anchoredMatch 把正则原文编译（带缓存）后对完整路径串做整段匹配（隐式 ^(?:...)$ 锚定）。
func anchoredMatch(body, resource string) bool {
	if v, ok := rxCache.Load(body); ok {
		rx, _ := v.(*regexp.Regexp)
		return rx != nil && rx.MatchString(resource)
	}
	rx, err := regexp.Compile("^(?:" + body + ")$")
	if err != nil {
		rxCache.Store(body, (*regexp.Regexp)(nil))
		return false
	}
	rxCache.Store(body, rx)
	return rx.MatchString(resource)
}

// ReplacePlaceholders 用 params 替换 topic 中的 {name} 占位符。
// 只要有任何一个占位符在 params 中取不到值，就返回 ok=false（调用方应据此把该规则判为不匹配）。
func ReplacePlaceholders(topic string, params map[string]string) (string, bool) {
	resolved := true
	out := placeholderRe.ReplaceAllStringFunc(topic, func(m string) string {
		key := m[1 : len(m)-1] // 去掉 { }
		if v, found := params[key]; found {
			return v
		}
		resolved = false
		return m
	})
	return out, resolved
}

// Match 判断资源 resource 是否命中模式 pattern（先替换占位符再匹配）。
// 两种模式：
//   - 整串正则：pattern 被反引号 ` 包裹时，把反引号内当 Go 正则，对完整路径串做锚定整段匹配；
//   - 分段匹配：否则按 / 逐段比较，+/* 匹配恰好一段，段尾 # 匹配其后所有段（含零段），无 # 时段数须相等。
//
// 若模式含占位符但 params 无法解析，直接判为不匹配。
func Match(pattern, resource string, params map[string]string) bool {
	if strings.Contains(pattern, "`") {
		body, ok := regexBody(pattern)
		if !ok {
			return false
		}
		resolved, ok := ReplacePlaceholders(body, params)
		if !ok {
			return false
		}
		return anchoredMatch(resolved, resource)
	}
	resolved, ok := ReplacePlaceholders(pattern, params)
	if !ok {
		return false
	}
	return matchSegs(Split(resolved), Split(resource))
}

// matchSegs 对已切分、已解析占位符的模式段与资源段做 MQTT 匹配。
func matchSegs(pat, res []string) bool {
	for i := 0; i < len(pat); i++ {
		if pat[i] == "#" { // # 恒为最后一段（ValidatePattern 已保证）
			return len(res) >= i // 匹配第 i 段及之后的所有段，允许零剩余段
		}
		if i >= len(res) {
			return false // 模式还需要更多段，而资源已结束
		}
		if pat[i] == "+" || pat[i] == "*" {
			continue // + / * 恰好匹配一段
		}
		if pat[i] != res[i] {
			return false
		}
	}
	return len(pat) == len(res) // 无 # 时，段数必须完全相等
}
