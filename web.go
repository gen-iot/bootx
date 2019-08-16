package bootx

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"sync"
)

const (
	DefaultHttpPort     = 8080
	DefaultStaticRoot   = "static"
	DefaultWebBodyLimit = 5
)

//noinspection ALL
const (
	jsonIndent       = "    "
	jsonIndentPrefix = ""
)

type WebConfig struct {
	Port          uint64 `yaml:"port" json:"port" validate:"min=1,max=65535"`
	StaticRootDir string `yaml:"staticRootDir" json:"staticRootDir" validate:"required"`
	Debug         bool   `yaml:"debug" json:"debug"`
	BodyLimit     int    `yaml:"bodyLimit" json:"bodyLimit"`
}

var WebDefaultConfig = &WebConfig{
	Port:          DefaultHttpPort,
	StaticRootDir: DefaultStaticRoot,
	Debug:         false,
	BodyLimit:     DefaultWebBodyLimit,
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
	if webConfig.BodyLimit <= 0 {
		webConfig.BodyLimit = DefaultWebBodyLimit
	}
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
