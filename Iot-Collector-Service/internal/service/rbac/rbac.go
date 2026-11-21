/*
* 日期: 2026.11.20
* 作用: 极简 RBAC 校验服务层：对外校验入口 + 具体主题判定结果 Redis 缓存。
*
* 判定流程（详见 pkg/rbac.Decide）：候选规则 → deny 优先拦截 → 精确>+># → match_sort → 默认拒绝。
* 缓存模型（本层核心）：
*   - 粒度下沉到"判定结果"：一个 key = 一个主体对一个具体主题的裁决；
*   - key 分主体类型：rbac:user:{userId}:{主题}（业务）/ rbac:api:{clientId}:{主题}（API）；
*   - value(JSON) 含 allow / topic(命中规则的 topic_key 模式) / role(来源角色) / sort(优先级)，便于排障；
*   - 纯 TTL 懒失效：TTL=config.RBAC.TTL，到期 Redis 自动删，下次请求回源 MySQL 重算并写回；
*     allow 与 deny 结果都缓存（防反复回源）；命中不续期（固定过期，保证最多 cache_ttl 后必刷新）；
*   - 无主动失效：RBAC 写操作后新权限最多 cache_ttl 后生效（不再有 SCAN+DEL 全清）；
*   - Redis 故障时读写各自安全降级（Get/MGet 报错视为未命中、Set/Pipeline 忽略错误），回源直连 MySQL，
*     正确性不受影响，仅暂时失去缓存加速。
* 安全要点：恒强制拦截、无超管旁路、默认拒绝、主体身份强制注入防伪造（{userId}/{clientId} 由真实 id 覆盖）。
 */

package rbac

