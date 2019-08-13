package bootx

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/go-redis/redis"
	"sync"
	"time"
)

const (
	defaultRedisReadWriteConnTimeout = 5
)

//Redis 配置
type RedisConfig struct {
	Host            string `yaml:"host" json:"host" validate:"required"`
	Port            int    `yaml:"port" json:"port" validate:"required,min=1025,max=65535"`
	Password        string `yaml:"password" json:"password"`
	MaxIdleCount    int    `yaml:"maxIdle" json:"maxIdle" validate:"min=0,max=1000"`
	MaxActiveCount  int    `yaml:"maxActive"  json:"maxActive" validate:"min=0,max=1000"`
	DialTimeoutSec  int64  `yaml:"dialTimeout" json:"dialTimeout" validate:"min=0,max=100"`
	ReadTimeoutSec  int64  `yaml:"readTimeout" json:"readTimeout" validate:"min=0,max=100"`
	WriteTimeoutSec int64  `yaml:"writeTimeout" json:"writeTimeout" validate:"min=0,max=100"`
}

var RedisDefaultConfig = &RedisConfig{
	Host:            "localhost",
	Port:            6379,
	Password:        "",
	MaxIdleCount:    10,
	MaxActiveCount:  100,
	DialTimeoutSec:  defaultRedisReadWriteConnTimeout,
	ReadTimeoutSec:  defaultRedisReadWriteConnTimeout,
	WriteTimeoutSec: defaultRedisReadWriteConnTimeout,
}

type RedisClient struct {
	*redis.Client
}

var client *RedisClient = nil
var redisOnce = sync.Once{}
var redisConfig = RedisDefaultConfig

func RedisCli() *RedisClient {
	std.Assert(redisConfig != nil, "redis not config yet")
	redisOnce.Do(func() {
		err := std.ValidateStruct(redisConfig)
		std.AssertError(err, "Redis配置不正确")
		logger.Printf("redis init ...")
		redisAddr := fmt.Sprintf("%s:%d", redisConfig.Host, redisConfig.Port)
		option := &redis.Options{
			Addr:         redisAddr,
			Password:     redisConfig.Password,
			DB:           0,
			DialTimeout:  time.Duration(redisConfig.DialTimeoutSec) * time.Second,
			ReadTimeout:  time.Duration(redisConfig.ReadTimeoutSec) * time.Second,
			WriteTimeout: time.Duration(redisConfig.WriteTimeoutSec) * time.Second,
		}
		client = &RedisClient{
			Client: redis.NewClient(option)}
	})
	return client
}

func redisInit(host string, pass string) {
	redisInitWithConfig(&RedisConfig{
		Host:            host,
		Port:            RedisDefaultConfig.Port,
		Password:        pass,
		MaxIdleCount:    RedisDefaultConfig.MaxIdleCount,
		MaxActiveCount:  RedisDefaultConfig.MaxActiveCount,
		DialTimeoutSec:  RedisDefaultConfig.DialTimeoutSec,
		ReadTimeoutSec:  RedisDefaultConfig.ReadTimeoutSec,
		WriteTimeoutSec: RedisDefaultConfig.WriteTimeoutSec,
	})
}
func redisInitWithConfig(conf *RedisConfig) {
	redisConfig = conf
	RedisCli()
}

func redisCleanup() {
	logger.Println("%redis cleanup ...")
	err := RedisCli().Close()
	if err != nil {
		logger.Printf("error occurred while redis close : %s ...", err)
	}
}
