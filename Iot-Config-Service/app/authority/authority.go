package authority

import (
	"main/app/user_service"
	"strings"
)

type AuthorityMatcher interface {
	// 函数A：从MySQL获取用户所有权限 []string
	GetUserThemesFromDB(userID uint) ([]string, error)

	// 函数B：设置需要匹配的模板列表（包含:xxx）
	SetMatchPatterns(patterns []string)

	// 函数C：解析单个模板中的 :变量名
	ExtractVarName(pattern string) string

	// 函数D：执行匹配 → 返回 map[模板]bool
	Match() (map[string]bool, error)
}

// 实现结构体
type MyAuthorityMatcher struct {
	token     string
	patterns  []string // 匹配模板列表
	dbThemes  []string // 从数据库获取的权限列表
	resultMap map[string]bool
}

func NewMyAuthorityMatcher(token string, patterns []string) (m *MyAuthorityMatcher) {
	m = &MyAuthorityMatcher{}
	m.GetUserThemesFromDB(token, patterns)

	return
}

// 函数A：从你原有函数获取权限列表（完全调用你原来的函数！）
func (m *MyAuthorityMatcher) GetUserThemesFromDB(token string, patterns []string) (r []string, err error) {
	m.token = token

	var (
		authority_list   user_service.Api_User_Authority_Theme_List__type
		theme_exist_list []string
	)
	authority_list, err = user_service.Api_User_Authority_Theme_List(token, patterns)
	if err != nil {
		return
	}
	for theme, exist := range authority_list.Authority_Theme_List {
		if exist {
			theme_exist_list = append(theme_exist_list, theme)
		}
	}
	m.patterns = theme_exist_list

	return
}

// 函数B：设置需要匹配的模板（//xxx//:xxx/xxx）
func (m *MyAuthorityMatcher) SetMatchPatterns(patterns []string) {
	m.patterns = patterns
}

// 函数C：提取所有 :变量 → 支持多个变量 ✅
func (m *MyAuthorityMatcher) ExtractVarNames(pattern string) (vars []string) {
	parts := strings.Split(pattern, "/")
	for _, p := range parts {
		if strings.HasPrefix(p, ":") {
			vars = append(vars, strings.TrimPrefix(p, ":"))
		}
	}
	return
}

// 函数D：核心匹配（支持 :xxx 单层通配）
func (m *MyAuthorityMatcher) Match() (result map[string]bool, err error) {
	result = make(map[string]bool)

	for _, pattern := range m.patterns {
		matched := false
		patternParts := strings.Split(pattern, "/")

		for _, theme := range m.dbThemes {
			themeParts := strings.Split(theme, "/")

			// 层级必须一样
			if len(themeParts) != len(patternParts) {
				continue
			}

			// 逐段匹配
			ok := true
			for i := range patternParts {
				p := patternParts[i]
				t := themeParts[i]

				if strings.HasPrefix(p, ":") {
					continue // 通配
				}
				if p != t {
					ok = false
					break
				}
			}

			if ok {
				matched = true
				break
			}
		}

		result[pattern] = matched
	}

	return
}

// 内部：判断真实权限是否匹配模板
func (m *MyAuthorityMatcher) isMatch(theme, pattern string) bool {
	tParts := strings.Split(theme, "/")
	pParts := strings.Split(pattern, "/")

	if len(tParts) != len(pParts) {
		return false
	}

	for i := range pParts {
		p := pParts[i]
		t := tParts[i]
		if strings.HasPrefix(p, ":") {
			continue
		}
		if p != t {
			return false
		}
	}

	return true
}

// 函数C：【你真正要的核心功能】
// 传入 varName = project
// 返回所有匹配的值：[test, test1]
// =============================================================================
func (m *MyAuthorityMatcher) ExtractVarValues(pattern string, varName string) []string {
	patternParts := strings.Split(pattern, "/")

	// 找到变量在模板的位置
	varIndex := -1
	for i, part := range patternParts {
		if strings.TrimPrefix(part, ":") == varName {
			varIndex = i
			break
		}
	}
	if varIndex < 0 {
		return nil
	}

	// 遍历所有真实权限，提取对应位置的值
	var res []string
	for _, theme := range m.dbThemes {
		if !m.isMatch(theme, pattern) {
			continue
		}

		themeParts := strings.Split(theme, "/")
		if len(themeParts) > varIndex {
			res = append(res, themeParts[varIndex])
		}
	}

	return res
}
