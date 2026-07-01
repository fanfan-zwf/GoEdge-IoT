package web

import (
	"strings"
)

// 结构体
type MyAuthorityMatcher struct {
	token     string
	patterns  []string // 匹配模板列表
	dbThemes  []string // 从数据库获取的权限列表
	resultMap map[string]bool
}

// 接口
type AuthorityMatcher interface {
	SetDBThemes(themes []string)
	SetPatterns(patterns []string)
	ExtractVarValues(varName string) []string // 只传变量名！
	Match() map[string]bool
}

// ----------------------------------------------
// 实现
// ----------------------------------------------
func (m *MyAuthorityMatcher) SetDBThemes(themes []string) {
	m.dbThemes = themes
}

func (m *MyAuthorityMatcher) SetPatterns(patterns []string) {
	m.patterns = patterns
}

// 核心匹配逻辑（支持 * 和 :）
func (m *MyAuthorityMatcher) isMatch(theme, pattern string) bool {
	tParts := strings.Split(theme, "/")
	pParts := strings.Split(pattern, "/")

	// 结尾 /* 匹配后面所有层级
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(theme, prefix+"/")
	}

	// 层级必须一致
	if len(tParts) != len(pParts) {
		return false
	}

	for i := range pParts {
		p := pParts[i]
		t := tParts[i]

		if p == "*" || strings.HasPrefix(p, ":") {
			continue
		}
		if p != t {
			return false
		}
	}
	return true
}

// 匹配结果
func (m *MyAuthorityMatcher) Match() map[string]bool {
	res := make(map[string]bool)
	for _, pattern := range m.patterns {
		found := false
		for _, theme := range m.dbThemes {
			if m.isMatch(theme, pattern) {
				found = true
				break
			}
		}
		res[pattern] = found
	}
	m.resultMap = res
	return res
}

// =============================================================================
// varName = project，自动提取所有值！
// =============================================================================
func (m *MyAuthorityMatcher) ExtractVarValues(varName string) []string {
	valueSet := make(map[string]struct{}) // 去重

	// 遍历所有模板
	for _, pattern := range m.patterns {
		patternParts := strings.Split(pattern, "/")

		// 找到变量在模板里的位置
		varIndex := -1
		for i, part := range patternParts {
			if strings.TrimPrefix(part, ":") == varName {
				varIndex = i
				break
			}
		}
		if varIndex == -1 {
			continue
		}

		// 遍历所有真实权限，提取值
		for _, theme := range m.dbThemes {
			if m.isMatch(theme, pattern) {
				tParts := strings.Split(theme, "/")
				if len(tParts) > varIndex {
					val := tParts[varIndex]
					valueSet[val] = struct{}{} // 自动去重
				}
			}
		}
	}

	// map 转数组
	var res []string
	for v := range valueSet {
		res = append(res, v)
	}
	return res
}

// func main() {
// 	// 1. 创建实例
// 	matcher := &MyAuthorityMatcher{}

// 	// 2. 设置数据库权限
// 	matcher.SetDBThemes([]string{
// 		"//api//test/admin/user",
// 		"//api//test1/admin/user",
// 		"//api//demo/xxx/list",
// 		"//api//test/abc/123",
// 	})

// 	// 3. 设置多个模板（每个模板有不同 :变量）
// 	matcher.SetPatterns([]string{
// 		"//api//:project/*/user",    // 变量 project
// 		"//api//:project/xxx/list",  // 变量 project
// 		"//api//:name/123",          // 变量 name
// 	})

// 	// 4. 匹配结果
// 	matchResult := matcher.Match()
// 	fmt.Println("匹配结果：", matchResult)

// 	// =========================================================================
// 	// ✅ 【你要的终极写法】只传 project，自动提取所有值！
// 	// =========================================================================
// 	values := matcher.ExtractVarValues("project")
// 	fmt.Println("project 所有值：", values)
// 	// 输出：[test, test1, demo]

// 	// 也可以提取别的变量
// 	nameValues := matcher.ExtractVarValues("name")
// 	fmt.Println("name 所有值：", nameValues)
// 	// 输出：[test]
// }
