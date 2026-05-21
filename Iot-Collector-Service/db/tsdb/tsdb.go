package tsdb

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

// ====================== 你要的结构体 ======================
type PointData struct {
	Tag   string          // 点位ID
	Value json.RawMessage // 任意类型值
	Time  time.Time       // 时间
	TTL   time.Duration   // 过期时间
}

// ====================== 你要的查询结构体 ======================
type PointQuery struct {
	Tag       string
	StartTime time.Time
	EndTime   time.Time
}

// ====================== 返回格式 ======================
type PointResult struct {
	Time  string          // RFC3339Nano 时间字符串
	Value json.RawMessage // 值
}

const keySeparator = "|"

var db *badger.DB

// ====================== 初始化DB ======================
func InitDB() {
	opts := badger.DefaultOptions("./badger-data")
	var err error
	db, err = badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
}

// ====================== 1. 写入：传入结构体 ======================
func SavePoint(data PointData) error {
	if db == nil {
		return fmt.Errorf("db not init")
	}

	key := buildKey(&data)
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry(key, val).WithTTL(data.TTL)
		return txn.SetEntry(entry)
	})
}

// ====================== 2. 批量范围查询 ======================
// 入参：[]PointQuery
// 返回：map[点位ID][]PointResult
func BatchQueryByTimeRange(queries []PointQuery) (map[string][]PointResult, error) {
	result := make(map[string][]PointResult)

	err := db.View(func(txn *badger.Txn) error {
		for _, q := range queries {
			prefix := []byte(q.Tag + keySeparator)
			startKey := buildKey(&PointData{Tag: q.Tag, Time: q.StartTime})
			endKey := buildKey(&PointData{Tag: q.Tag, Time: q.EndTime})

			it := txn.NewIterator(badger.DefaultIteratorOptions)

			var list []PointResult

			for it.Seek(startKey); it.Valid(); it.Next() {
				key := it.Item().Key()
				keyStr := string(key)

				if keyStr > string(endKey) {
					break
				}
				if !strings.HasPrefix(keyStr, string(prefix)) {
					break
				}

				// 【修复点1】使用 ValueCopy 正确获取数据
				val, err := it.Item().ValueCopy(nil)
				if err != nil {
					return err
				}

				var data PointData
				if err := json.Unmarshal(val, &data); err != nil {
					return err
				}

				list = append(list, PointResult{
					Time:  data.Time.Format(time.RFC3339Nano),
					Value: data.Value,
				})
			}
			it.Close() // 【优化点】及时关闭迭代器，避免 defer 在循环中累积
			result[q.Tag] = list
		}
		return nil
	})
	return result, err
}

// ====================== 3. 批量查【最后一次写入】的值 ======================
// 入参：[]点位ID
// 返回：map[点位ID]PointResult
func BatchGetLastValue(pointIDs []string) (map[string]PointResult, error) {
	result := make(map[string]PointResult)

	err := db.View(func(txn *badger.Txn) error {
		for _, pid := range pointIDs {
			prefix := []byte(pid + keySeparator)
			// 倒序迭代，取第一条就是最新的
			opts := badger.DefaultIteratorOptions
			opts.Reverse = true
			opts.Prefix = prefix

			it := txn.NewIterator(opts)

			var last PointResult
			if it.Rewind(); it.Valid() {
				// 【修复点2】使用 ValueCopy 正确获取数据
				val, err := it.Item().ValueCopy(nil)
				if err != nil {
					it.Close()
					return err
				}

				var data PointData
				if err := json.Unmarshal(val, &data); err != nil {
					it.Close()
					return err
				}

				last.Time = data.Time.Format(time.RFC3339Nano)
				last.Value = data.Value

				if last.Value != nil {
					result[pid] = last
				}
			}
			it.Close() // 【优化点】及时关闭迭代器
		}
		return nil
	})
	return result, err
}

// ====================== 工具：构建key ======================
func buildKey(pd *PointData) []byte {
	timeStr := pd.Time.Format(time.RFC3339Nano)
	return []byte(pd.Tag + keySeparator + timeStr)
}

// ====================== 测试示例 ======================
func main() {
	InitDB()
	defer db.Close()

	// 1. 写入测试
	fmt.Println("=== 写入测试 ===")
	_ = SavePoint(PointData{
		Tag:   "tag1",
		Value: json.RawMessage(`25.6`),
		Time:  time.Now().Add(-2 * time.Hour),
		TTL:   24 * time.Hour,
	})
	_ = SavePoint(PointData{
		Tag:   "tag1",
		Value: json.RawMessage(`26.1`),
		Time:  time.Now(),
		TTL:   24 * time.Hour,
	})
	_ = SavePoint(PointData{
		Tag:   "tag2",
		Value: json.RawMessage(`"正常"`),
		Time:  time.Now(),
		TTL:   24 * time.Hour,
	})

	// 2. 批量时间范围查询
	fmt.Println("\n=== 批量时间范围查询 ===")
	queries := []PointQuery{
		{
			Tag:       "tag1",
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
		},
		{
			Tag:       "tag2",
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
		},
	}
	res, _ := BatchQueryByTimeRange(queries)
	for pid, list := range res {
		fmt.Printf("点位 %s 共 %d 条\n", pid, len(list))
		for _, item := range list {
			fmt.Printf("  %s | %s\n", item.Time, string(item.Value))
		}
	}

	// 3. 批量查最新值
	fmt.Println("\n=== 批量查最新值 ===")
	lastRes, _ := BatchGetLastValue([]string{"tag1", "tag2"})
	for pid, item := range lastRes {
		fmt.Printf("%s 最新：%s | %s\n", pid, item.Time, string(item.Value))
	}
}

// // 每 10 秒记录一次 Badger 监控指标
// go func() {
//     ticker := time.NewTicker(10 * time.Second)
//     defer ticker.Stop()

//     for range ticker.C {
//         ms := db.Metrics()

//         log.Printf(`
// Badger 监控:
//   总写入数: %d
//   总读取数: %d
//   缓存命中: %d
//   ValueLog大小: %d MB
//   活跃Level0文件: %d
// `,
//             ms.PutCount(),        // 写入次数
//             ms.GetCount(),        // 读取次数
//             ms.BlockCacheHit(),   // 缓存命中
//             ms.VLogSize(), // 值日志大小 MB
//             ms.L0NumTables(),     // L0文件数
//         )
//     }
// }()
