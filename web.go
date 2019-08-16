package bootx

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"os"
	"strings"
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
	Port              uint64 `yaml:"port" json:"port" validate:"min=1,max=65535"`
	StaticPathPrefix  string `yaml:"staticPathPrefix" json:"staticPathPrefix"`
	StaticRootDir     string `yaml:"staticRootDir" json:"staticRootDir"`
	DirectoryBrowsing bool   `yaml:"directoryBrowsing" json:"directoryBrowsing"`
	Debug             bool   `yaml:"debug" json:"debug"`
	BodyLimit         int    `yaml:"bodyLimit" json:"bodyLimit"`
}

var WebDefaultConfig = &WebConfig{
	Port:              DefaultHttpPort,
	StaticRootDir:     DefaultStaticRoot,
	DirectoryBrowsing: false,
	Debug:             false,
	BodyLimit:         DefaultWebBodyLimit,
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
	if webConfig.BodyLimit > 0 {
		web.Use(middleware.BodyLimit(fmt.Sprintf("%dM", webConfig.BodyLimit)))
	}
	//static
	webConfig.StaticRootDir = strings.Trim(webConfig.StaticRootDir, " ")
	webConfig.StaticPathPrefix = strings.Trim(webConfig.StaticPathPrefix, " ")
	if webConfig.StaticRootDir != "" {
		err := os.MkdirAll(webConfig.StaticRootDir, os.ModePerm)
		std.AssertError(err, "create web static dir failed")
		staticConfig := middleware.StaticConfig{
			Root:   webConfig.StaticRootDir,
			HTML5:  true,
			Browse: webConfig.DirectoryBrowsing,
		}
		if webConfig.StaticPathPrefix != "" {
			g := web.Group(webConfig.StaticPathPrefix)
			g.Use(middleware.StaticWithConfig(staticConfig))
		} else {
			web.Use(middleware.StaticWithConfig(staticConfig))
		}
	}
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
