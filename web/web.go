package web

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"sync"
)

const (
	logTag             = "[Web]"
	kDefaultHttpPort   = 19378
	kDefaultStaticRoot = "static"
)
const (
	jsonIndent       = "    "
	jsonIndentPrefix = ""
)

type Config struct {
	Port          uint64 `json:"port" validate:"min=1025,max=65535"`
	StaticRootDir string `json:"staticRootDir" validate:"required"`
	Debug         bool   `json:"debug"`
}

var DefaultConfig = Config{
	Port:          kDefaultHttpPort,
	StaticRootDir: kDefaultStaticRoot,
	Debug:         false,
}

type Web struct {
	echo *echo.Echo
}

func newWeb() *Web {
	web := &Web{
		echo: echo.New(),
	}
	web.echo.Use(middleware.Recover())
	web.echo.Use(middleware.RequestID())
	web.echo.Use(CustomContextMiddleware)
	if config.Debug {
		web.echo.Use(Dump())
		web.echo.Use(middleware.Logger())
	}
	//允许跨域
	web.echo.Use(middleware.CORS())
	//启用gzip
	web.echo.Use(middleware.Gzip())
	//限制body大小
	web.echo.Use(middleware.BodyLimit("50M"))
	web.echo.Use(middleware.Static(config.StaticRootDir))
	web.echo.Validator = NewWebValidator()
	web.echo.Binder = NewCustomBinder()
	web.echo.HideBanner = true
	web.echo.HidePort = true
	//统一异常处理
	web.echo.HTTPErrorHandler = defaultErrorHandler
	return web
}

func (web *Web) GetEcho() *echo.Echo {
	return web.echo
}

func (web *Web) execute() {
	err := std.ValidateStruct(config)
	std.AssertError(err, "web配置不正确")
	addr := fmt.Sprintf(":%d", config.Port)
	if err := web.echo.Start(addr); err != nil {
		fmt.Println(logTag, " got error when shutting down: ", err)
	}
}

func (web *Web) free() error {
	return web.echo.Close()
}

var once = sync.Once{}
var web *Web = nil
var config = DefaultConfig

func GetWeb() *Web {
	once.Do(func() {
		web = newWeb()
	})
	return web
}

func Init() {
	InitWithConfig(DefaultConfig)
}

//初始化Web服务
func InitWithConfig(conf Config) {
	config = conf
	fmt.Println(logTag, "web(port=", config.Port, ",debug=", config.Debug, ") init  ...")
	GetWeb()
}

func Execute() {
	fmt.Println(logTag, "web listen on : ", config.Port)
	go func() {
		GetWeb().execute()
	}()
}

func Free() {
	fmt.Println(logTag, "stopping ...")
	if err := GetWeb().free(); err != nil {
		fmt.Println(logTag, " got error when shutting down: ", err)
	}
}
