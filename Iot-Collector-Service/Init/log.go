// package Init

// import (
// 	"bytes"
// 	"fmt"
// 	"io"
// 	"path/filepath"
// 	"regexp"
// 	"sort"
// 	"strings"
// 	"sync"
// 	"time"

// 	"log"
// 	"os"
// )

// /*
// ***************初始化配置文件*****************
//  */

// // 缓存条目结构
// type cacheEntry struct {
// 	timestamp   time.Time
// 	count       int    // 重复次数
// 	lastMessage string // 原始消息（用于调试）
// }

// // 缓存统计信息
// type CacheStats struct {
// 	TotalEntries int
// 	TotalHits    int
// 	Size         int
// }

// // 精确去重日志记录器（带缓存功能）
// type DedupDailyLogger struct {
// 	mu          sync.Mutex
// 	logDir      string
// 	prefix      string
// 	currentFile *os.File
// 	currentDate string
// 	cooldown    time.Duration // 11分钟去重时间
// 	patternMap  map[string]*regexp.Regexp

// 	// 缓存相关字段
// 	cache         map[string]cacheEntry // 日志消息缓存
// 	cacheMutex    sync.RWMutex          // 缓存读写锁
// 	cacheTTL      time.Duration         // 缓存存活时间（11分钟）
// 	cleanupTicker *time.Ticker          // 定期清理缓存
// 	stopChan      chan struct{}         // 停止信号
// 	stats         CacheStats            // 缓存统计
// 	statsMutex    sync.RWMutex          // 统计锁
// }

// // NewDedupDailyLogger 创建新的日志记录器
// func NewDedupDailyLogger(logDir, prefix string, cacheTTL time.Duration) *DedupDailyLogger {
// 	os.MkdirAll(logDir, 0755)

// 	// 预编译常见日志模式的正则表达式
// 	patterns := map[string]*regexp.Regexp{
// 		"timestamp":  regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d+`), // 匹配完整时间戳
// 		"filepath":   regexp.MustCompile(`/[^ ]+\.go:\d+`),                           // 匹配文件路径
// 		"ip_address": regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`),       // IP地址
// 		"port":       regexp.MustCompile(`:\d{1,5}`),                                 // 端口号
// 		"driver_id":  regexp.MustCompile(`驱动id:\d+`),                                 // 驱动ID
// 		"wait_time":  regexp.MustCompile(`等待:\d+ms`),                                 // 等待时间
// 		"any_number": regexp.MustCompile(`\b\d+\b`),                                  // 任意数字
// 	}

// 	logger := &DedupDailyLogger{
// 		logDir:     logDir,
// 		prefix:     prefix,
// 		cooldown:   cacheTTL, // 11分钟去重
// 		patternMap: patterns,
// 		cache:      make(map[string]cacheEntry),
// 		cacheTTL:   cacheTTL, // 缓存保持11分钟
// 		stopChan:   make(chan struct{}),
// 		stats:      CacheStats{},
// 	}

// 	// 启动缓存清理goroutine
// 	logger.startCacheCleanup()

// 	return logger
// }

// // startCacheCleanup 启动缓存清理任务
// func (d *DedupDailyLogger) startCacheCleanup() {
// 	// 每5分钟清理一次过期缓存
// 	d.cleanupTicker = time.NewTicker(5 * time.Minute)
// 	go func() {
// 		for {
// 			select {
// 			case <-d.cleanupTicker.C:
// 				d.cleanupExpiredCache()
// 			case <-d.stopChan:
// 				return
// 			}
// 		}
// 	}()
// }

// // cleanupExpiredCache 清理过期的缓存条目
// func (d *DedupDailyLogger) cleanupExpiredCache() {
// 	d.cacheMutex.Lock()
// 	defer d.cacheMutex.Unlock()

// 	now := time.Now()
// 	removedCount := 0
// 	for key, entry := range d.cache {
// 		if now.Sub(entry.timestamp) > d.cacheTTL {
// 			delete(d.cache, key)
// 			removedCount++
// 		}
// 	}

