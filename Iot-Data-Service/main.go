package main

import (
	"log"
	"main/IO/Rec_Yun_Cloud"
	"main/db/db_point"
	"main/db/influxdb"
	"main/db/mysql"
	"main/db/redis"
	"main/web"
	"os"
	"os/signal"
	"syscall"

	"time"

	_ "github.com/icattlecoder/godaemon"
)

func app() (err error) {

	// Rinit()
	// 注入到其他包
	err = web.Web()
	if err != nil {
		log.Panic(err.Error())
	}

	err = db_point.New()
	if err != nil {
		log.Panic(err.Error())
	}

	err = influxdb.New()
	if err != nil {
		log.Panic(err.Error())
	}

	err = Rec_Yun_Cloud.New()
	if err != nil {
		log.Panic(err.Error())
	}

	time.Sleep(200 * time.Millisecond)

	return
}

func exit() {
	mysql.DB.Close()
	redis.Rdb.Close()

}

func main() {
	log.Print("INFO 程序开始 =======================================")
	// 关键：defer 执行顺序是“后进先出”，建议把日志defer放在最前面，确保最后执行
	defer log.Print("INFO 程序结束 ---------------------------------------")

	defer exit()
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

	_ = <-sigChan
}
