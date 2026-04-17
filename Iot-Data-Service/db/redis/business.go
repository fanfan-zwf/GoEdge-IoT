/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: redis 用户逻辑
 */

package redis

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
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
		return fmt.Errorf("MSet批量写入失败: %v", err)
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