// 	// 如果清理了很多条目，可以尝试缩减map大小
// 	if removedCount > 100 && len(d.cache) < 1000 {
// 		newCache := make(map[string]cacheEntry, len(d.cache))
// 		for k, v := range d.cache {
// 			newCache[k] = v
// 		}
// 		d.cache = newCache
// 	}
// }

// // getFilename 获取当前日志文件名
// func (d *DedupDailyLogger) getFilename(date string) string {
// 	return filepath.Join(d.logDir, fmt.Sprintf("%s%s.log", d.prefix, date))
// }

// // rotate 轮转日志文件
// func (d *DedupDailyLogger) rotate() error {
// 	today := time.Now().Format("2006-01-02")

// 	if d.currentFile != nil && d.currentDate == today {
// 		return nil
// 	}

// 	if d.currentFile != nil {
// 		d.currentFile.Close()
// 	}

// 	filename := d.getFilename(today)
// 	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
// 	if err != nil {
// 		return err
// 	}

// 	d.currentFile = f
// 	d.currentDate = today
// 	return nil
// }

// // normalizeMessage 标准化日志消息 - 针对提供的日志格式优化
// func (d *DedupDailyLogger) normalizeMessage(msg string) string {
// 	// 按顺序应用所有正则表达式替换
// 	for _, pattern := range []string{
// 		"timestamp",  // 时间戳
// 		"filepath",   // 文件路径
// 		"ip_address", // IP地址
// 		"port",       // 端口号
// 		"driver_id",  // 驱动ID
// 		"wait_time",  // 等待时间
// 		"any_number", // 任意数字
// 	} {
// 		if re, ok := d.patternMap[pattern]; ok {
// 			msg = re.ReplaceAllString(msg, "["+strings.ToUpper(pattern)+"]")
// 		}
// 	}

// 	// 移除多余空格
// 	msg = strings.Join(strings.Fields(msg), " ")

// 	// 特殊处理常见模式
// 	msg = strings.ReplaceAll(msg, "驱动id:[DRIVER_ID]", "驱动id:[ID]")
// 	msg = strings.ReplaceAll(msg, "等待:[WAIT_TIME]", "等待:[TIME]ms")

// 	return msg
// }

// // isDuplicate 检查11分钟内是否有重复日志
// func (d *DedupDailyLogger) isDuplicate(msg string) bool {
// 	// 标准化消息
// 	normalized := d.normalizeMessage(msg)
// 	now := time.Now()

// 	d.cacheMutex.Lock()
// 	defer d.cacheMutex.Unlock()

// 	// 检查缓存中是否存在该消息
// 	if entry, exists := d.cache[normalized]; exists {
// 		// 检查是否在11分钟内
// 		if now.Sub(entry.timestamp) <= d.cooldown {
// 			// 更新缓存条目的时间戳和计数
// 			d.cache[normalized] = cacheEntry{
// 				timestamp:   now,
// 				count:       entry.count + 1,
// 				lastMessage: msg,
// 			}

// 			// 更新统计
// 			d.statsMutex.Lock()
// 			d.stats.TotalHits++
// 			d.statsMutex.Unlock()

// 			return true
// 		}
// 		// 如果超过11分钟，删除旧条目
// 		delete(d.cache, normalized)
// 	}

// 	// 添加到缓存
// 	d.cache[normalized] = cacheEntry{
// 		timestamp:   now,
// 		count:       1,
// 		lastMessage: msg,
// 	}

// 	// 更新统计
// 	d.statsMutex.Lock()
// 	d.stats.TotalEntries = len(d.cache)
// 	d.stats.Size = len(normalized) + 16 // 估算内存占用
// 	d.statsMutex.Unlock()

// 	return false
// }

// // Write 实现io.Writer接口
// func (d *DedupDailyLogger) Write(p []byte) (n int, err error) {
// 	d.mu.Lock()
// 	defer d.mu.Unlock()

