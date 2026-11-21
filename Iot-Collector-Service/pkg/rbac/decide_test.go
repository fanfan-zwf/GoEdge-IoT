package rbac

import "testing"

func r(id uint64, code, topic, action string, sort int, allow bool) Rule {
	return Rule{PermId: id, PermCode: code, TopicKey: topic, Action: action, MatchSort: sort, Allow: allow}
}

func TestDecide_ExactAllow(t *testing.T) {
	rules := []Rule{r(1, "drive.view", "/config/drive/query", "view", 10, true)}
	d := Decide("/config/drive/query", "view", nil, rules)
	if !d.Allowed || d.PermCode != "drive.view" {
		t.Fatalf("期望精确允许命中, got=%+v", d)
	}
}

func TestDecide_ActionMismatch(t *testing.T) {
	rules := []Rule{r(1, "drive.view", "/config/drive/query", "view", 10, true)}
	d := Decide("/config/drive/query", "delete", nil, rules)
	if d.Allowed || d.Reason != "no_match_default_deny" {
		t.Fatalf("动作不符应默认拒绝, got=%+v", d)
	}
}

func TestDecide_StarAction(t *testing.T) {
	rules := []Rule{r(1, "all", "/config/drive/#", "*", 10, true)}
	if d := Decide("/config/drive/del", "delete", nil, rules); !d.Allowed {
		t.Fatalf("* 动作应覆盖 delete, got=%+v", d)
	}
}

func TestDecide_DenyInterceptsOverAllow(t *testing.T) {
	// allow 精确 + deny 尾部 #：请求同时命中，拒绝优先拦截
	rules := []Rule{
		r(1, "allow.query", "/config/drive/query", "view", 10, true),
		r(2, "deny.hash", "/config/drive/#", "*", 99, false),
	}
	d := Decide("/config/drive/query", "view", nil, rules)
	if d.Allowed || d.Reason != "matched_deny_intercept" || d.PermCode != "deny.hash" {
		t.Fatalf("期望 deny 优先拦截, got=%+v", d)
	}
}

func TestDecide_SpecificityOrdering(t *testing.T) {
	// 同为 allow，但精确规则更具体：命中结果都是 allow；这里用 deny 分层验证 tier 生效
	rules := []Rule{
		r(1, "hash.deny", "/config/#", "*", 1, false),          // tier=2 #
		r(2, "plus.allow", "/config/drive/+", "view", 5, true), // tier=1 +
	}
	// 资源命中两条；deny(#,tier2) 因 denyRank 优先于 allow(plus,tier1) —— 拒绝优先拦截
	d := Decide("/config/drive/query", "view", nil, rules)
	if d.Allowed {
		t.Fatalf("deny 应优先于 allow(无论具体度), got=%+v", d)
	}
}

func TestDecide_MatchSortAmongSameTier(t *testing.T) {
	// 两条 allow 精确、tier 相同，match_sort 小者优先命中
	rules := []Rule{
		r(10, "win", "/a/b", "view", 1, true),
		r(20, "lose", "/a/b", "view", 5, true),
	}
	d := Decide("/a/b", "view", nil, rules)
	if d.PermCode != "win" {
		t.Fatalf("match_sort 小者优先, got=%+v", d)
	}
}

func TestDecide_NoRuleDefaultDeny(t *testing.T) {
	d := Decide("/anything", "view", nil, nil)
	if d.Allowed || d.Reason != "no_match_default_deny" {
		t.Fatalf("无规则应默认拒绝, got=%+v", d)
	}
}

func TestDecide_RegexAllow(t *testing.T) {
	rules := []Rule{r(1, "perm.del.re", "`/config/rbac/perm/del/[0-9]+`", "*", 10, true)}
	if d := Decide("/config/rbac/perm/del/13", "delete", nil, rules); !d.Allowed {
		t.Fatalf("整串正则应命中, got=%+v", d)
	}
	if d := Decide("/config/rbac/perm/del/abc", "delete", nil, rules); d.Allowed {
		t.Fatalf("非数字不应命中, got=%+v", d)
	}
}

func TestDecide_RegexLeastSpecific(t *testing.T) {
	// 同为 allow、match_sort 相同：精确(0) 胜过 整串正则(3)
	rules := []Rule{
		r(1, "re", "`/x/[0-9]+`", "view", 1, true), // tier=3
		r(2, "exact", "/x/7", "view", 1, true),     // tier=0
	}
	if d := Decide("/x/7", "view", nil, rules); d.PermCode != "exact" {
		t.Fatalf("精确应胜过正则, got=%+v", d)
	}
}

func TestDecide_PlaceholderScope(t *testing.T) {
	params := map[string]string{"userId": "77"}
	rules := []Rule{r(1, "self.profile", "/user/{userId}/profile", "view", 10, true)}
	if d := Decide("/user/77/profile", "view", params, rules); !d.Allowed {
		t.Fatalf("本人应命中, got=%+v", d)
	}
	if d := Decide("/user/78/profile", "view", params, rules); d.Allowed {
		t.Fatalf("他人不应命中, got=%+v", d)
	}
	// 缺少 userId 参数 → 规则不解析 → 默认拒绝
	if d := Decide("/user/77/profile", "view", nil, rules); d.Allowed {
		t.Fatalf("占位符无法解析应默认拒绝, got=%+v", d)
	}
}
