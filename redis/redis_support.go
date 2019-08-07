package redis

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/go-redis/redis"
	"sync"
	"time"
)

const (
	logTag                            = "[Redis]"
	kDefaultRedisReadWriteConnTimeout = 5
)

//Redis 配置
type Config struct {
	Host            string `json:"host" validate:"required"`
	Port            int    `json:"port" validate:"required,min=1025,max=65535"`
	Password        string `json:"password"`
	MaxIdleCount    int    `json:"maxIdle" validate:"min=0,max=1000"`
	MaxActiveCount  int    `json:"maxActive" validate:"min=0,max=1000"`
	DialTimeoutSec  int64  `json:"dialTimeout" validate:"min=0,max=100"`
	ReadTimeoutSec  int64  `json:"readTimeout" validate:"min=0,max=100"`
	WriteTimeoutSec int64  `json:"writeTimeout" validate:"min=0,max=100"`
}

var DefaultConfig = Config{
	Host:            "localhost",
	Port:            6379,
	Password:        "",
	MaxIdleCount:    10,
	MaxActiveCount:  100,
	DialTimeoutSec:  kDefaultRedisReadWriteConnTimeout,
	ReadTimeoutSec:  kDefaultRedisReadWriteConnTimeout,
	WriteTimeoutSec: kDefaultRedisReadWriteConnTimeout,
}

type Cli struct {
	*redis.Client
}

var client *Cli = nil
var once = sync.Once{}
var config = DefaultConfig

func GetCli() *Cli {
	once.Do(func() {
		err := std.ValidateStruct(config)
		std.AssertError(err, "Redis配置不正确")
		redisAddr := fmt.Sprintf("%s:%d", config.Host, config.Port)
		option := &redis.Options{
			Addr:         redisAddr,
			Password:     config.Password,
			DB:           0,
			DialTimeout:  time.Duration(config.DialTimeoutSec) * time.Second,
			ReadTimeout:  time.Duration(config.ReadTimeoutSec) * time.Second,
			WriteTimeout: time.Duration(config.WriteTimeoutSec) * time.Second,
		}
		client = &Cli{
			Client: redis.NewClient(option)}
	})
	return client
}

func Init(host string, pass string) {
	InitWithConfig(Config{
		Host:            host,
		Port:            DefaultConfig.Port,
		Password:        pass,
		MaxIdleCount:    DefaultConfig.MaxIdleCount,
		MaxActiveCount:  DefaultConfig.MaxActiveCount,
		DialTimeoutSec:  DefaultConfig.DialTimeoutSec,
		ReadTimeoutSec:  DefaultConfig.ReadTimeoutSec,
		WriteTimeoutSec: DefaultConfig.WriteTimeoutSec,
	})
}
func InitWithConfig(conf Config) {
	config = conf
	fmt.Printf("%s init ...", logTag)
	GetCli()
}

func Cleanup() {
	fmt.Printf("%s cleanup ...", logTag)
	err := GetCli().Close()
	if err != nil {
		fmt.Printf("%s error occurred while redis close : %s ...", logTag, err)
	}
}
