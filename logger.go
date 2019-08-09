package bootx

import "fmt"

const logTag = "[Bootx]"

type Logger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

type defaultLogger struct {
}

func (this *defaultLogger) Println(v ...interface{}) {
	vWithTag := []interface{}{logTag}
	vWithTag = append(vWithTag, v...)
	fmt.Println(vWithTag...)
}

func (this *defaultLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(fmt.Sprintf("%s %s\n", logTag, format), v...)
}

var logger = defaultLogger{}
