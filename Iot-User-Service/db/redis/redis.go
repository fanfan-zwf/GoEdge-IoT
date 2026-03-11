package redis

import (
	"main/Init"

	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

var Rdb *redis.Client

func Init_rdb(Addr string, Passwd string, Database int) {

	Rdb = redis.NewClient(&redis.Options{
		Addr:     Addr,
		Password: Passwd,   // 密码为空
		DB:       Database, // 使用默认数据库
	})

	ctx := context.Background()
	if _, err := Rdb.Ping(ctx).Result(); err != nil {
		panic("连接失败: " + err.Error())
	}

}

func init() {
	Config := Init.Config

	Init_rdb(
		fmt.Sprintf("%s:%d", Config.REDIS.Ip, Config.REDIS.Post),
		Config.REDIS.Passwd,
		Config.REDIS.Database,
	)
}

// key搜索
func Redis_scanKeys(ctx context.Context, rdb *redis.Client, pattern string, count int64) ([]string, error) {
	var allKeys []string
	var cursor uint64
	var totalIterations int

	for {
		// 执行SCAN命令
		keys, nextCursor, err := rdb.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			return nil, fmt.Errorf("SCAN命令执行失败: %w", err)
		}

		allKeys = append(allKeys, keys...)
		totalIterations++

		// 检查是否完成遍历 (游标为0表示结束)
		if nextCursor == 0 {
			break
		}

		// 更新游标，继续下一次迭代
		cursor = nextCursor

		// 安全措施：避免意外无限循环
		if totalIterations > 1000 {
			return nil, fmt.Errorf("超过最大迭代次数，可能陷入无限循环")
		}
	}

	return allKeys, nil
}
