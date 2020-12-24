package vanilla

import (
	"time"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/kfchen81/beego"
	"fmt"
	"context"
	"github.com/opentracing/opentracing-go"
)

var redisAddress string = ""
var dbNum int = 1
var redisPassword string = ""
var pool *redis.Pool = nil

type redisStruct struct {

}

// actually do the redis cmds, args[0] must be the key name.
func (this *redisStruct) Do(ctx context.Context, commandName string, args ...interface{}) (reply interface{}, err error) {
	//记录open tracing
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		operationName := fmt.Sprintf("redis-%s", commandName)
		subSpan := span.Tracer().StartSpan(
			operationName,
			opentracing.ChildOf(span.Context()),
		)
		if subSpan != nil {
			defer subSpan.Finish()
		}
	}

	c := pool.Get()
	defer c.Close()

	return c.Do(commandName, args...)
}

var hash2script = make(map[string]*redis.Script)

func (this *redisStruct) LoadScript(keyCount int, scriptContent string) (hash string, err error) {
	c := pool.Get()
	defer c.Close()
	
	script := redis.NewScript(keyCount, scriptContent)
	err = script.Load(c)
	if err != nil {
		return "", err
	}
	
	hash = script.Hash()
	hash2script[hash] = script
	
	return hash, nil
}

func (this *redisStruct) RunScript(hash string, keysAndArgs ...interface{}) (interface{}, error) {
	c := pool.Get()
	defer c.Close()
	
	if script, ok := hash2script[hash]; ok {
		return script.Do(c, keysAndArgs...)
	} else {
		return "", errors.New("no script for hash")
	}
}

// Get cache from redis.
func (this *redisStruct) Get(ctx context.Context, key string) interface{} {
	if v, err := this.Do(ctx,"GET", key); err == nil {
		return v
	}
	return nil
}

// GetMulti get cache from redis.
func (this *redisStruct) GetMulti(ctx context.Context, keys []string) []interface{} {
	c := pool.Get()
	defer c.Close()

	//记录open tracing
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		operationName := "redis-MGET"
		subSpan := span.Tracer().StartSpan(
			operationName,
			opentracing.ChildOf(span.Context()),
		)
		if subSpan != nil {
			defer subSpan.Finish()
		}
	}

	var args []interface{}
	for _, key := range keys {
		args = append(args, key)
	}
	values, err := redis.Values(c.Do("MGET", args...))
	if err != nil {
		return nil
	}
	return values
}

// Hset Sets field in the hash stored at key to value.
// If key does not exist, a new key holding a hash is created.
// If field already exists in the hash, it is overwritten.
func (this *redisStruct) Hset(ctx context.Context, key string, field string, val interface{}) error {
	_, err := this.Do(ctx,"HSET", key, field, val)
	return err
}

// Put put cache to redis.
func (this *redisStruct) SexEx(ctx context.Context, key string, val interface{}, timeout time.Duration) error {
	_, err := this.Do(ctx, "SETEX", key, int64(timeout/time.Second), val)
	return err
}

// Put put cache to redis.
func (this *redisStruct) Set(ctx context.Context, key string, val interface{}) error {
	_, err := this.Do(ctx,"SET", key, val)
	return err
}

// Delete delete cache in redis.
func (this *redisStruct) Delete(ctx context.Context, key string) error {
	_, err := this.Do(ctx, "DEL", key)
	return err
}

// IsExist check cache's existence in redis.
func (this *redisStruct) IsExist(ctx context.Context, key string) bool {
	v, err := redis.Bool(this.Do(ctx,"EXISTS", key))
	if err != nil {
		return false
	}
	return v
}


// Incr increase counter in redis.
func (this *redisStruct) Incr(ctx context.Context, key string) error {
	_, err := redis.Bool(this.Do(ctx,"INCRBY", key, 1))
	return err
}

// Decr decrease counter in redis.
func (this *redisStruct) Decr(ctx context.Context, key string) error {
	_, err := redis.Bool(this.Do(ctx,"INCRBY", key, -1))
	return err
}

// ClearAll clean all cache in redis. delete this redis collection.
func (this *redisStruct) ClearAll(ctx context.Context, prefix string) error {
	c := pool.Get()
	defer c.Close()

	//记录open tracing
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		operationName := "redis-DEL_KEYS"
		subSpan := span.Tracer().StartSpan(
			operationName,
			opentracing.ChildOf(span.Context()),
		)
		if subSpan != nil {
			defer subSpan.Finish()
		}
	}

	cachedKeys, err := redis.Strings(c.Do("KEYS", prefix+":*"))
	if err != nil {
		return err
	}
	for _, str := range cachedKeys {
		if _, err = c.Do("DEL", str); err != nil {
			return err
		}
	}
	return err
}

func dialFunc() (c redis.Conn, err error) {
	if redisAddress == "" {
		return nil, errors.New("invalid redisAddress")
	}

	c, err = redis.Dial("tcp", redisAddress)
	if err != nil {
		beego.Error(err)
		return nil, err
	}

	if redisPassword != "" {
		if _, err := c.Do("AUTH", redisPassword); err != nil {
			beego.Error(err)
			c.Close()
			return nil, err
		}
	}

	_, selecterr := c.Do("SELECT", dbNum)
	if selecterr != nil {
		beego.Error(selecterr)
		c.Close()
		return nil, selecterr
	}
	return
}

var Redis *redisStruct = &redisStruct{}

func init() {
	redisAddress = beego.AppConfig.String("redis::ADDRESS")
	dbNum, _ = beego.AppConfig.Int("redis::DB")
	redisPassword = beego.AppConfig.String("redis::PASSWORD")

	if redisAddress == "" {
		beego.Info("[init] redis: DISABLE!")
		return
	}
	
	maxIdleCount := beego.AppConfig.DefaultInt("redis::POOL_MAX_IDLE", 100)
	beego.Info(fmt.Sprintf("[init] redis: %s - %d, maxIdle(%d)", redisAddress, dbNum, maxIdleCount))
	// initialize a new pool
	pool = &redis.Pool{
		MaxIdle: maxIdleCount,
		IdleTimeout: 180 * time.Second,
		Dial: dialFunc,
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
	c := pool.Get()
	defer c.Close()
}