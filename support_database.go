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
	DatabaseType     string `yaml:"databaseType" json:"databaseType" validate:"oneof=mysql postgres sqlite3 mssql"`
	ConnStr          string `yaml:"connStr" json:"connStr" validate:"required"`
	ShowSql          bool   `yaml:"showSql" json:"showSql"`
	MaxIdleConnCount int    `yaml:"maxIdleConn" json:"maxIdleConn" validate:"min=0,max=1000"`
	MaxOpenConnCount int    `yaml:"maxOpenConn" json:"maxOpenConn" validate:"min=0,max=1000"`
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

func NewDB(dbType string, connStr string) *DataBase {
	c := DBDefaultConfig
	c.DatabaseType = dbType
	c.ConnStr = connStr
	return NewDBWithConf(c)
}

func NewDBWithConf(conf DBConfig) *DataBase {
	err := std.ValidateStruct(conf)
	std.AssertError(err, "数据库配置不正确")
	logger.Printf("database db(%s) init ...", conf.ConnStr)
	db, err := gorm.Open(conf.DatabaseType, conf.ConnStr)
	std.AssertError(err, "database open failed")
	if conf.ShowSql {
		//use gorm default logger
		//gDb.SetLogger(log.DEBUG)
		db.LogMode(true)
	}
	//dbConfig connection pool
	db.DB().SetMaxIdleConns(conf.MaxIdleConnCount)
	db.DB().SetMaxOpenConns(conf.MaxOpenConnCount)
	return &DataBase{db}
}

func DB() *DataBase {
	std.Assert(gDb != nil, "database not init yet")
	return gDb
}

func ReplaceGlobalDataBase(db *DataBase) (old *DataBase) {
	std.Assert(db != nil && db.DB != nil, "illegal param")
	old, gDb = gDb, db
	return
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

func (this *DataBase) Query() (query *gorm.DB) {
	return DB().DB
}

func dbInit(dbType string, connStr string) {
	c := DBDefaultConfig
	c.DatabaseType = dbType
	c.ConnStr = connStr
	dbInitWithConfig(c)
}

func dbInitWithConfig(conf DBConfig) {
	dbOnce.Do(func() {
		gDb = NewDBWithConf(conf)
	})
}

func dbCleanup() {
	logger.Println("database cleanup ...")
	if err := gDb.Close(); err != nil {
		logger.Printf("error occurred while database close : %s ...", err)
	}
}
