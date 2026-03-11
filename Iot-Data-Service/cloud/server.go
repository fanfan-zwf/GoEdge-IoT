package cloud

import (
// _ "main/db/mysql"
// _ "main/db/redis"
)

// var Config singleton.Config_type

// func init() {
// 	/*
// 	 ***************初始化日志文件*****************
// 	 */

// 	logFile, err := os.OpenFile("../run.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
// 	if err != nil {
// 		fmt.Println("open log file failed, err:", err)
// 		return
// 	}
// 	log.SetOutput(logFile)
// 	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

// 	/*
// 	 ***************初始化配置文件*****************
// 	 */
// 	data, err := ioutil.ReadFile("../config/config.yaml")
// 	if err != nil {
// 		log.Println("ERR", "读取配置文件错误", err)
// 		return
// 	}

// 	err = yaml.Unmarshal(data, &Config)
// 	if err != nil {
// 		log.Println("ERR", "写入结构体错误", err)
// 		return
// 	}
// 	singleton.GetInstance().Config = Config

// }

// // 初始化数据库
// func Rinit() {
// 	err := mysql.Init_sql(
// 		Config.MYSQL.Ip,
// 		Config.MYSQL.Post,
// 		Config.MYSQL.User,
// 		Config.MYSQL.Passwd,
// 		Config.MYSQL.Database,
// 		Config.MYSQL.Charset,
// 	)
// 	if err != nil {
// 		log.Print(err)
// 		panic(err)
// 	}
// 	redis.Init_rdb(
// 		fmt.Sprintf("%s:%d", Config.REDIS.Ip, Config.REDIS.Post),
// 		Config.REDIS.Passwd,
// 		int(Config.REDIS.Database),
// 	)
// }

// 程序启动
func Server() {
	// instance := singleton.GetInstance()
	// // Rinit()
	// // 注入到其他包
	// go api.Api()

	// time.Sleep(200 * time.Millisecond)
	// if instance.Mysql == nil {
	// 	panic("mysql实例:nil")
	// }
	// if instance.Redis == nil {
	// 	panic("redis实例:nil")
	// }
	// fmt.Print("\nok-----------\n")

	// go Modbus_Tcp.Run()
}
