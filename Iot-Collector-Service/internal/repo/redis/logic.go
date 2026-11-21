/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: redis 用户逻辑
 */

package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	Default_Expiration_Second time.Duration = 60 * 60 * time.Second // 默认时间
	Short_Expiration_Second   time.Duration = 60 * 60 * time.Second // 短时间
)

/*
***************刷新令牌***************
 */
// 访问令牌结构体
type Refresh_Token_redis_type struct {
	User_Id       uint   // 用户id
	Terminal_Uuid string // 用户终端Id
	Expires_in    string // 访问令牌过期时间
}

// 创建访问令牌
func Refresh_Token_Add(User_Id uint, Refresh_Token string, Value Refresh_Token_redis_type, Expiration time.Duration) (err error) {
	if User_Id == 0 || Refresh_Token == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	// 结构体转换json
	jsonByte, err := json.Marshal(Value)
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	key := fmt.Sprintf("Refresh_Token:%d:%s", User_Id, Refresh_Token)
	err = Rdb.Set(ctx, key, string(jsonByte), Expiration).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// 查看访问令牌
func Refresh_Token_Query(User_Id uint, Refresh_Token string) (Access_Token_redis Refresh_Token_redis_type, err error) {
	if User_Id == 0 || Refresh_Token == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	key := fmt.Sprintf("Refresh_Token:%d:%s", User_Id, Refresh_Token)

	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = json.Unmarshal([]byte(val), &Access_Token_redis)
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

/*
***************访问令牌***************
 */
// 访问令牌结构体
type Access_Token_redis_type struct {
	User_Id       uint   // 用户id
	Expires_in    string // 访问令牌过期时间
	Refresh_Token string // 本访问令牌的刷新令牌
}

// 创建访问令牌
func Access_Token_Add(Access_Token string, Value Access_Token_redis_type, Expiration time.Duration) (err error) {
	// 结构体转换json
	jsonByte, err := json.Marshal(Value)
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	key := fmt.Sprintf("Access_Token:%s", Access_Token)
	err = Rdb.Set(ctx, key, string(jsonByte), Expiration).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// 查看访问令牌
func Access_Token_Query(Access_Token string) (Access_Token_redis Access_Token_redis_type, err error) {

	key := fmt.Sprintf("Access_Token:%s", Access_Token)

	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = json.Unmarshal([]byte(val), &Access_Token_redis)
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

/*
***************用户权限***************
 */
// 读取用户权限
func Authority_User_Query(User_Id uint, Authority_Theme string) (exist bool, err error) {
	var value string
	User_Id_str := fmt.Sprintf("%d", User_Id)
	value, err = Rdb.HGet(
		ctx,
		User_Id_str, // 用户id
		fmt.Sprintf("Authority_User:%s", Authority_Theme), // 权限主题
	).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = Rdb.Expire(ctx, User_Id_str, Default_Expiration_Second).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}

	switch value {
	case "true":
		exist = true
	case "false":
		exist = false
	default:
		err = fmt.Errorf("未知状态 %v", value)
	}

	return
}

// 写入用户权限
func Authority_User_Add(User_Id uint, Authority_Theme string, exist bool) (err error) {
	User_Id_str := fmt.Sprintf("%d", User_Id)
	err = Rdb.HSet(
		ctx,
		User_Id_str, // 用户id
		fmt.Sprintf("Authority_User:%s", Authority_Theme), // 权限主题
		fmt.Sprintf("%t", exist),
	).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = Rdb.Expire(ctx, User_Id_str, Default_Expiration_Second).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}

	return
}

// 删除指定的value
func Authority_User_Del_Value(User_Id uint, Authority_Theme ...string) (err error) {
	for i, v := range Authority_Theme {
		Authority_Theme[i] = fmt.Sprintf("Authority_User:%s", v)
	}

	User_Id_str := fmt.Sprintf("%d", User_Id)
	_, err = Rdb.HDel(
		ctx,
		User_Id_str,        // 用户id
		Authority_Theme..., // 权限主题
	).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

/*
***************用户密码输入错误锁定***************
 */

// 用户输入错误记录
func User_Passwd_Input_Error_Record(User_Id uint, Value string, Limit_TTL_Second uint) (length int, err error) {
	key := fmt.Sprintf("User_Passwd_Input_Error:%d", User_Id)

	var val int64
	val, err = Rdb.LPush(ctx, key, Value).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	length = int(val)

	err = Rdb.Expire(ctx, key, Default_Expiration_Second).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}

	return
}

func User_Passwd_Input_Error_Record_Del(User_Id uint) (err error) {
	key := fmt.Sprintf("User_Passwd_Input_Error:%d", User_Id)
	err = Rdb.Del(ctx, key).Err()
	return
}

/*
***************刷新令牌***************
 */
// 访问令牌结构体
type Api_Refresh_Token_redis_type struct {
	Api_Id     uint   // 用户id
	Expires_in string // 访问令牌过期时间
	Allow_Ip   string // 也许的ip
}

// 创建访问令牌
func Api_Refresh_Token_Add(Refresh_Token string, Value Api_Refresh_Token_redis_type, Expiration time.Duration) (err error) {
	if Value.Api_Id == 0 || Refresh_Token == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	// 结构体转换json
	jsonByte, err := json.Marshal(Value)
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	key := fmt.Sprintf("Api_Refresh_Token:%d:%s", Value.Api_Id, Refresh_Token)
	err = Rdb.Set(ctx, key, string(jsonByte), Expiration).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// 查看访问令牌
func Api_Refresh_Token_Query(Api_Id uint, Refresh_Token string) (Refresh_Token_redis Api_Refresh_Token_redis_type, err error) {
	if Api_Id == 0 || Refresh_Token == "" {
		err = fmt.Errorf("参数错误")
		return
	}

	key := fmt.Sprintf("Api_Refresh_Token:%d:%s", Api_Id, Refresh_Token)

	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = json.Unmarshal([]byte(val), &Refresh_Token_redis)
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

/*
***************访问令牌***************
 */
// 访问令牌结构体
type Api_Access_Token_redis_type struct {
	Api_Id        uint   // 用户id
	Expires_in    string // 访问令牌过期时间
	Refresh_Token string // 本访问令牌的刷新令牌
	Allow_Ip      string // 允许ip
}

// 创建访问令牌
func Api_Access_Token_Add(Access_Token string, Value Api_Access_Token_redis_type, Expiration time.Duration) (err error) {
	// 结构体转换json
	jsonByte, err := json.Marshal(Value)
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	key := fmt.Sprintf("Api_Access_Token:%s", Access_Token)
	err = Rdb.Set(ctx, key, string(jsonByte), Expiration).Err()
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// 查看访问令牌
func Api_Access_Token_Query(Access_Token string) (Access_Token_redis Api_Access_Token_redis_type, err error) {

	key := fmt.Sprintf("Api_Access_Token:%s", Access_Token)

	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("ERROR %v", err)
		return
	}

	err = json.Unmarshal([]byte(val), &Access_Token_redis)
	if err != nil {
		log.Printf("ERROR %v", err)
	}
	return
}

// KeyValue 定义你传入的key-value结构体格式
type KeyValue struct {
	Key   string        // 对应Redis的key
	Value string        // 对应Redis的value（已转成字符串）
	TTL   time.Duration // 可选：单个key的过期时间
}

// Write_Key_list 批量写入Redis（支持批量写入+单独设置TTL）
// 参数：data-要写入的数组
// 返回：error-错误信息
func Write_Key_list(data []KeyValue) error {
	// 校验1：空数据直接返回
	if len(data) == 0 {
		return nil
	}

	// 转换为[]interface{}类型（核心修复点）
	args := make([]interface{}, 0, len(data)*2) // 直接创建interface{}切片
	for idx, kv := range data {
		// 校验2：key不能为空
		if kv.Key == "" {
			return fmt.Errorf("第%d个元素的key为空，无效数据", idx)
		}
		// 校验3：value为空时给出警告（可选）
		if kv.Value == "" {
			log.Printf("WARNING 第%d个元素的value为空（key=%s）\n", idx, kv.Key)
		}
		// 直接添加string到interface{}切片（Go允许string隐式转interface{}）
		args = append(args, kv.Key, kv.Value)
	}

	// 校验4：确保参数是偶数个（key-value成对）
	if len(args)%2 != 0 {
		return fmt.Errorf("参数元素数为%d（奇数），key-value不成对", len(args))
	}

	// 批量写入：现在参数类型完全匹配
	err := Rdb.MSet(ctx, args...).Err()
	if err != nil {
		return fmt.Errorf("MSet批量写入失败: %w", err)
	}

	// 遍历设置TTL（仅对非零TTL的key生效）
	for idx, kv := range data {
		// 跳过无过期时间的key
		if kv.TTL <= 0 {
			continue
		}
		// 单独设置过期时间
		expireErr := Rdb.Expire(ctx, kv.Key, kv.TTL).Err()
		if expireErr != nil {
			// 可根据需求调整：是返回错误中断，还是仅记录警告
			log.Printf("ERROR 警告：第%d个key(%s)设置TTL失败: %v\n", idx, kv.Key, expireErr)
			// 如果需要严格失败，取消下面注释：
			// return fmt.Errorf("第%d个key(%s)设置TTL失败: %v", idx, kv.Key, expireErr)
		}
	}

	return nil
}

// Read_Key 读取单个Redis键值
// 参数：ctx-上下文，keys-要读取的key数组
// 返回：KeyValue map[string]string，error-错误信息
func Read_Key(key string) (value string, err error) {
	// 校验：key为空直接返回空值+无错误（或根据需求返回错误）
	if key == "" {
		return
	}

	// 单个key读取：使用Get命令替代MGet
	value, err = Rdb.Get(ctx, key).Result()
	return
}

// Read_Key_list 批量读取Redis
// 参数：ctx-上下文，keys-要读取的key数组
// 返回：KeyValue map[string]string，error-错误信息
func Read_Key_list(keys []string) (keyvalue map[string]string, err error) {
	// 空 keys 直接返回空 map
	if len(keys) == 0 {
		return
	}
	// 初始化返回的 map
	keyvalue = make(map[string]string, len(keys))

	// 批量获取值：MGet 命令会按 keys 顺序返回对应的值，不存在的键返回 nil
	values, err := Rdb.MGet(ctx, keys...).Result()
	if err != nil {
		// Redis 命令执行失败（如网络错误、连接失败）
		err = fmt.Errorf("redis MGet 执行失败: %w", err)
		return
	}

	// 遍历结果，组装键值对
	for i, key := range keys {
		val := values[i]
		// 跳过 nil 值（键不存在的情况）
		if val == nil {
			continue
		}
		// 将值转换为 string 类型
		strVal, ok := val.(string)
		if !ok {
			// 值类型不是 string（如数字、哈希等），这里做兼容处理
			strVal = fmt.Sprintf("%v", val)
		}
		keyvalue[key] = strVal
	}

	return
}

/*
***************统一鉴权 access_key 版令牌 / nonce***************
 */

// Auth_Token_Info 令牌在 Redis 中的绑定值（JSON 格式），记录归属账号与登录环境，用于会话一致性校验
type Auth_Token_Info struct {
	User_Id    uint      `json:"user_id"` // 登录用户 user_account.Id；登录/签发时写入，请求期直接取用免查库
	Access_Key string    `json:"access_key"`
	IP         string    `json:"ip"`         // 签发时的客户端 IP
	User_Agent string    `json:"user_agent"` // 签发时的浏览器 User-Agent
	Issued_At  time.Time `json:"issued_at"`  // 签发时间，time.Time 序列化为 RFC3339Nano（如 2026-11-11T22:05:38.738129642+08:00）
}

// Auth_ApiKey_Creds AK/SK 验签所需凭据的 Redis 缓存值：HMAC 热路径据此完成验签与状态校验，避免每请求查 MySQL
type Auth_ApiKey_Creds struct {
	Id       uint   `json:"id"`
	Secret   string `json:"secret"`
	Enable   int    `json:"enable"`    // 1=启用 0=禁用
	ExpireAt int64  `json:"expire_at"` // Unix 秒；0=永不过期
}

const (
	authAccessKeyPrefix  = "auth:access_token:"
	authRefreshKeyPrefix = "auth:refresh_token:"
	// authAccountSetPrefix 账号 → 令牌 key 的索引集合，用于 IP 不匹配时将该账号全部令牌立即失效
	authAccountSetPrefix = "auth:account_tokens:"
	// authApiKeyCredsPrefix AK/SK 验签凭据缓存 key 前缀（按 access_key 存）
	authApiKeyCredsPrefix = "auth:apikey_creds:"
)

// authTokenIndexAdd 将令牌 key 登记到账号索引集合，并把集合 TTL 续期到不小于令牌 TTL
func authTokenIndexAdd(accessKey, redisKey string, ttl time.Duration) error {
	if accessKey == "" || redisKey == "" {
		return nil
	}
	setKey := authAccountSetPrefix + accessKey
	if err := Rdb.SAdd(ctx, setKey, redisKey).Err(); err != nil {
		return err
	}
	return Rdb.Expire(ctx, setKey, ttl).Err()
}

// authTokenUnmarshal 解析令牌值；旧格式（纯 access_key 字符串 / 未含 user_id 的历史令牌）或数据损坏均视为无效，迫使重新登录
func authTokenUnmarshal(raw string) (Auth_Token_Info, bool) {
	var info Auth_Token_Info
	if err := json.Unmarshal([]byte(raw), &info); err != nil || info.Access_Key == "" || info.User_Id == 0 {
		return info, false
	}
	return info, true
}

// authTokenDel 删除令牌 key，并同步从账号索引集合中摘除（读不到绑定信息时仍照常删除）
func authTokenDel(prefix, token string) error {
	if token == "" {
		return nil
	}
	key := prefix + token
	if raw, err := Rdb.Get(ctx, key).Result(); err == nil {
		if info, ok := authTokenUnmarshal(raw); ok {
			_ = Rdb.SRem(ctx, authAccountSetPrefix+info.Access_Key, key).Err()
		}
	}
	return Rdb.Del(ctx, key).Err()
}

// authTokenGet 按前缀读取令牌绑定信息；ok=false 表示不存在/已过期/格式非法（此时 err=nil）
func authTokenGet(prefix, token string) (info Auth_Token_Info, ok bool, err error) {
	if token == "" {
		return info, false, nil
	}
	raw, err := Rdb.Get(ctx, prefix+token).Result()
	if errors.Is(err, goredis.Nil) {
		return info, false, nil
	}
	if err != nil {
		log.Printf("ERROR authTokenGet(%s): %v", prefix, err)
		return info, false, err
	}
	info, ok = authTokenUnmarshal(raw)
	return info, ok, nil
}

// Auth_Access_Token_Add 写入短时效 access_token，value 为 JSON 绑定信息（access_key / IP / User-Agent）
// 过期时间由调用方按账号 TTL 传入，同时将令牌 key 登录到账号索引集合
func Auth_Access_Token_Add(token string, info Auth_Token_Info, ttl time.Duration) error {
	if token == "" || info.Access_Key == "" {
		return fmt.Errorf("Auth_Access_Token_Add: 参数错误")
	}
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("Auth_Access_Token_Add: 序列化失败: %v", err)
	}

	key := authAccessKeyPrefix + token
	if err := Rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		return err
	}
	return authTokenIndexAdd(info.Access_Key, key, ttl)
}

// Auth_Access_Token_Get 读取 access_token 绑定的会话信息（含 IP / User-Agent）
// ok=false 表示 token 不存在或已过期（此时 err=nil）；Redis 本身故障时返回 err
func Auth_Access_Token_Get(token string) (info Auth_Token_Info, ok bool, err error) {
	return authTokenGet(authAccessKeyPrefix, token)
}

// Auth_ApiKey_Creds_Set 写入/刷新某 access_key 的验签凭据缓存（固定过期，由调用方传 TTL）
func Auth_ApiKey_Creds_Set(accessKey string, c Auth_ApiKey_Creds, ttl time.Duration) error {
	if accessKey == "" {
		return fmt.Errorf("Auth_ApiKey_Creds_Set: access_key 不能为空")
	}
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("Auth_ApiKey_Creds_Set: 序列化失败: %v", err)
	}
	return Rdb.Set(ctx, authApiKeyCredsPrefix+accessKey, data, ttl).Err()
}

// Auth_ApiKey_Creds_Get 读取 AK/SK 验签凭据缓存；ok=false 表示未命中/已过期/数据损坏（此时 err=nil）；Redis 故障返回 err
func Auth_ApiKey_Creds_Get(accessKey string) (c Auth_ApiKey_Creds, ok bool, err error) {
	if accessKey == "" {
		return c, false, nil
	}
	raw, err := Rdb.Get(ctx, authApiKeyCredsPrefix+accessKey).Result()
	if errors.Is(err, goredis.Nil) {
		return c, false, nil
	}
	if err != nil {
		log.Printf("ERROR Auth_ApiKey_Creds_Get: %v", err)
		return c, false, err
	}
	if json.Unmarshal([]byte(raw), &c) != nil {
		return Auth_ApiKey_Creds{}, false, nil // 损坏视为未命中，回源重建
	}
	return c, true, nil
}

// Auth_ApiKey_Creds_Del 删除某 access_key 的验签凭据缓存（AK/SK 删除或同步下线的账号）
func Auth_ApiKey_Creds_Del(accessKey string) error {
	if accessKey == "" {
		return nil
	}
	return Rdb.Del(ctx, authApiKeyCredsPrefix+accessKey).Err()
}

// Auth_Access_Token_Del 删除指定 access_token（刷新/登出时使旧 token 失效）
func Auth_Access_Token_Del(token string) error {
	return authTokenDel(authAccessKeyPrefix, token)
}

// Auth_Refresh_Token_Add 写入长时效 refresh_token，value 同样为 JSON 绑定信息（语义同 Auth_Access_Token_Add）
func Auth_Refresh_Token_Add(token string, info Auth_Token_Info, ttl time.Duration) error {
	if token == "" || info.Access_Key == "" {
		return fmt.Errorf("Auth_Refresh_Token_Add: 参数错误")
	}
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("Auth_Refresh_Token_Add: 序列化失败: %v", err)
	}

	key := authRefreshKeyPrefix + token
	if err := Rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		return err
	}
	return authTokenIndexAdd(info.Access_Key, key, ttl)
}

