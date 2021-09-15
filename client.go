package lock

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"log"
	"time"
)

type RedisClient struct {
	*redis.Client //redis客户端
	ctx           context.Context
}

/**
 *  @Description: 初始化redis客户端
 *  @param ctx
 *  @param client
 *  @return *RedisClient
 */
func NewRedisClient(ctx context.Context, client *redis.Client) *RedisClient {
	return &RedisClient{
		Client: client,
		ctx:    ctx,
	}
}

/**
 *  @Description: 获取普通锁
 *  @receiver r
 *  @param key 锁
 *  @param renewSec 续约时长
 *  @return *RedisLock
 */
func (r *RedisClient) GetLock(key string, renewSec int64) *RedisLock {
	return &RedisLock{
		client:   r,
		key:      key,
		renewSec: renewSec,
	}
}

/**
 *  @Description: 锁续期
 *  @receiver r
 *  @param unlockCh
 *  @param key
 *  @param random
 */
func (r *RedisClient) WatchDog(unlockCh chan struct{}, renewSec int64, key string, random uuid.UUID) {
	defer func() {
		if e := recover(); e != nil {
		}
	}()
	//确认锁与锁续期打包原子化
	script := redis.NewScript(`
    if redis.call('get', KEYS[1]) == ARGV[1]
    then 
      return redis.call('expire', KEYS[1], ARGV[2]) 
    else 
      return 0 
    end
  `)
	for {
		select {
		case <-unlockCh: //任务完成后用户解锁通知看门狗退出
			fmt.Println("unlockNotify", key)
			return
		default:
			time.Sleep(time.Duration(renewSec/2) * time.Second)
			resp := script.Run(r.ctx, r, []string{key}, random, renewSec)
			if result, err := resp.Result(); err != nil || result == int64(0) {
				//续约失败
				log.Println("expire lock", key, renewSec, "sec failed", err)
			} else {
				//续约成功
				log.Println("expire lock", key, renewSec, "sec success")
			}
		}
	}
}
