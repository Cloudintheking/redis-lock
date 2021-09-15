package lock

import (
	"context"
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
 *  @return *RedisLock
 */
func (r *RedisClient) GetLock(key string) *RedisLock {
	return &RedisLock{
		client: r,
		key:    key,
	}
}

/**
 *  @Description: 锁续约
 *  @receiver r
 *  @param unlockCh 解锁通知通道
 *  @param key 锁key
 *  @param random 锁值
 *  @param renewSec 锁续约时长
 */
func (r *RedisClient) WatchDog(unlockCh chan struct{}, key string, random uuid.UUID, renewSec int64) {
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
		case <-unlockCh:
			//收到解锁通知,看门狗退出
			log.Println(" receive unlockNotify", key, ",watch dog out")
			return
		default:
			time.Sleep(time.Duration(renewSec/3) * time.Second)
			resp := script.Run(r.ctx, r, []string{key}, random, renewSec)
			if result, err := resp.Result(); err != nil || result == int64(0) {
				//续约失败
				log.Println("renew lock", key, renewSec, "sec failed", err)
			} else {
				//续约成功
				log.Println("renew lock", key, renewSec, "sec success")
			}
		}
	}
}
