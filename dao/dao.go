package dao

import (
	"BruteHttp/config"
	"fmt"
	"github.com/jinzhu/gorm"
	"time"
)

var (
	Db  *gorm.DB
	err error
)

func init() {

	var err error
	Db, err = gorm.Open("mysql", config.Mysqldb)
	Db.LogMode(true)

	if err != nil {
		fmt.Println(err)
		return
	}

	Db.DB().SetMaxIdleConns(10)
	Db.DB().SetMaxOpenConns(100)
	Db.DB().SetConnMaxLifetime(time.Hour)
}
