package bootx

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/pkg/errors"
	"sync"
)

type DBConfig struct {
	Default          bool   `yaml:"default" json:"default"`
	Name             string `yaml:"name" json:"name"`
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
	conf DBConfig
}

var dbOnce = sync.Once{}
var dbMap = make(map[string]*DataBase)
var dbNames = make([]string, 0)
var defaultDb *DataBase = nil
var dbRwLock = &sync.RWMutex{}

func NewDB(dbType string, connStr string, name ...string) *DataBase {
	c := DBDefaultConfig
	c.DatabaseType = dbType
	c.ConnStr = connStr
	if len(name) > 0 && len(name[0]) > 0 {
		c.Name = name[0]
	}
	return NewDBWithConf(c)
}

func NewDBWithConf(conf DBConfig) *DataBase {
	if len(conf.Name) == 0 {
		conf.Name = std.GenRandomUUID()
	}
	err := std.ValidateStruct(conf)
	std.AssertError(err, "invalid database configuration")
	logger.Printf("database db(%s %s) init ...", conf.Name, conf.ConnStr)
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
	return &DataBase{db, conf}
}

func DB() *DataBase {
	std.Assert(defaultDb != nil, "default database not init yet")
	dbRwLock.RLock()
	defer dbRwLock.RUnlock()
	return defaultDb
}

type NoSuchDatabase string

func (e NoSuchDatabase) Error() string {
	return fmt.Sprintf("no such databse named '%s'", string(e))
}

func DBNames() []string {
	return dbNames
}

func DB2(name string) (*DataBase, error) {
	dbRwLock.RLock()
	defer dbRwLock.RUnlock()
	return db2(name)
}

func db2(name string) (*DataBase, error) {
	if len(name) == 0 {
		return nil, errors.New("please specified the name of db to get")
	}
	if len(dbMap) == 0 {
		return nil, NoSuchDatabase(name)
	}
	db, ok := dbMap[name]
	if !ok {
		return nil, NoSuchDatabase(name)
	}
	return db, nil
}

//
// Deprecated: use ChDefaultDb(name) instead
//
func ReplaceGlobalDataBase(db *DataBase) (old *DataBase) {
	dbRwLock.Lock()
	defer dbRwLock.Unlock()
	name := db.Name()
	if len(name) == 0 {
		db.conf.Name = std.GenRandomUUID()
		name = db.Name()
	}
	exist, ok := dbMap[name]
	if ok {
		std.Assert(exist == db, "")
	} else {
		std.AssertError(addDB(db), "replace default database failed ")
	}
	defaultDb, old = db, defaultDb
	return
}

type DBNameDuplicateAdd string

func (e DBNameDuplicateAdd) Error() string {
	return fmt.Sprintf("database '%s' duplicate already exist", string(e))
}

func AddDB(db *DataBase) error {
	dbRwLock.Lock()
	defer dbRwLock.Unlock()
	return addDB(db)
}

func addDB(db *DataBase) error {
	name := db.Name()
	std.Assert(len(name) > 0, "not a valid database")
	_, ok := dbMap[name]
	if ok {
		return DBNameDuplicateAdd(name)
	}
	dbMap[name] = db
	dbNames = append(dbNames, name)
	return nil
}

func RemoveDB(name string) {
	dbRwLock.Lock()
	defer dbRwLock.Unlock()
	delete(dbMap, name)
	dbNames := make([]string, 0, len(dbNames))
	for _, n := range dbNames {
		if n != name {
			dbNames = append(dbNames, n)
		}
	}
}

func ChDefaultDb(name string) error {
	dbRwLock.Lock()
	defer dbRwLock.Unlock()
	db, err := db2(name)
	if err != nil {
		return err
	}
	defaultDb = db
	return nil
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
	return this.DB
}

func (this *DataBase) Name() string {
	return this.conf.Name
}

func (this *DataBase) DBType() string {
	return this.conf.DatabaseType
}

func (this *DataBase) ConnStr() string {
	return this.conf.ConnStr
}

func (this *DataBase) Conf() DBConfig {
	return this.conf
}

func dbInit(dbType string, connStr string) {
	c := DBDefaultConfig
	c.DatabaseType = dbType
	c.ConnStr = connStr
	dbInitWithConfig([]DBConfig{c})
}

func dbInitWithConfig(conf []DBConfig) {
	std.Assert(len(conf) > 0, "at least one database config should be specified")
	dbOnce.Do(func() {
		dbRwLock.Unlock()
		defer dbRwLock.Unlock()
		dbMap = make(map[string]*DataBase, len(conf))
		dbNames = make([]string, 0, len(conf))
		var first *DataBase = nil
		for i, c := range conf {
			db := NewDBWithConf(c)
			name := db.Name()
			if i == 0 {
				first = db
			}
			if c.Default {
				std.Assert(defaultDb == nil, "more than one database set to default")
				defaultDb = db
			}
			std.AssertError(addDB(db), fmt.Sprintf("database '%s' init failed ", name))
		}
		if defaultDb == nil {
			defaultDb = first
		}
	})
}

func dbCleanup() {
	for name, db := range dbMap {
		logger.Printf("database(%s) cleanup ...", name)
		std.CloseIgnoreErr(db)
	}
}
