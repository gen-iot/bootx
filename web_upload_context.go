package bootx

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"io/ioutil"
	"mime/multipart"
	"path"
	"strings"
)

const (
	KSizeByte = 1
	KSizeKb   = KSizeByte * 1024
	KSizeMb   = KSizeKb * 1024
	KSizeGb   = KSizeMb * 1024
)

type UploadContext struct {
	echo.Context
	FormKeyName     string
	FileMd5         string   // 文件 Md5值,如果不存在,则忽略 md5校验
	FileMaxSize     int64    // 最大允许的文件大小
	AllowExtensions []string //允许的文件类型
	*multipart.FileHeader
	FileExt      string
	FileName     string
	FormKeyValid bool
	DataBytes    []byte
}

var errMissFile = errors.New("缺少file参数")
var errTooLargeFile = errors.New("文件大小超过上传限制")
var errNotAllowFileType = errors.New("不支持的文件类型")

func (ctx *UploadContext) Validate() error {
	file, err := ctx.FormFile(ctx.FormKeyName)
	if err != nil {
		return err
	}
	if len(strings.Trim(file.Filename, " ")) == 0 {
		return errMissFile
	}
	ctx.FormKeyValid = true
	if file.Size > ctx.FileMaxSize {
		return errTooLargeFile
	}
	ctx.FileName = file.Filename
	ctx.FileExt = strings.ToLower(path.Ext(file.Filename))
	if len(ctx.AllowExtensions) > 0 {
		for _, ext := range ctx.AllowExtensions {
			if strings.ToLower(ext) == ctx.FileExt {
				break
			}
		}
		return errNotAllowFileType
	}
	ctx.FileHeader = file
	return nil
}

func (ctx *UploadContext) ReadAll() ([]byte, error) {
	if !ctx.FormKeyValid {
		return nil, errMissFile
	}
	file, err := ctx.FileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	ctx.DataBytes, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return ctx.DataBytes[:], nil
}
