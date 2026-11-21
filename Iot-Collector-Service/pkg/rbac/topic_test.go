package rbac

import "testing"

func TestSplit(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/config/drive/query", []string{"config", "drive", "query"}},
		{"config/drive/query", []string{"config", "drive", "query"}}, // 前导 / 可选
		{"/a/+", []string{"a", "+"}},
		{"", nil},
		{"/", nil}, // 裁掉前导 / 后为空
	}
	for _, c := range cases {
		got := Split(c.in)
		if len(got) != len(c.want) {
			t.Errorf("Split(%q)=%v want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("Split(%q)=%v want %v", c.in, got, c.want)
				break
			}
		}
	}
}

func TestValidatePattern(t *testing.T) {
	ok := []string{"/a/b", "/a/+", "/a/#", "+", "#", "/a/+/c", "/org/{orgId}/device/+", "/a/*", "*", "`/a/b`", "`/a/[0-9]+`", "`.*`", "`/org/{orgId}/device/[0-9]+`"}
	for _, p := range ok {
		if err := ValidatePattern(p); err != nil {
			t.Errorf("ValidatePattern(%q) 期望合法，却报错: %v", p, err)
		}
	}
	bad := []string{"/a/#/b", "/a/#x", "/a/x#", "/a/b+", "/a/+b", "/a/x+y", "/a/x*y", "`[`", "a`b`", "``"}
	for _, p := range bad {
		if err := ValidatePattern(p); err == nil {
			t.Errorf("ValidatePattern(%q) 期望非法，却通过", p)
		}
	}
}

func TestReplacePlaceholders(t *testing.T) {
	params := map[string]string{"orgId": "5", "userId": "77"}
	if got, ok := ReplacePlaceholders("/org/{orgId}/user/{userId}", params); !ok || got != "/org/5/user/77" {
		t.Errorf("全量解析失败: got=%q ok=%v", got, ok)
	}
	if _, ok := ReplacePlaceholders("/org/{orgId}/x/{miss}", params); ok {
		t.Errorf("含未提供占位符应 ok=false")
	}
}

func TestMatch(t *testing.T) {
	params := map[string]string{"orgId": "5"}
	cases := []struct {
		pat, res string
		want     bool
	}{
		{"/config/drive/query", "/config/drive/query", true}, // 精确
		{"/config/drive/query", "/config/drive/add", false},  // 精确不等
		{"/config/drive/+", "/config/drive/query", true},     // 单层 +
		{"/config/drive/+", "/config/drive/a/b", false},      // + 只匹配一段
		{"/config/+/query", "/config/drive/query", true},     // 中间 +
		{"/config/drive/#", "/config/drive/query", true},     // 尾部 #
		{"/config/drive/#", "/config/drive", true},           // # 匹配零剩余段(父级)
		{"/config/drive/#", "/config/drive/a/b/c", true},     // # 匹配多段
		{"/config/drive/#", "/config/point/query", false},    // 前缀不符
		{"/org/{orgId}/device/+", "/org/5/device/9", true},   // 占位符命中
		{"/org/{orgId}/device/+", "/org/6/device/9", false},  // 占位符不符
		{"/org/{unknown}/x", "/org/6/x", false},              // 占位符无法解析→不匹配
		// * 同义单层通配
		{"/config/*/query", "/config/drive/query", true},
		{"/config/*/query", "/config/drive/add", false},
		// [..] 已废弃；改为整串正则（反引号包裹，对完整路径锚定整段）
		{"`/config/rbac/perm/del/[0-9]+`", "/config/rbac/perm/del/13", true},
		{"`/config/rbac/perm/del/[0-9]+`", "/config/rbac/perm/del/abc", false},
		{"`/config/rbac/perm/del`", "/config/rbac/perm/del", true},
		{"`/config/rbac/perm/del`", "/config/rbac/perm/del/13", false}, // 锚定整段，非前缀
		{"`/config/rbac/[^/]+/del/.*`", "/config/rbac/perm/del/13", true},
		{"`/config/rbac/[^/]+/del/.*`", "/config/rbac/perm/perm1/del/13", false},
		{"`/org/{orgId}/device/[0-9]+`", "/org/5/device/9", true}, // 正则内占位符
		{"`/org/{orgId}/device/[0-9]+`", "/org/6/device/9", false},
		// 等长精确：末尾空段≠13
		{"/config/rbac/perm/del/", "/config/rbac/perm/del/13", false},
		{"/config/rbac/perm/del/", "/config/rbac/perm/del/", true},
		{"/config/rbac/+/del", "/config/rbac/perm/del/13", false}, // + 只一段，段数不等
	}
	for _, c := range cases {
		if got := Match(c.pat, c.res, params); got != c.want {
			t.Errorf("Match(%q,%q)=%v want %v", c.pat, c.res, got, c.want)
		}
	}
}
