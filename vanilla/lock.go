package vanilla

import (
	"github.com/kfchen81/beego"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"errors"
	"time"
	"github.com/go-redsync/redsync"
)

type ILock interface {
	Lock(key string, args ...*LockOption) (*redsync.Mutex, error)
}

type LockOption struct{
	key string
	timeout int
	tries int
}

func (this *LockOption) SetTimeout(n int) *LockOption{
	this.timeout = n
	return this
}

func (this *LockOption) SetTryTimes(n int) *LockOption{
	this.tries = n
	return this
}

func NewLockOption(key string) *LockOption{
	instance := new(LockOption)
	instance.key = key
	instance.timeout = 10
	instance.tries = 3

	return instance
}

//DummyLock 空的锁引擎
type DummyLock struct {

}

func (this *DummyLock) Lock(key string, args ...*LockOption) (*redsync.Mutex, error) {
	beego.Debug(fmt.Sprintf("[lock] lock by dummy engine : %s", key))
	return nil, nil
}

//RedisLock 基于redis的锁引擎
type RedisLock struct {
	engine *redsync.Redsync
}

func (this *RedisLock) Lock(key string, args ...*LockOption) (*redsync.Mutex, error) {
	beego.Debug(fmt.Sprintf("[lock] lock by redis engine : %s", key))
	if this.engine == nil {
		beego.Warn("[lock] redsync engine is nil")
		return nil, nil
	} else {
		var option *LockOption
		switch len(args) {
		case 1:
			option = args[0]
		default:
			option = NewLockOption(key)
		}
		mutex := this.engine.NewMutex(key, redsync.SetExpiry(time.Duration(option.timeout)*time.Second), redsync.SetTries(option.tries))
		err := mutex.Lock()
		if err != nil {
			beego.Error(err)
			return nil, err
		} else {
			if mutex != nil{
				beego.Debug(fmt.Sprintf("[lock] redis lock: %s acquired", key))
			}
			return mutex, nil
		}
	}
}


var Lock ILock //暴露的锁

var lockRedisAddress string = ""
var lockDbNum int = 1
var lockRedisPassword string = ""
var lockRedisPool *redis.Pool = nil


func init() {
	lockEngine := beego.AppConfig.String("lock::ENGINE")
	if lockEngine == "" {
		lockEngine = "dummy"
	}

	if lockEngine == "dummy" {
		beego.Info("[lock] use DummyLock")
		Lock = new(DummyLock)
	} else {
		lockRedisAddress = beego.AppConfig.String("lock::REDIS_ADDRESS")
		lockDbNum, _ = beego.AppConfig.Int("lock::REDIS_DB")
		lockRedisPassword = beego.AppConfig.String("lock::REDIS_PASSWORD")
		beego.Info(fmt.Sprintf("[lock] use RedisLock: %s - %d", lockRedisAddress, lockDbNum))

		// initialize a new pool
		lockRedisPool = &redis.Pool{
			MaxIdle:     10,
			IdleTimeout: 180 * time.Second,
			Dial: func() (c redis.Conn, err error) {
				if lockRedisAddress == "" {
					return nil, errors.New("invalid redisAddress")
				}

				c, err = redis.Dial("tcp", lockRedisAddress)
				if err != nil {
					beego.Error(err)
					return nil, err
				}

				if lockRedisPassword != "" {
					if _, err := c.Do("AUTH", lockRedisPassword); err != nil {
						beego.Error(err)
						c.Close()
						return nil, err
					}
				}

				_, selecterr := c.Do("SELECT", lockDbNum)
				if selecterr != nil {
					beego.Error(selecterr)
					c.Close()
					return nil, selecterr
				}
				return
			},
			MaxConnLifetime: 60 * time.Minute,
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				if time.Since(t) < time.Minute {
					return nil
				}
				_, err := c.Do("PING")
				return err
			},
		}

		//pool热身
		c := lockRedisPool.Get()
		defer c.Close()

		//创建
		Lock = &RedisLock{
			engine: redsync.New([]redsync.Pool{lockRedisPool}),
		}
	}
}