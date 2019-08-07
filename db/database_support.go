package db

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"sync"
)

const (
	logTag = "[DataBase]"
)

type Config struct {
	DatabaseType     string `json:"databaseType" validate:"oneof=mysql postgres sqlite3 mssql"`
	ConnStr          string `json:"connStr" validate:"required"`
	ShowSql          bool   `json:"showSql"`
	MaxIdleConnCount int    `json:"maxIdleConn" validate:"min=0,max=1000"`
	MaxOpenConnCount int    `json:"maxOpenConn" validate:"min=0,max=1000"`
}

var DefaultConfig = Config{
	DatabaseType:     "",
	ConnStr:          "",
	ShowSql:          false,
	MaxIdleConnCount: 10,
	MaxOpenConnCount: 100,
}

type DataBase struct {
	*gorm.DB
}

var once = sync.Once{}
var gDb *DataBase = nil

var config = DefaultConfig

func GetDB() *DataBase {
	once.Do(func() {
		err := std.ValidateStruct(config)
		std.AssertError(err, "数据库配置不正确")
		db, err := gorm.Open(config.DatabaseType, config.ConnStr)
		std.AssertError(err, "database open failed")
		if config.ShowSql {
			//use gorm default logger
			//gDb.SetLogger(log.DEBUG)
			db.LogMode(true)
		}
		//config connection pool
		db.DB().SetMaxIdleConns(config.MaxIdleConnCount)
		db.DB().SetMaxOpenConns(config.MaxOpenConnCount)
		gDb = &DataBase{db}
	})
	return gDb
}

func Tx(txFunc func(*gorm.DB) error) (err error) {
	tx := GetDB().Begin()
	defer tx.Rollback()
	if err = tx.Error; err != nil {
		return
	}
	if err = txFunc(tx); err != nil {
		return
	}
	return tx.Commit().Error
}

func Init(dbType string, connStr string) {
	InitWithConfig(Config{
		DatabaseType:     dbType,
		ConnStr:          connStr,
		ShowSql:          DefaultConfig.ShowSql,
		MaxIdleConnCount: DefaultConfig.MaxIdleConnCount,
		MaxOpenConnCount: DefaultConfig.MaxOpenConnCount,
	})
}

func InitWithConfig(conf Config) {
	config = conf
	fmt.Printf("%s db(%s) init ...", logTag, config.ConnStr)
	GetDB()
}

func Cleanup() {
	fmt.Printf("%s cleanup ...", logTag)
	if err := gDb.Close(); err != nil {
		fmt.Printf("%s error occurred while database close : %s ...", logTag, err)
	}
}
