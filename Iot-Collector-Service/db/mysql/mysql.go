package mysql

import (
	"main/Init"

	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Init_sql(dsn string) error {
	var err error
	// dsn := "用户名:密码@tcp(IP地址:端口)/数据库名?charset=utf8mb4&parseTime=True"
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	// defer db.Close() // 确保关闭连接

	// 实际验证连接
	err = DB.Ping()
	if err != nil {
		return err
	}

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
