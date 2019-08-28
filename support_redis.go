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
	Port            int    `yaml:"port" json:"port" validate:"required,min=1,max=65535"`
	Password        string `yaml:"password" json:"password"`
	MaxIdleCount    int    `yaml:"maxIdle" json:"maxIdle" validate:"min=0,max=1000"`
	MaxActiveCount  int    `yaml:"maxActive"  json:"maxActive" validate:"min=0,max=1000"`
	DialTimeoutSec  int64  `yaml:"dialTimeout" json:"dialTimeout" validate:"min=0,max=100"`
	ReadTimeoutSec  int64  `yaml:"readTimeout" json:"readTimeout" validate:"min=0,max=100"`
	WriteTimeoutSec int64  `yaml:"writeTimeout" json:"writeTimeout" validate:"min=0,max=100"`
}

var RedisDefaultConfig = RedisConfig{
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

var gRedisCli *RedisClient = nil
var redisOnce = sync.Once{}

func NewRedisCli(host string, pass string) *RedisClient {
	c := RedisDefaultConfig
	c.Host = host
	c.Password = pass
	return NewRedisCliWithConf(c)
}

func NewRedisCliWithConf(conf RedisConfig) *RedisClient {
	err := std.ValidateStruct(conf)
	std.AssertError(err, "Redis配置不正确")
	logger.Println("redis init ...")
	redisAddr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	option := &redis.Options{
		Addr:         redisAddr,
		Password:     conf.Password,
		DB:           0,
		DialTimeout:  time.Duration(conf.DialTimeoutSec) * time.Second,
		ReadTimeout:  time.Duration(conf.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(conf.WriteTimeoutSec) * time.Second,
	}
	return &RedisClient{Client: redis.NewClient(option)}
}

func RedisCli() *RedisClient {
	std.Assert(gRedisCli != nil, "redis not init yet")
	return gRedisCli
}

func redisInit(host string, pass string) {
	c := RedisDefaultConfig
	c.Host = host
	c.Password = pass
	redisInitWithConfig(c)
}

func redisInitWithConfig(conf RedisConfig) {
	redisOnce.Do(func() {
		gRedisCli = NewRedisCliWithConf(conf)
	})
}

func redisCleanup() {
	logger.Println("redis cleanup ...")
	err := RedisCli().Close()
	if err != nil {
		logger.Printf("error occurred while redis close : %s ...", err)
	}
}
