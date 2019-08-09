package bootx

import (
	"github.com/gen-iot/std"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"sync"
)

type DBConfig struct {
	DatabaseType     string `json:"databaseType" validate:"oneof=mysql postgres sqlite3 mssql"`
	ConnStr          string `json:"connStr" validate:"required"`
	ShowSql          bool   `json:"showSql"`
	MaxIdleConnCount int    `json:"maxIdleConn" validate:"min=0,max=1000"`
	MaxOpenConnCount int    `json:"maxOpenConn" validate:"min=0,max=1000"`
}

var DBDefaultConfig = DBConfig{
	DatabaseType:     "",
	ConnStr:          "",
	ShowSql:          false,
	MaxIdleConnCount: 10,
	MaxOpenConnCount: 100,
}

type DataBase struct {
	*gorm.DB
}

var dbOnce = sync.Once{}
var gDb *DataBase = nil
var dbConfig *DBConfig = nil

func DB() *DataBase {
	std.Assert(dbConfig != nil, "database not config yet")
	dbOnce.Do(func() {
		err := std.ValidateStruct(dbConfig)
		std.AssertError(err, "数据库配置不正确")
		logger.Printf("database db(%s) init ...", dbConfig.ConnStr)
		db, err := gorm.Open(dbConfig.DatabaseType, dbConfig.ConnStr)
		std.AssertError(err, "database open failed")
		if dbConfig.ShowSql {
			//use gorm default logger
			//gDb.SetLogger(log.DEBUG)
			db.LogMode(true)
		}
		//dbConfig connection pool
		db.DB().SetMaxIdleConns(dbConfig.MaxIdleConnCount)
		db.DB().SetMaxOpenConns(dbConfig.MaxOpenConnCount)
		gDb = &DataBase{db}
	})
	return gDb
}

func (this *DataBase) Tx(txFunc func(*gorm.DB) error) (err error) {
	tx := this.Begin()
	defer tx.Rollback()
	if err = tx.Error; err != nil {
		return
	}
	if err = txFunc(tx); err != nil {
		return
	}
	return tx.Commit().Error
}

func dbInit(dbType string, connStr string) {
	dbInitWithConfig(&DBConfig{
		DatabaseType:     dbType,
		ConnStr:          connStr,
		ShowSql:          DBDefaultConfig.ShowSql,
		MaxIdleConnCount: DBDefaultConfig.MaxIdleConnCount,
		MaxOpenConnCount: DBDefaultConfig.MaxOpenConnCount,
	})
}

func dbInitWithConfig(conf *DBConfig) {
	dbConfig = conf
	DB()
}

func dbCleanup() {
	logger.Println("database cleanup ...")
	if err := gDb.Close(); err != nil {
		logger.Printf("%s error occurred while database close : %s ...", logTag, err)
	}
}
