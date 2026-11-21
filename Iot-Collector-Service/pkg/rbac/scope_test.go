/*
* 日期: 2026.11.20
* 作用: 实例级 scope 计算纯函数核心（Can/ScopeAll/FilterIds）的表驱动单测（无 DB、直接传内存规则集）。
 */

package rbac

import "testing"

const scopeBase = "/config/drive/query"

func TestScopeAll(t *testing.T) {
	cases := []struct {
		name  string
		rules []Rule
		want  bool
	}{
		{"通配*判全量", []Rule{r(1, "q", scopeBase+"/*", "*", 10, true)}, true},
		{"尾部#判全量", []Rule{r(1, "q", scopeBase+"/#", "*", 10, true)}, true},
		{"整串正则判全量", []Rule{r(1, "q", "`"+scopeBase+`/[0-9]+`+"`", "*", 10, true)}, true},
		{"仅显式id非全量", []Rule{r(1, "q", scopeBase+"/5", "*", 10, true)}, false},
		{"*allow+7deny仍全量", []Rule{
			r(1, "allow", scopeBase+"/*", "*", 10, true),
			r(2, "deny", scopeBase+"/7", "*", 10, false),
		}, true},
		{"空规则非全量", nil, false},
	}
	for _, c := range cases {
		if got := ScopeAll(c.rules, nil, scopeBase, "view"); got != c.want {
			t.Errorf("%s: ScopeAll=%v want %v", c.name, got, c.want)
		}
	}
}

func TestFilterIds(t *testing.T) {
	cases := []struct {
		name  string
		rules []Rule
		ids   []uint
		want  []uint
	}{
		{"通配全放行", []Rule{r(1, "q", scopeBase+"/*", "*", 10, true)}, []uint{1, 2, 3}, []uint{1, 2, 3}},
		{"仅显式id", []Rule{r(1, "q", scopeBase+"/5", "*", 10, true)}, []uint{1, 5, 9}, []uint{5}},
		{"*allow排除7", []Rule{
			r(1, "allow", scopeBase+"/*", "*", 10, true),
			r(2, "deny", scopeBase+"/7", "*", 10, false),
		}, []uint{5, 6, 7}, []uint{5, 6}},
		{"无任何授权", []Rule{r(1, "q", scopeBase+"/5", "*", 10, true)}, []uint{1, 2}, []uint{}},
		{"空规则空结果", nil, []uint{1, 2}, []uint{}},
	}
	for _, c := range cases {
		got := FilterIds(c.rules, nil, scopeBase, "view", c.ids)
		if len(got) != len(c.want) {
			t.Errorf("%s: FilterIds=%v want %v", c.name, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: FilterIds=%v want %v", c.name, got, c.want)
				break
			}
		}
	}
}

func TestCan_ScopeTopic(t *testing.T) {
	rules := []Rule{r(1, "d", "/config/drive/restart/*", "*", 10, true)}
	if !Can(rules, nil, "/config/drive/restart/5", "edit") {
		t.Errorf("通配应允许 restart/5")
	}
	if Can(rules, nil, "/config/drive/restart", "edit") {
		t.Errorf("缺 id 段(等长)不应命中 /restart/*")
	}
}
