package bootx

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
)

//统一异常处理
func (this *WebX) defaultErrorHandler(err error, ctx Context) {
	code := 500
	rsp := make(map[string]interface{})
	switch cause := errors.Cause(err).(type) {
	case *echo.HTTPError:
		{
			msg := fmt.Sprintf("%v", cause.Message)
			code = cause.Code
			rsp["code"] = cause.Code
			rsp["message"] = msg
		}
	default:
		{
			logger.Printf("unexpect error :%v", err)
			msg := "unexpect error "
			if this.conf.Debug {
				msg = fmt.Sprintf("%v", err)
			}
			code = http.StatusInternalServerError
			rsp["code"] = code
			rsp["message"] = msg
		}
	}
	// Send response
	if !ctx.Response().Committed {
		if ctx.Request().Method == http.MethodHead {
			err = ctx.NoContent(code)
		} else {
			err = ctx.JSONPretty(code, rsp, jsonIndent)
		}
		if err != nil {
			logger.Println(err)
		}
	}
}
