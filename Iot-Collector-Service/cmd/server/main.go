package main

import (
	"collector/internal/config"
	_ "collector/internal/config"
	"collector/internal/handler"
	"collector/internal/infra/mqtt"
	"collector/internal/repo/influxdb"
	"collector/internal/repo/mysql"
	"collector/internal/repo/redis"
	"collector/internal/service/collector"
	"collector/internal/service/configsync"
	"collector/internal/service/driver"
	"collector/internal/service/internalmqtt"

	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"time"

	_ "github.com/icattlecoder/godaemon"
)

func app() {
	if err := mqtt.New(); err != nil {
		log.Printf("ERROR MQTT 初始化失败: %v", err)
	}

	// 内部 MQTT 功能（依赖 mqtt 全局连接，须在 mqtt.New 之后启动）
	if err := internalmqtt.Start(); err != nil {
		log.Printf("ERROR 内部MQTT功能初始化失败: %v", err)
	}

	if err := handler.Web(); err != nil {
		log.Printf("ERROR Web 服务启动失败: %v", err)
	}

	if err := collector.New(); err != nil {
		log.Printf("ERROR 数据点初始化失败: %v", err)
	}
	if config.Config.Influxdb.Enable {
		if err := influxdb.New(); err != nil {
			log.Printf("ERROR InfluxDB 初始化失败: %v", err)
		}

	}

	time.Sleep(200 * time.Millisecond)

	if err := driver.Start(collector.Collection_Publisher); err != nil {
		log.Printf("ERROR 驱动初始化失败: %v", err)
	}
}

// 优雅退出时间预算：总共最多等 shutdownTotal，单步最多等 shutdownStep，
// 避免某个设备/连接卡住就让进程再也退不掉（以前 systemd 只能靠 SIGKILL）。
const (
	shutdownTotal = 15 * time.Second
	shutdownStep  = 5 * time.Second
)

// shutdownOnce 保证收尾只跑一次：信号路径与 defer exit() 可能都会触发。
var shutdownOnce sync.Once

// gracefulExit 按"先断对外入口、再停采集推送、最后关存储连接"的顺序收尾，
// 每步失败或超时只记日志，不阻断后续步骤。
func gracefulExit() {
	shutdownOnce.Do(func() {
		start := time.Now()
		log.Print("INFO 开始优雅退出 =====================================")

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTotal)
		defer cancel()

		// 1. 先通知实时推送前端，让它拿到正常的 1001 关闭事件而非断链
		step(ctx, "WebSocket 推送", func() error {
			handler.WSCloseAll("服务正在关闭")
			return nil
		})
		// 2. 停止接口：不再接新连接，在途请求（写值/导入等）跑完再关
		step(ctx, "HTTP 接口", func() error { return handler.WebShutdown(ctx) })
		// 3. 停周期发布，避免往马上关闭的 MQTT 写数据
		step(ctx, "内部 MQTT 发布", func() error {
			internalmqtt.Stop()
			return nil
		})
		// 4. 停采集：关掉轮询协程并断开 PLC/Modbus 连接（设备侧看到的是一次正常断连）
		step(ctx, "驱动连接", func() error {
			driver.CloseAll(ctx)
			return nil
		})
		// 5. 向 broker 发正常 DISCONNECT
		step(ctx, "MQTT 连接", func() error {
			mqtt.CloseAll()
			return nil
		})
		// 6. 把时序库客户端缓冲区里未发送的数据先落盘
		if config.Config.Influxdb.Enable {
			step(ctx, "InfluxDB 缓冲", func() error { return closeInfluxOnce() })
		}
		// 7. 最后才关底层连接池（上面几步还可能用到 Redis/MySQL）
		step(ctx, "Redis 连接", func() error { return redis.Close() })
		step(ctx, "MySQL 连接", func() error { return mysql.Close() })

		log.Printf("INFO 优雅退出完成，用时 %s --------------------------------", time.Since(start).Round(time.Millisecond))
	})
}

// step 执行一个收尾动作：总预算已用完则跳过；单步再给一层超时；panic 降级为错误，
// 绝不让收尾把进程带倒（参考驱动管理器空指针那次崩溃）。
func step(ctx context.Context, name string, fn func() error) {
	if ctx.Err() != nil {
		log.Printf("WARN 退出超时，跳过[%s]", name)
		return
	}

	stepCtx, cancel := context.WithTimeout(ctx, shutdownStep)
	defer cancel()

	type result struct{ err error }
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("panic: %v", r)}
			}
		}()
		done <- result{err: fn()}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			log.Printf("WARN 退出步骤[%s]失败: %v", name, res.err)
			return
		}
		log.Printf("INFO 退出步骤[%s]已完成", name)
	case <-stepCtx.Done():
		log.Printf("WARN 退出步骤[%s]超时(%v)，放弃等待", name, shutdownStep)
	}
}

// closeInfluxOnce 只关闭一次 InfluxDB 客户端，让信号路径与 defer 兜底路径都能安全调用。
var influxCloseOnce sync.Once

func closeInfluxOnce() error {
	var err error
	influxCloseOnce.Do(func() { err = influxdb.Close() })
	return err
}

func exit() {
	// 收尾已统一放到 gracefulExit（内部已按依赖顺序包含 InfluxDB 落盘），
	// 且由 sync.Once 幂等保护：信号路径跑过后这里直接空转，不重复关连接。
	gracefulExit()
}

func main() {
	log.Print("INFO 程序开始 =======================================")
	// 关键：defer 执行顺序是"后进先出"，建议把日志defer放在最前面，确保最后执行
	defer log.Print("INFO 程序结束 ---------------------------------------")

	defer exit()

	// 启动前检查配置是否有更新，有更新则同步到 MySQL
	if config.Config.Config_Service.Enable {
		syncConfigOnStartup()
	}

	app()

	// ********** 关键2：提前初始化退出信号监听（app之前）**********
	// 创建带缓冲的信号通道（避免信号丢失）
	sigChan := make(chan os.Signal, 2)
	// 监听所有常见退出信号（覆盖更多场景）
	signal.Notify(
		sigChan,
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // kill 进程ID（非-9）
		syscall.SIGHUP,  // 关闭终端/容器窗口
		syscall.SIGQUIT, // Ctrl+\
	)

	sig := <-sigChan
	log.Printf("INFO 收到退出信号 %v，开始优雅退出（再按一次 Ctrl+C 强制退出）", sig)

	// 收尾期间再收到一次信号就立即强制退出，防止卡死在某个收尾步骤上脱不了身
	done := make(chan struct{})
	go func() {
		select {
		case s := <-sigChan:
			log.Printf("WARN 再次收到退出信号 %v，强制退出（未完成收尾，缓冲区数据可能丢失）", s)
			os.Exit(1)
		case <-done:
		}
	}()

	gracefulExit()
	close(done)
}

// syncConfigOnStartup 启动时检查配置服务是否有更新，有更新则同步配置到 MySQL
func syncConfigOnStartup() {

	configUpdateTime, err := configsync.IsConfigUpdated()
	if err != nil {
		log.Printf("WARN 配置更新检查失败: %v", err)
		return
	}
	if configUpdateTime.IsZero() {
		return
	}
	if err := configsync.SyncConfigFromService(configUpdateTime); err != nil {
		log.Printf("ERROR 配置同步失败: %v", err)
	}
}
