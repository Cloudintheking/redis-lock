package lock

import (
	"context"
	"github.com/go-redis/redis/v8"
	uuid "github.com/satori/go.uuid"
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
 *  @param key
 *  @return *RedisLock
 */
func (r *RedisClient) GetLock(key string) *RedisLock {
	return &RedisLock{
		client:   r,
		key:      key,
	}
}

/**
 *  @Description: 锁续期
 *  @receiver r
 *  @param unlockCh
 *  @param key
 *  @param random
 */
func (r *RedisClient) WatchDog(unlockCh chan struct{}, key string, random uuid.UUID) {
	// 创建一个触发器NewTicker, 每隔2秒触发一次,类似于闹钟
	expTicker := time.NewTicker(time.Second * 2)
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
		case <-expTicker.C: //每隔2s触发一次
			resp := script.Run(r.ctx, r, []string{key}, random, 10)
			if result, err := resp.Result(); err != nil || result == int64(0) {
				//续期失败
				log.Println("expire lock failed", err)
			}
		case <-unlockCh: //任务完成后用户解锁通知看门狗退出
			return
		}
	}
}
