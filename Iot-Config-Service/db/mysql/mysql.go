package mysql

import (
	"main/Init"

	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var ErrNoRows = sql.ErrNoRows
var DB *sql.DB

func Init_sql(dsn string) error {
	var err error
	// dsn := "用户名:密码@tcp(IP地址:端口)/数据库名?charset=utf8mb4&parseTime=True"
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	// 连接池配置：database/sql 自带池，但默认 MaxOpenConns 不限、MaxIdleConns 仅 2，
	// 不配置会打爆 MySQL max_connections，且空闲连接被立即关闭导致频繁重建握手。
	DB.SetMaxOpenConns(20)                  // 并发打开的最大连接数，防止打爆 MySQL
	DB.SetMaxIdleConns(10)                  // 保留的空闲连接数，避免频繁建连握手
	DB.SetConnMaxLifetime(time.Hour)        // 连接最长存活，小于 MySQL wait_timeout，防服务端单方断开后拿到坏连接
	DB.SetConnMaxIdleTime(10 * time.Minute) // 空闲超时回收，低负载时释放资源

	// 带超时的实际验证连接（sql.Open 不建连，Ping 才是真连接；无超时会永久阻塞启动）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = DB.PingContext(ctx)
	if err != nil {
		return err
	}

	CheckSqlStructure()
	return nil
}

func Example() *sql.DB {
	return DB
}

func init() {
	Config := Init.Config

	err := Init_sql(Config.MYSQL.Dsn)
	if err != nil {
		log.Print("ERROR ", err)
		panic(err)
	}
}
