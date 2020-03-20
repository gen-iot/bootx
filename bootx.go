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
	webConf, dbConf, redisConf := initModules(configs...)
	//if no gWebX config ,use default config
	if !webConf {
		webInit()
	}
	if dbConf {
		defer dbCleanup()
	}
	if redisConf {
		defer redisCleanup()
	}
	app.Bootstrap()
	defer app.Shutdown()
	//start gWebX service
	Web().start()
	defer Web().stop()
	getKernel().waitForExit()
}

func Kill() {
	getKernel().kill()
}

func initModules(configs ...interface{}) (webConf, dbConf, redisConf bool) {
	for _, conf := range configs {
		switch c := conf.(type) {
		case WebConfig:
			webInitWithConfig(c)
		case *WebConfig:
			webInitWithConfig(*c)
			webConf = true
		case DBConfig:
			dbInitWithConfig([]DBConfig{c})
			dbConf = true
		case *DBConfig:
			dbInitWithConfig([]DBConfig{*c})
			dbConf = true
		case []DBConfig:
			dbInitWithConfig(c)
			dbConf = true
		case []*DBConfig:
			ds := make([]DBConfig, 0, len(c))
			for _, db := range c {
				ds = append(ds, *db)
			}
			dbInitWithConfig(ds)
			dbConf = true
		case RedisConfig:
			redisInitWithConfig(c)
		case *RedisConfig:
			redisInitWithConfig(*c)
			redisConf = true
		}
	}
	return
}
