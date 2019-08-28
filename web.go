package bootx

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"os"
	"strings"
	"sync"
)

const (
	DefaultHttpPort     = 8080
	DefaultStaticRoot   = ""
	DefaultWebBodyLimit = 5
)

//noinspection ALL
const (
	jsonIndent       = "    "
	jsonIndentPrefix = ""
)

type WebConfig struct {
	Port              int    `yaml:"port" json:"port" validate:"min=1,max=65535"`
	StaticPathPrefix  string `yaml:"staticPathPrefix" json:"staticPathPrefix"`
	StaticRootDir     string `yaml:"staticRootDir" json:"staticRootDir"`
	DirectoryBrowsing bool   `yaml:"directoryBrowsing" json:"directoryBrowsing"`
	Debug             bool   `yaml:"debug" json:"debug"`
	BodyLimit         int    `yaml:"bodyLimit" json:"bodyLimit"`
}

var WebDefaultConfig = WebConfig{
	Port:              DefaultHttpPort,
	StaticRootDir:     DefaultStaticRoot,
	DirectoryBrowsing: false,
	Debug:             false,
	BodyLimit:         DefaultWebBodyLimit,
}

type WebX struct {
	conf WebConfig
	*echo.Echo
	ctxPool *sync.Pool
}

func (this *WebX) grabCtx() *contextImpl {
	ctxImpl := this.ctxPool.Get().(*contextImpl)
	return ctxImpl
}

func (this *WebX) releaseCtx(ctx *contextImpl) {
	std.Assert(ctx != nil, "return ctx is nil")
	this.ctxPool.Put(ctx)
}

func NewWeb() *WebX {
	return NewWebWithConf(WebDefaultConfig)
}

func (this *WebX) NewContext(r *http.Request, w http.ResponseWriter) Context {
	return &contextImpl{
		Context: this.Echo.NewContext(r, w),
		code:    http.StatusOK,
	}
}

func NewWebWithConf(conf WebConfig) *WebX {
	web := &WebX{
		conf,
		echo.New(),
		&sync.Pool{
			New: func() interface{} {
				return new(contextImpl)
			},
		},
	}
	web.Use(middleware.Recover())
	web.Use(middleware.RequestID())
	web.Use(web.customContextMiddleware)
	//允许跨域
	web.Use(middleware.CORS())
	//启用gzip
	web.Use(middleware.Gzip())
	if conf.Debug {
		web.Use(Dump())
		web.Use(middleware.Logger())
	}
	//限制body大小
	if conf.BodyLimit > 0 {
		web.Use(middleware.BodyLimit(fmt.Sprintf("%dM", conf.BodyLimit)))
	}
	//static
	conf.StaticRootDir = strings.Trim(conf.StaticRootDir, " ")
	conf.StaticPathPrefix = strings.Trim(conf.StaticPathPrefix, " ")
	if conf.StaticRootDir != "" {
		err := os.MkdirAll(conf.StaticRootDir, os.ModePerm)
		std.AssertError(err, "create gWebX static dir failed")
		staticConfig := middleware.StaticConfig{
			Root:   conf.StaticRootDir,
			HTML5:  true,
			Browse: conf.DirectoryBrowsing,
		}
		if conf.StaticPathPrefix != "" {
			g := web.Group(conf.StaticPathPrefix)
			g.Use(middleware.StaticWithConfig(staticConfig))
		} else {
			web.Use(middleware.StaticWithConfig(staticConfig))
		}
	}
	web.Validator = NewWebValidator()
	web.Binder = NewCustomBinder()
	web.HideBanner = true
	//gWebX.HidePort = true
	//统一异常处理
	web.HTTPErrorHandler = web.defaultErrorHandler
	return web
}

var webOnce = sync.Once{}
var gWebX *WebX = nil

func Web() *WebX {
	std.Assert(gWebX != nil, "gWebX not init yet")
	return gWebX
}

func webInit() {
	webInitWithConfig(WebDefaultConfig)
}

func webInitWithConfig(conf WebConfig) {
	if conf.BodyLimit <= 0 {
		conf.BodyLimit = DefaultWebBodyLimit
	}
	webOnce.Do(func() {
		err := std.ValidateStruct(conf)
		std.AssertError(err, "web配置不正确")
		logger.Printf("gWebX(port=%d,debug=%v) init  ...", conf.Port, conf.Debug)
		gWebX = NewWebWithConf(conf)
	})
}

func (this *WebX) start() {
	go func() {
		addr := fmt.Sprintf(":%d", this.conf.Port)
		if err := this.Start(addr); err != nil {
			logger.Println("Web got error when shutting down : ", err)
		}
	}()
}

func (this *WebX) stop() {
	if err := this.Close(); err != nil {
		logger.Println(" got error when shutting down: ", err)
	}
}