// Auth_Refresh_Token_Get 读取 refresh_token 绑定的会话信息（语义同 Auth_Access_Token_Get）
func Auth_Refresh_Token_Get(token string) (info Auth_Token_Info, ok bool, err error) {
	return authTokenGet(authRefreshKeyPrefix, token)
}

// Auth_Refresh_Token_Del 删除指定 refresh_token（刷新轮换时使旧 refresh token 失效）
func Auth_Refresh_Token_Del(token string) error {
	return authTokenDel(authRefreshKeyPrefix, token)
}

// Auth_Account_Token_Logout 使某账号的全部 access/refresh 令牌立即失效（登录 IP 变化时的“账号全部退出”）
func Auth_Account_Token_Logout(accessKey string) error {
	if accessKey == "" {
		return nil
	}
	setKey := authAccountSetPrefix + accessKey
	keys, err := Rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		log.Printf("ERROR Auth_Account_Token_Logout: 读取账号令牌索引失败: %v", err)
		return err
	}
	if len(keys) > 0 {
		if err := Rdb.Del(ctx, keys...).Err(); err != nil {
			return err
		}
	}
	return Rdb.Del(ctx, setKey).Err()
}

// Auth_Nonce_Add 用 SetNX 记录 nonce 防重放，key 按 access_key 隔离
// 返回 fresh=true 表示首次出现（放行）；fresh=false 表示已存在（重放，拒绝）
func Auth_Nonce_Add(accessKey, nonce string, ttl time.Duration) (fresh bool, err error) {
	if accessKey == "" || nonce == "" {
		return false, fmt.Errorf("Auth_Nonce_Add: 参数错误")
	}

	key := fmt.Sprintf("auth:nonce:%s:%s", accessKey, nonce)
	fresh, err = Rdb.SetNX(ctx, key, 1, ttl).Result()
	if err != nil {
		log.Printf("ERROR Auth_Nonce_Add: %v", err)
		return false, err
	}
	return fresh, nil
}

// Auth_AkSk_Touch_Throttle AK/SK 通路没有显式登录动作，用 SetNX 做“首次使用”节流：
// 窗口内第一次调用返回 first=true（调用方据此写一条登陆日志），后续返回 false。
// key 按 access_key 隔离，ttl 由调用方传入（建议远大于 HMAC 时间窗口，避免登陆日志刷屏）
func Auth_AkSk_Touch_Throttle(accessKey string, ttl time.Duration) (first bool, err error) {
	if accessKey == "" {
		return false, fmt.Errorf("Auth_AkSk_Touch_Throttle: 参数错误")
	}

	key := fmt.Sprintf("auth:aksk:touch:%s", accessKey)
	first, err = Rdb.SetNX(ctx, key, 1, ttl).Result()
	if err != nil {
		log.Printf("ERROR Auth_AkSk_Touch_Throttle: %v", err)
		return false, err
	}
	return first, nil
}
