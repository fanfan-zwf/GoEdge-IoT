/*
* 日期: 2025.12.21 16:40
* 作者: 范范zwf
* 作用: redis 用户逻辑
 */

package redis

import (
	"fmt"
	"log"
	"time"
)

// KeyValue 定义你传入的key-value结构体格式
type KeyValue struct {
	Key   string        // 对应Redis的key
	Value string        // 对应Redis的value（已转成字符串）
	TTL   time.Duration // 可选：单个key的过期时间
}

// Write_Key_list 批量写入Redis（支持批量写入+单独设置TTL）
// 参数：data-要写入的数组
// 返回：error-错误信息
func Write_Key_list(data ...KeyValue) error {
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
