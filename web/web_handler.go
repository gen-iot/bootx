package web

import (
	"fmt"
	"github.com/gen-iot/log"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
)

//统一异常处理
func defaultErrorHandler(err error, ctx echo.Context) {
	switch cause := errors.Cause(err).(type) {
	case *echo.HTTPError:
		{
			//http error
			msg := fmt.Sprintf("%v", cause.Message)
			e := ctx.JSONPretty(cause.Code,
				map[string]interface{}{
					"code":    cause.Code,
					"message": msg,
				}, jsonIndent)
			if e != nil {
				log.ERROR.Println(e)
			}
		}
	default:
		{
			log.ERROR.Printf("unknown error :%v", err)
			msg := "unknown error "
			if config.Debug {
				msg = fmt.Sprintf("%v", err)
			}
			//500
			e := ctx.JSONPretty(http.StatusInternalServerError,
				map[string]interface{}{
					"code":    -1,
					"message": msg,
				}, jsonIndent)
			if e != nil {
				log.ERROR.Println(e)
			}
		}
	}
}
