package main

import (
	"fmt"
	"github.com/gen-iot/bootx"
	"github.com/gen-iot/bootx/middleware"
)

type FooApp struct {
}

func (f FooApp) GetName() string {
	return "Foo"
}

func (f FooApp) GetVersion() string {
	return "1.0.0"
}

type FooResponse struct {
	Msg string `json:"msg"`
}

func (f FooApp) Bootstrap() {
	web := bootx.Web()
	web.PreUse(middleware.BodyDump(middleware.DumpTextPlain | middleware.DumpForm | middleware.DumpJson))
	web.GET("", web.BuildHttpHandler(func(ctx bootx.Context) error {
		return ctx.String(200, "hello word")
	}))
	web.GET("/foo/bar",
		web.BuildHttpHandler(
			func() (*FooResponse, error) {
				return &FooResponse{Msg: "hello word"}, nil
			}))
	web.POST("/foo/bar",
		web.BuildHttpHandler(
			func(ctx bootx.Context, req *TestBindRequest) (*FooResponse, error) {
				fmt.Println(req.Id)
				r1 := new(TestBindAgainRequest)
				err := ctx.Bind(r1)
				if err != nil {
					return &FooResponse{Msg: "bind again err"}, nil
				}
				return &FooResponse{Msg: "hello word"}, nil
			}))
}

type TestBindRequest struct {
	Id  string `json:"id"`
	Msg string `json:"msg"`
}

type TestBindAgainRequest struct {
	Id   string `json:"id"`
	Msg  string `json:"msg"`
	Name string `json:"name"`
}

func (f FooApp) Shutdown() {

}

func main() {
	//only start web (default listen at 8080)
	bootx.Bootstrap(&FooApp{},
		bootx.WebConfig{
			Port:  8080,
			Debug: true,
		})
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
