package bootx

import (
	"github.com/gen-iot/std"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

var k *kernel = nil
var kOnce = sync.Once{}

type kernel struct {
	WorkSpace       string
	choseSignalChan chan bool
	Lock            *sync.Mutex
}

func newKernel() *kernel {
	wd, err := os.Getwd()
	std.AssertError(err, "get wd failed")
	return &kernel{
		WorkSpace:       wd,
		choseSignalChan: make(chan bool),
		Lock:            &sync.Mutex{},
	}
}

func getKernel() *kernel {
	kOnce.Do(func() {
		k = newKernel()
	})
	return k
}

func (this *kernel) kill() {
	logger.Println("shutdown ...")
	close(this.choseSignalChan)
}

//block event
func (this *kernel) waitForExit() {
	logger.Println("running ...")
	this.handleKillSignal()
	<-this.choseSignalChan
}

func (this *kernel) handleKillSignal() {
	c := make(chan os.Signal)
	//监听指定信号 ctrl+c kill
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL:
				logger.Println("receive signal", s)
				this.kill()
			default:
				logger.Println("receive signal", s)
			}
		}
	}()
}

func initKernel() {
	logger.Println("main loop init ...")
	initGolang()
	getKernel()
}

func initGolang() {
	logger.Println("golang init ...")
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func cleanupKernel() {
	logger.Println("exited !")
}
