/*
* 日期: 2025.10.29 PM3:31
* 作者: 范范zwf
* 作用: 消息队列
 */

package redis

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var (
	ctx = context.Background()
)

/*******************消息队列*******************/

// 定义一个结构体
type Queue_struct struct {
	Key string // redis key值
	mu  sync.Mutex
}

// 定义接口
type Queue_interface interface {
	New(key string) error // 初始化

	// 输入值
	Write(value string) error

	// 获取消息队列长度
	Length() (int, error)

	// 输出
	Read() (string, error)

	// 全部输出
	Read_array() ([]string, error)
}

func (c *Queue_struct) New(key string) {
	c.Key = key
}

// 输入值
func (c *Queue_struct) Write(value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return Rdb.LPush(ctx, c.Key, value).Err()
}

func (c *Queue_struct) Length() (int, error) {
	length, err := Rdb.LLen(ctx, c.Key).Result()
	return int(length), err
}

// 输出
func (c *Queue_struct) Read() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result, err := Rdb.BRPop(ctx, 100*time.Millisecond, c.Key).Result()
	if result[0] != c.Key {
		return "", fmt.Errorf("redis 读取key不一致")
	}

	return result[1], err
}

// 全部输出
func (c *Queue_struct) Read_array() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 使用 LUA 脚本原子性地获取并删除所有消息
	script := `
        local messages = redis.call('LRANGE', KEYS[1], 0, -1)
        redis.call('DEL', KEYS[1])
        return messages
    `

	result, err := Rdb.Eval(ctx, script, []string{c.Key}).Result()
	if err != nil {
		return nil, err
	}

	// 类型转换
	if result == nil {
		return []string{}, nil
	}

	interfaces := result.([]interface{})
	messages := make([]string, len(interfaces))
	for i, v := range interfaces {
		messages[i] = v.(string)
	}

	return messages, nil
}
