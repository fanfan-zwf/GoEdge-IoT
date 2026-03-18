package main

import (
	"log"
	"main/db/db_point"
	"main/db/mysql"
	"main/db/redis"
	"main/web"

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

	time.Sleep(200 * time.Millisecond)

	return
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
