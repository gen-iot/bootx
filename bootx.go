package bootx

type Application interface {
	GetName() string
	GetVersion() string
	Bootstrap()
	Shutdown()
}

func Bootstrap(app Application, configs ...interface{}) {
	appName := app.GetName()
	appVersion := app.GetVersion()
	logger.Printf("%s %s bootstrap ...", appName, appVersion)
	initKernel()
	defer cleanupKernel()
	//todo init with configs

	webInit()
	app.Bootstrap()
	defer app.Shutdown()
	//启动Web服务
	Web().start()
	defer Web().stop()
	getKernel().waitForExit()
}
