package bootx

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"sync"
)

const (
	kDefaultHttpPort   = 19378
	kDefaultStaticRoot = "static"
)

//noinspection ALL
const (
	jsonIndent       = "    "
	jsonIndentPrefix = ""
)

type WebConfig struct {
	Port          uint64 `json:"port" validate:"min=1025,max=65535"`
	StaticRootDir string `json:"staticRootDir" validate:"required"`
	Debug         bool   `json:"debug"`
	BodyLimit     *int   `json:"bodyLimit"`
}

var WebDefaultConfig = &WebConfig{
	Port:          kDefaultHttpPort,
	StaticRootDir: kDefaultStaticRoot,
	Debug:         false,
}

type WebX struct {
	*echo.Echo
}

func newWeb() *WebX {
	web := &WebX{
		echo.New(),
	}
	web.Use(middleware.Recover())
	web.Use(middleware.RequestID())
	web.Use(CustomContextMiddleware)
	if webConfig.Debug {
		web.Use(Dump())
		web.Use(middleware.Logger())
	}
	//允许跨域
	web.Use(middleware.CORS())
	//启用gzip
	web.Use(middleware.Gzip())
	//限制body大小
	web.Use(middleware.BodyLimit(fmt.Sprintf("%dM", webConfig.BodyLimit)))
	web.Use(middleware.Static(webConfig.StaticRootDir))
	web.Validator = NewWebValidator()
	web.Binder = NewCustomBinder()
	web.HideBanner = true
	//web.HidePort = true
	//统一异常处理
	web.HTTPErrorHandler = defaultErrorHandler
	return web
}

var webOnce = sync.Once{}
var web *WebX = nil
var webConfig *WebConfig = nil

func Web() *WebX {
	std.Assert(webConfig != nil, "web not config yet")
	webOnce.Do(func() {
		err := std.ValidateStruct(webConfig)
		std.AssertError(err, "web配置不正确")
		logger.Printf("web(port=%d,debug=%v) init  ...", webConfig.Port, webConfig.Debug)
		web = newWeb()
	})
	return web
}

func webInit() {
	webInitWithConfig(WebDefaultConfig)
}

func webInitWithConfig(conf *WebConfig) {
	webConfig = conf
	Web()
}

func (web *WebX) start() {
	go func() {
		addr := fmt.Sprintf(":%d", webConfig.Port)
		if err := web.Start(addr); err != nil {
			logger.Println("web got error when shutting down : ", err)
		}
	}()
}

func (web *WebX) stop() {
	if err := web.Close(); err != nil {
		logger.Println(logTag, " got error when shutting down: ", err)
	}
}