// 	// 确保日志文件已轮转
// 	if err := d.rotate(); err != nil {
// 		return 0, err
// 	}

// 	msg := string(bytes.TrimSpace(p))

// 	// 检查11分钟内是否有重复日志
// 	if d.isDuplicate(msg) {
// 		return len(p), nil
// 	}

// 	// 记录新日志
// 	_, err = d.currentFile.Write(p)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return len(p), nil
// }

// // GetCacheStats 获取缓存统计信息
// func (d *DedupDailyLogger) GetCacheStats() CacheStats {
// 	d.statsMutex.RLock()
// 	defer d.statsMutex.RUnlock()

// 	// 从缓存获取当前条目数
// 	d.cacheMutex.RLock()
// 	currentEntries := len(d.cache)
// 	d.cacheMutex.RUnlock()

// 	stats := d.stats
// 	stats.TotalEntries = currentEntries
// 	return stats
// }

// // GetTopDuplicates 获取重复次数最多的日志（用于调试）
// func (d *DedupDailyLogger) GetTopDuplicates(limit int) []struct {
// 	Message string
// 	Count   int
// 	Age     time.Duration
// } {
// 	d.cacheMutex.RLock()
// 	defer d.cacheMutex.RUnlock()

// 	type entry struct {
// 		msg   string
// 		count int
// 		age   time.Duration
// 	}

// 	entries := make([]entry, 0, len(d.cache))
// 	now := time.Now()

// 	for key, cacheEntry := range d.cache {
// 		entries = append(entries, entry{
// 			msg:   key,
// 			count: cacheEntry.count,
// 			age:   now.Sub(cacheEntry.timestamp),
// 		})
// 	}

// 	// 按重复次数排序
// 	sort.Slice(entries, func(i, j int) bool {
// 		return entries[i].count > entries[j].count
// 	})

// 	// 限制返回数量
// 	if limit > len(entries) {
// 		limit = len(entries)
// 	}

// 	result := make([]struct {
// 		Message string
// 		Count   int
// 		Age     time.Duration
// 	}, limit)

// 	for i := 0; i < limit; i++ {
// 		result[i] = struct {
// 			Message string
// 			Count   int
// 			Age     time.Duration
// 		}{
// 			Message: entries[i].msg,
// 			Count:   entries[i].count,
// 			Age:     entries[i].age,
// 		}
// 	}

// 	return result
// }

// // CleanupOldLogs 清理旧日志文件
// func (d *DedupDailyLogger) CleanupOldLogs(retentionDays int) {
// 	files, err := os.ReadDir(d.logDir)
// 	if err != nil {
// 		return
// 	}

// 	cutoffTime := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)

// 	for _, file := range files {
// 		if file.IsDir() {
// 			continue
// 		}

// 		filename := file.Name()
// 		if !strings.HasPrefix(filename, d.prefix) || !strings.HasSuffix(filename, ".log") {
// 			continue
// 		}

// 		// 尝试从文件名解析日期
// 		dateStr := strings.TrimPrefix(filename, d.prefix)
// 		dateStr = strings.TrimSuffix(dateStr, ".log")

// 		// 尝试多种日期格式
// 		var logDate time.Time
// 		for _, layout := range []string{"2006-01-02", "20060102", "2006_01_02"} {
// 			if t, err := time.Parse(layout, dateStr); err == nil {
// 				logDate = t
// 				break
// 			}
// 		}

// 		// 如果解析失败，使用文件修改时间
// 		if logDate.IsZero() {
// 			if info, err := file.Info(); err == nil {
// 				logDate = info.ModTime()
// 			} else {
// 				continue
// 			}
// 		}

// 		if logDate.Before(cutoffTime) {
// 			filePath := filepath.Join(d.logDir, filename)
// 			os.Remove(filePath)
// 		}
// 	}
// }

// // CleanCache 手动清理缓存
// func (d *DedupDailyLogger) CleanCache() {
// 	d.cacheMutex.Lock()
// 	d.cache = make(map[string]cacheEntry)
// 	d.cacheMutex.Unlock()