import (
	"collector/internal/config"
	"collector/internal/repo/mysql"
	redisrepo "collector/internal/repo/redis"
	rbacmatch "collector/pkg/rbac"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// cacheValue 具体主题判定结果缓存值：允许位 + 命中溯源（模式主题/来源角色/优先级）。
type cacheValue struct {
	Allow bool   `json:"allow"`
	Topic string `json:"topic"` // 命中规则的 topic_key 模式串（默认拒绝时为空）
	Role  string `json:"role"`  // 命中规则的来源角色编码（直接授权为 "direct"）
	Sort  int    `json:"sort"`  // 命中规则的 match_sort 优先级
}

// decKey 生成判定结果缓存键：业务 rbac:user:{id}:{主题}，API rbac:api:{id}:{主题}。
func decKey(subjectType uint64, subjectId uint, topic string) string {
	kind := "api"
	if subjectType == mysql.Rbac_Subject_Biz {
		kind = "user"
	}
	return fmt.Sprintf("rbac:%s:%d:%s", kind, subjectId, topic)
}

// injectSubject 复制调用方 params 并强制覆盖主体身份占位符，防止调用方伪造 {userId}/{clientId}。
func injectSubject(caller map[string]string, key string, id uint64) map[string]string {
	m := make(map[string]string, len(caller)+1)
	for k, v := range caller {
		m[k] = v
	}
	m[key] = strconv.FormatUint(id, 10)
	return m
}

// scopeParams 依主体类型注入身份占位符（与判定入口同口径，防伪造）。
func scopeParams(subjectType uint64, subjectId uint) map[string]string {
	if subjectType == mysql.Rbac_Subject_Api {
		return injectSubject(nil, "clientId", uint64(subjectId))
	}
	return injectSubject(nil, "userId", uint64(subjectId))
}

// compute 用规则集对具体主题跑一次判定引擎，抽取结果与溯源组装缓存值。
func compute(topic string, params map[string]string, rules []rbacmatch.Rule) cacheValue {
	d := rbacmatch.Decide(topic, "*", params, rules)
	return cacheValue{Allow: d.Allowed, Topic: d.TopicKey, Role: d.Role, Sort: d.MatchSort}
}

// decideOne 单主题判定：命中 Redis 直接返回；未命中回源 MySQL 计算并写回（allow/deny 都缓存）。
// Redis 不可用视为未命中，降级直连 MySQL；MySQL 故障返回 error（调用方从严处理）。
func decideOne(subjectType uint64, subjectId uint, topic string) (cacheValue, error) {
	key := decKey(subjectType, subjectId, topic)
	ctx := context.Background()
	if data, err := redisrepo.Rdb.Get(ctx, key).Bytes(); err == nil {
		var v cacheValue
		if json.Unmarshal(data, &v) == nil {
			return v, nil
		}
		// 反序列化异常视为未命中，回源重建
	}

	rules, err := mysql.LoadEffectiveRules(subjectType, uint64(subjectId))
	if err != nil {
		return cacheValue{}, err
	}
	v := compute(topic, scopeParams(subjectType, subjectId), rules)
	if data, mErr := json.Marshal(v); mErr == nil {
		_ = redisrepo.Rdb.Set(ctx, key, data, config.Config.RBAC.TTL).Err()
	}
	return v, nil
}

// decideBatch 批量主题判定（供 FilterIds 用）：MGET 取命中，miss 集合仅回源一次规则再逐个算并 Pipeline 写回。
// 返回 topic→结果；规则加载失败时返回已命中结果 + error（调用方据 error 从严）。
func decideBatch(subjectType uint64, subjectId uint, topics []string) (map[string]cacheValue, error) {
	result := make(map[string]cacheValue, len(topics))
	if len(topics) == 0 {
		return result, nil
	}
	ctx := context.Background()

	keys := make([]string, len(topics))
	for i, t := range topics {
		keys[i] = decKey(subjectType, subjectId, t)
	}

	var miss []string
	if vals, err := redisrepo.Rdb.MGet(ctx, keys...).Result(); err == nil {
		for i, raw := range vals {
			s, ok := raw.(string)
			if !ok {
				miss = append(miss, topics[i])
				continue
			}
			var v cacheValue
			if json.Unmarshal([]byte(s), &v) != nil {
				miss = append(miss, topics[i])
				continue
			}
			result[topics[i]] = v
		}
	} else {
		miss = append(miss, topics...) // Redis 故障：全部回源
	}

	if len(miss) > 0 {
		rules, err := mysql.LoadEffectiveRules(subjectType, uint64(subjectId))
		if err != nil {
			return result, err
		}
		params := scopeParams(subjectType, subjectId)
		pipe := redisrepo.Rdb.Pipeline()
		for _, topic := range miss {
			v := compute(topic, params, rules)
			result[topic] = v
			if data, mErr := json.Marshal(v); mErr == nil {
				pipe.Set(ctx, decKey(subjectType, subjectId, topic), data, config.Config.RBAC.TTL)
			}
		}
		_, _ = pipe.Exec(ctx) // 写失败忽略：不影响本次判定正确性
	}
	return result, nil
}

// checkSubject 判定入口公共实现：对具体资源主题求裁决，回填溯源供日志。
func checkSubject(subjectType uint64, subjectId uint, resource string) (rbacmatch.Decision, error) {
	v, err := decideOne(subjectType, subjectId, resource)
	if err != nil {
		return rbacmatch.Decision{Allowed: false, Reason: "load_error"}, err
	}
	reason := "cached_allow"
	if !v.Allow {
		reason = "cached_deny"
	}
	return rbacmatch.Decision{
		Allowed:   v.Allow,
		Reason:    reason,
		TopicKey:  v.Topic,
		Role:      v.Role,
		MatchSort: v.Sort,
	}, nil
}

// CheckBizUser 业务用户权限校验入口。
// 传入：userId — user_account.Id；resource — 具体资源主题串（含真实 id 段）；action/params 保留签名兼容，
// 实际判定以主体真实 id 注入占位符后按主题匹配（action 维度已退化）。
func CheckBizUser(userId uint, resource, action string, params map[string]string) (rbacmatch.Decision, error) {
	return checkSubject(mysql.Rbac_Subject_Biz, userId, resource)
}

// CheckApiClient API 客户端权限校验入口；{clientId} 由判定用真实 id 注入。
func CheckApiClient(clientId uint, resource, action string, params map[string]string) (rbacmatch.Decision, error) {
	return checkSubject(mysql.Rbac_Subject_Api, clientId, resource)
}

// ===================== 实例级（数据级）授权 scope 计算 =====================
// 用于驱动/点位/报警/历史等"接口+资源行 id"两级授权：主题形如 /config/drive/query/{id}，
// 通配 /config/drive/query/* 代表"全部可见/可操作"（经哨兵 base+"/0" 探测）。

// Can 单资源主题判定（restart 单 id、add 通配放行等）。判定失败 → false（从严）。
func Can(subjectType uint64, subjectId uint, topic, action string) bool {
	v, err := decideOne(subjectType, subjectId, topic)
	if err != nil {
		return false
	}
	return v.Allow
}

// ScopeAll 判断主体在 base 下是否"全量可见/可操作"：探测哨兵主题 base+"/0"。
func ScopeAll(subjectType uint64, subjectId uint, base, action string) bool {
	v, err := decideOne(subjectType, subjectId, base+"/0")
	if err != nil {
		return false
	}
	return v.Allow
}

// FilterIds 对候选 ids 求授权子集（保序）。批量走 MGET+单次回源；判定失败 → 返回空（从严拒绝）。
func FilterIds(subjectType uint64, subjectId uint, base, action string, ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	topics := make([]string, len(ids))
	for i, id := range ids {
		topics[i] = base + "/" + strconv.FormatUint(uint64(id), 10)
	}
	res, err := decideBatch(subjectType, subjectId, topics)
	if err != nil {
		return nil
	}
	out := make([]uint, 0, len(ids))
	for i, id := range ids {
		if res[topics[i]].Allow {
			out = append(out, id)
		}
	}
	return out
}

// SubjectAuth 为登录回填前端主体权限：返回角色编码集合与"允许"类权限编码集合。
// 登录期低频，直接回源 MySQL（不经判定结果缓存，因其需要枚举全部权限而非单主题裁决）。
func SubjectAuth(subjectType uint64, subjectId uint) (roles, perms []string, err error) {
	rules, err := mysql.LoadEffectiveRules(subjectType, uint64(subjectId))
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rules {
		if r.Allow {
			perms = append(perms, r.PermCode)
		}
	}
	roles, err = mysql.LoadRoleCodes(subjectType, uint64(subjectId))
	if err != nil {
		return nil, nil, err
	}
	return roles, perms, nil
}
