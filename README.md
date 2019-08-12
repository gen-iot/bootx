# Bootx

> go web api的基础框架

## Fetures

- [x] Database 
- [x] Redis 
- [x] Mqtt Publish

## Usages

**See example**

```go
package main

import (
	"github.com/gen-iot/bootx"
)

type FooApp struct {
}

func (f FooApp) GetName() string {
	return "Foo"
}

func (f FooApp) GetVersion() string {
	return "1.0.0"
}

func (f FooApp) Bootstrap() {
	bootx.Web().GET("", bootx.BuildHttpHandler(func(ctx bootx.Context) error {
		return ctx.String(200, "hello word")
	}))
}

func (f FooApp) Shutdown() {

}

func main() {
	//only start web (default listen at 8080)
	bootx.Bootstrap(&FooApp{})
	//only start web ,use custom config
	/*
		bootx.Bootstrap(&FooApp{}, bootx.WebConfig{
			Port:          8090,
			StaticRootDir: "",
			Debug:         false,
			BodyLimit:     0,
		})
	*/
	/* start web with db & redis
	bootx.Bootstrap(&FooApp{},
		bootx.WebConfig{
			Port:          8090,
			StaticRootDir: "static",
			Debug:         false,
			BodyLimit:     50,
		},
	    // conn string see https://gorm.io/docs/connecting_to_the_database.html
		bootx.DBConfig{
			DatabaseType:     "mysql", //support mysql postgres sqlite3 mssql
			ConnStr:          "user:password@/dbname?charset=utf8&parseTime=True&loc=Local",
			ShowSql:          false,
			MaxIdleConnCount: 0,
			MaxOpenConnCount: 0,
		},
		bootx.RedisConfig{
			Host:            "127.0.0.1",
			Port:            6379,
			Password:        "password",
			MaxIdleCount:    10,
			MaxActiveCount:  100,
			DialTimeoutSec:  10,
			ReadTimeoutSec:  10,
			WriteTimeoutSec: 10,
		},
	)
	*/
}
```