// 	d.statsMutex.Lock()
// 	d.stats = CacheStats{}
// 	d.statsMutex.Unlock()
// }

// // Close 关闭日志记录器，清理资源
// func (d *DedupDailyLogger) Close() error {
// 	// 停止缓存清理goroutine
// 	if d.cleanupTicker != nil {
// 		d.cleanupTicker.Stop()
// 		close(d.stopChan)
// 	}

// 	// 关闭日志文件
// 	if d.currentFile != nil {
// 		return d.currentFile.Close()
// 	}
// 	return nil
// }

// // // 日志初始化
// // func init() {
// // 	logger := NewDedupDailyLogger("./logs", "")
// // 	log.SetOutput(io.MultiWriter(os.Stdout, logger))
// // 	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

// // 	// 可选：注册一个清理函数，程序退出时关闭logger
// // 	// runtime.SetFinalizer(logger, (*DedupDailyLogger).Close)
// // }

// // 日志初始化
// func init_log() {

// 	if !Config.LOG.Enable {
// 		return
// 	}

// 	// 将字符串标志转换为log.Lflags
// 	var flags int
// 	for _, flag := range strings.Fields(Config.LOG.Flags) {
// 		switch flag {
// 		case "date", "Ldate":
// 			flags |= log.Ldate
// 		case "time", "Ltime":
// 			flags |= log.Ltime
// 		case "microseconds", "Lmicroseconds":
// 			flags |= log.Lmicroseconds
// 		case "longfile", "Llongfile":
// 			flags |= log.Llongfile
// 		case "shortfile", "Lshortfile":
// 			flags |= log.Lshortfile
// 		case "UTC", "LUTC":
// 			flags |= log.LUTC
// 		case "msgprefix", "Lmsgprefix":
// 			flags |= log.Lmsgprefix
// 		}
// 	}

// 	logger := NewDedupDailyLogger(Config.LOG.Path, "", time.Duration(Config.LOG.CacheTTL)*time.Second)
// 	log.SetOutput(io.MultiWriter(os.Stdout, logger))
// 	log.SetFlags(flags)

// 	// defer runtime.SetFinalizer(logger, (*DedupDailyLogger).Close)
// }

package Init

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"log"
	"os"
)

var Log__Add2 func(User_Id uint, Type string, Message string) (Id uint, err error)

/*
***************初始化配置文件*****************
 */

// 缓存条目结构
type cacheEntry struct {
	timestamp   time.Time
	count       int    // 重复次数
	lastMessage string // 原始消息（用于调试）
}

// 缓存统计信息
type CacheStats struct {
	TotalEntries int
	TotalHits    int
	Size         int
}

// 回调函数类型
type LogCallback func(timestamp time.Time, microseconds, longfile, message string) error

// 精确去重日志记录器（带回调函数和缓存功能）
type DedupDailyLogger struct {
	mu          sync.Mutex
	logDir      string
	prefix      string
	currentFile *os.File
	currentDate string
	cooldown    time.Duration // 11分钟去重时间
	patternMap  map[string]*regexp.Regexp

	// 缓存相关字段
	cache         map[string]cacheEntry
	cacheMutex    sync.RWMutex
	cacheTTL      time.Duration
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
	stats         CacheStats
	statsMutex    sync.RWMutex

	// 回调函数
	callback LogCallback
}

