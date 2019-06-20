package mysql

import (
	"github.com/jinzhu/gorm"
)

var mysqlDb *gorm.DB

func init() {
	db, err := gorm.Open("mysql", "root:db_password@tcp(db_host)/test?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(15)
	db.SingularTable(true)
	mysqlDb = db
}

func GetDb() *gorm.DB {
	return mysqlDb
}

func mysqlDestroy() {
	mysqlDb.Close()
	mysqlDb = nil
}
