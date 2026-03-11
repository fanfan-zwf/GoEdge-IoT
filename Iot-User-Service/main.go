package main

import (
	"main/db/mysql"
	"main/db/redis"
	"main/web"

	"time"

	_ "github.com/icattlecoder/godaemon"
)

func app() {

	// Rinit()
	// 注入到其他包
	go web.Web()

	time.Sleep(200 * time.Millisecond)

}

func exit() {
	mysql.DB.Close()
	redis.Rdb.Close()

}

func main() {
	defer exit()
	app()

	select {}
}