// NewDedupDailyLogger 创建新的日志记录器
func NewDedupDailyLogger(logDir, prefix string, cacheTTL time.Duration) *DedupDailyLogger {
	os.MkdirAll(logDir, 0755)

	// 预编译常见日志模式的正则表达式
	patterns := map[string]*regexp.Regexp{
		"timestamp":  regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d+`),
		"filepath":   regexp.MustCompile(`/[^ ]+\.go:\d+`),
		"ip_address": regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`),
		"port":       regexp.MustCompile(`:\d{1,5}`),
		"driver_id":  regexp.MustCompile(`驱动id:\d+`),
		"wait_time":  regexp.MustCompile(`等待:\d+ms`),
		"any_number": regexp.MustCompile(`\b\d+\b`),
	}

	logger := &DedupDailyLogger{
		logDir:     logDir,
		prefix:     prefix,
		cooldown:   cacheTTL,
		patternMap: patterns,
		cache:      make(map[string]cacheEntry),
		cacheTTL:   cacheTTL,
		stopChan:   make(chan struct{}),
		stats:      CacheStats{},
	}

	// 启动缓存清理goroutine
	logger.startCacheCleanup()

	return logger
}

// SetCallback 设置回调函数
func (d *DedupDailyLogger) SetCallback(callback LogCallback) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.callback = callback
}

// startCacheCleanup 启动缓存清理任务
func (d *DedupDailyLogger) startCacheCleanup() {
	// 每5分钟清理一次过期缓存
	d.cleanupTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-d.cleanupTicker.C:
				d.cleanupExpiredCache()
			case <-d.stopChan:
				return
			}
		}
	}()
}

// cleanupExpiredCache 清理过期的缓存条目
func (d *DedupDailyLogger) cleanupExpiredCache() {
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	now := time.Now()
	removedCount := 0
	for key, entry := range d.cache {
		if now.Sub(entry.timestamp) > d.cacheTTL {
			delete(d.cache, key)
			removedCount++
		}
	}

	// 如果清理了很多条目，可以尝试缩减map大小
	if removedCount > 100 && len(d.cache) < 1000 {
		newCache := make(map[string]cacheEntry, len(d.cache))
		for k, v := range d.cache {
			newCache[k] = v
		}
		d.cache = newCache
	}
}

// getFilename 获取当前日志文件名
func (d *DedupDailyLogger) getFilename(date string) string {
	return filepath.Join(d.logDir, fmt.Sprintf("%s%s.log", d.prefix, date))
}

// rotate 轮转日志文件
func (d *DedupDailyLogger) rotate() error {
	today := time.Now().Format("2006-01-02")

	if d.currentFile != nil && d.currentDate == today {
		return nil
	}

	if d.currentFile != nil {
		d.currentFile.Close()
	}

	filename := d.getFilename(today)
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	d.currentFile = f
	d.currentDate = today
	return nil
}

// normalizeMessage 标准化日志消息
func (d *DedupDailyLogger) normalizeMessage(msg string) string {
	// 按顺序应用所有正则表达式替换
	for _, pattern := range []string{
		"timestamp",  // 时间戳
		"filepath",   // 文件路径
		"ip_address", // IP地址
		"port",       // 端口号
		"driver_id",  // 驱动ID
		"wait_time",  // 等待时间
		"any_number", // 任意数字
	} {
		if re, ok := d.patternMap[pattern]; ok {
			msg = re.ReplaceAllString(msg, "["+strings.ToUpper(pattern)+"]")
		}
	}

	// 移除多余空格
	msg = strings.Join(strings.Fields(msg), " ")

	// 特殊处理常见模式
	msg = strings.ReplaceAll(msg, "驱动id:[DRIVER_ID]", "驱动id:[ID]")
	msg = strings.ReplaceAll(msg, "等待:[WAIT_TIME]", "等待:[TIME]ms")

	return msg
}

// isDuplicate 检查11分钟内是否有重复日志
func (d *DedupDailyLogger) isDuplicate(msg string) bool {
	// 标准化消息
	normalized := d.normalizeMessage(msg)
	now := time.Now()

	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	// 检查缓存中是否存在该消息
	if entry, exists := d.cache[normalized]; exists {
		// 检查是否在11分钟内
		if now.Sub(entry.timestamp) <= d.cooldown {
			// 更新缓存条目的时间戳和计数
			d.cache[normalized] = cacheEntry{
				timestamp:   now,
				count:       entry.count + 1,
				lastMessage: msg,
			}

			// 更新统计
			d.statsMutex.Lock()
			d.stats.TotalHits++
			d.statsMutex.Unlock()

			return true
		}
		// 如果超过11分钟，删除旧条目
		delete(d.cache, normalized)
	}

	// 添加到缓存
	d.cache[normalized] = cacheEntry{
		timestamp:   now,
		count:       1,
		lastMessage: msg,
	}

	// 更新统计
	d.statsMutex.Lock()
	d.stats.TotalEntries = len(d.cache)
	d.stats.Size = len(normalized) + 16
	d.statsMutex.Unlock()

	return false
}

// parseLogMessage 解析日志消息，提取时间戳、微秒、文件路径等信息
func (d *DedupDailyLogger) parseLogMessage(msg string) (time.Time, string, string, string) {
	// 默认值
	timestamp := time.Now()
	microseconds := ""
	longfile := ""
	message := msg

	// 尝试解析标准日志格式: 2026/01/30 23:23:19.870683 /path/to/file.go:123: message
	parts := strings.SplitN(msg, " ", 4)
	if len(parts) >= 3 {
		// 解析日期和时间部分
		datetimeStr := parts[0] + " " + parts[1]
		if t, err := time.Parse("2006/01/02 15:04:05.999999", datetimeStr); err == nil {
			timestamp = t
			microseconds = parts[1][strings.Index(parts[1], ".")+1:]
		}

		// 提取文件路径
		longfile = parts[2]
		if len(parts) >= 4 {
			message = parts[3]
		}
	}

	return timestamp, microseconds, longfile, message
}

// Write 实现io.Writer接口 - 核心修改点
func (d *DedupDailyLogger) Write(p []byte) (n int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	msg := string(bytes.TrimSpace(p))

	// 检查11分钟内是否有重复日志
	if d.isDuplicate(msg) {
		return len(p), nil
	}

	// 解析日志消息
	timestamp, microseconds, longfile, message := d.parseLogMessage(msg)

	// 如果有回调函数，先执行回调
	if d.callback != nil {
		err := d.callback(timestamp, microseconds, longfile, message)
		if err == nil {
			// 回调执行成功，不写入文件，直接返回
			return len(p), nil
		}
		// 回调执行失败，继续写入文件
	}

	// 确保日志文件已轮转
	if err := d.rotate(); err != nil {
		return 0, err
	}

	// 记录新日志到文件
	_, err = d.currentFile.Write(p)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// GetCacheStats 获取缓存统计信息
func (d *DedupDailyLogger) GetCacheStats() CacheStats {
	d.statsMutex.RLock()
	defer d.statsMutex.RUnlock()

	// 从缓存获取当前条目数
	d.cacheMutex.RLock()
	currentEntries := len(d.cache)
	d.cacheMutex.RUnlock()

	stats := d.stats
	stats.TotalEntries = currentEntries
	return stats
}

// GetTopDuplicates 获取重复次数最多的日志（用于调试）
func (d *DedupDailyLogger) GetTopDuplicates(limit int) []struct {
	Message string
	Count   int
	Age     time.Duration
} {
	d.cacheMutex.RLock()
	defer d.cacheMutex.RUnlock()

	type entry struct {
		msg   string
		count int
		age   time.Duration
	}

	entries := make([]entry, 0, len(d.cache))
	now := time.Now()

	for key, cacheEntry := range d.cache {
		entries = append(entries, entry{
			msg:   key,
			count: cacheEntry.count,
			age:   now.Sub(cacheEntry.timestamp),
		})
	}

	// 按重复次数排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	// 限制返回数量
	if limit > len(entries) {
		limit = len(entries)
	}

	result := make([]struct {
		Message string
		Count   int
		Age     time.Duration
	}, limit)

	for i := 0; i < limit; i++ {
		result[i] = struct {
			Message string
			Count   int
			Age     time.Duration
		}{
			Message: entries[i].msg,
			Count:   entries[i].count,
			Age:     entries[i].age,
		}
	}

	return result
}

// CleanupOldLogs 清理旧日志文件
func (d *DedupDailyLogger) CleanupOldLogs(retentionDays int) {
	files, err := os.ReadDir(d.logDir)
	if err != nil {
		return
	}

	cutoffTime := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasPrefix(filename, d.prefix) || !strings.HasSuffix(filename, ".log") {
			continue
		}

		// 尝试从文件名解析日期
		dateStr := strings.TrimPrefix(filename, d.prefix)
		dateStr = strings.TrimSuffix(dateStr, ".log")

		// 尝试多种日期格式
		var logDate time.Time
		for _, layout := range []string{"2006-01-02", "20060102", "2006_01_02"} {
			if t, err := time.Parse(layout, dateStr); err == nil {
				logDate = t
				break
			}
		}

		// 如果解析失败，使用文件修改时间
		if logDate.IsZero() {
			if info, err := file.Info(); err == nil {
				logDate = info.ModTime()
			} else {
				continue
			}
		}

		if logDate.Before(cutoffTime) {
			filePath := filepath.Join(d.logDir, filename)
			os.Remove(filePath)
		}
	}
}

// CleanCache 手动清理缓存
func (d *DedupDailyLogger) CleanCache() {
	d.cacheMutex.Lock()
	d.cache = make(map[string]cacheEntry)
	d.cacheMutex.Unlock()

	d.statsMutex.Lock()
	d.stats = CacheStats{}
	d.statsMutex.Unlock()
}

// Close 关闭日志记录器，清理资源
func (d *DedupDailyLogger) Close() error {
	// 停止缓存清理goroutine
	if d.cleanupTicker != nil {
		d.cleanupTicker.Stop()
		close(d.stopChan)
	}

	// 关闭日志文件
	if d.currentFile != nil {
		return d.currentFile.Close()
	}
	return nil
}

// 日志初始化
func init_log() {
	if !Config.LOG.Enable {
		return
	}

	// 将字符串标志转换为log.Lflags
	var flags int
	for _, flag := range strings.Fields(Config.LOG.Flags) {
		switch flag {
		case "date", "Ldate":
			flags |= log.Ldate
		case "time", "Ltime":
			flags |= log.Ltime
		case "microseconds", "Lmicroseconds":
			flags |= log.Lmicroseconds
		case "longfile", "Llongfile":
			flags |= log.Llongfile
		case "shortfile", "Lshortfile":
			flags |= log.Lshortfile
		case "UTC", "LUTC":
			flags |= log.LUTC
		case "msgprefix", "Lmsgprefix":
			flags |= log.Lmsgprefix
		}
	}

	logger := NewDedupDailyLogger(Config.LOG.Path, "", time.Duration(Config.LOG.CacheTTL)*time.Second)

	// 设置回调函数

	// logger.SetCallback(func(timestamp time.Time, microseconds, longfile, message string) error {
	// fmt.Print(
	// 	"timestamp ", timestamp,
	// 	"\nmicroseconds ", microseconds,
	// 	"\nlongfile ", longfile,
	// 	"\nmessage ", message,
	// 	"++++++++++++++\n")

	// Log__Add2(0, "APP_LOG", longfile+message)
	// db_mysql.Log__Add(db_mysql.Log__table_type{})
	// 这里实现你的回调逻辑，比如保存到MySQL、发送到消息队列等

	// 示例：保存到MySQL
	// err := saveLogToMySQL(timestamp, microseconds, longfile, message)
	// if err != nil {
	//     return err // 返回错误会触发写入文件
	// }

	// 示例：发送到消息队列
	// err := sendToMessageQueue(timestamp, microseconds, longfile, message)
	// if err != nil {
	//     return err
	// }

	// 如果执行成功，返回nil，日志不会写入文件
	// 如果执行失败，返回error，日志会写入文件作为备份
	// return nil
	// })

	// 设置输出
	log.SetOutput(io.MultiWriter(os.Stdout, logger))
	log.SetFlags(flags)
}
