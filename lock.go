package lock

import (
	"errors"
	"github.com/go-redis/redis/v8"
	uuid "github.com/satori/go.uuid"
	"log"
	"time"
)

type RedisLock struct {
	key         string        //锁key
	randomValue interface{}   //随机值
	unlockCh    chan struct{} //解锁通知通道
	client      *RedisClient  //redis客户端
}

/**
 *  @Description: 循环阻塞获取永久锁
 *  @receiver l
 *  @param block 是否阻塞获取
 *  @param expire 锁过期时间(设置0永不过期)
 *  @return bool
 *  @return error
 */
func (l *RedisLock) Lock(block bool, expire time.Duration) (bool, error) {
	var resp *redis.BoolCmd
	uuid := uuid.NewV1()
	l.randomValue = uuid
	for {
		resp = l.client.SetNX(l.client.ctx, l.key, uuid, expire) //返回执行结果
		lockSuccess, err := resp.Result()
		if err == nil && lockSuccess {
			//加锁成功,跳出循环
			return true, nil
		} else {
			if !block {
				return lockSuccess, err
			}
			time.Sleep(time.Millisecond * 30) //可以适当休眠
		}
	}
}

/**
 *  @Description: 限时阻塞获取锁
 *  @receiver l
 *  @param seconds 等待时长
 *  @param expire 锁过期时间
 *  @return bool
 *  @return error
 */
func (l *RedisLock) LockWaitSeconds(seconds int64, expire time.Duration) (bool, error) {
	if seconds <= 0 {
		return false, errors.New("seconds must grater than zero")
	}
	expTimer := time.NewTimer(time.Duration(seconds) * time.Second) //定时器
	var resp *redis.BoolCmd
	uuid := uuid.NewV1()
	l.randomValue = uuid
	for {
		select {
		case <-expTimer.C: //两秒后触发
			return false, nil
		default:
			resp = l.client.SetNX(l.client.ctx, l.key, uuid, expire) //返回执行结果
			lockSuccess, err := resp.Result()
			if err == nil && lockSuccess {
				//加锁成功
				//如果设置锁有过期时间,则通知看门狗
				if expire > 0 {
					l.unlockCh = make(chan struct{}, 0)
					go l.client.WatchDog(l.unlockCh, l.key, uuid)
				}
				return true, nil
			} else {
				time.Sleep(time.Millisecond * 30) //可以适当休眠
			}
		}
	}
}

/**
 *  @Description: 解锁
 *  @receiver l
 */
func (l *RedisLock) Unlock() {
	//若未上锁则不执行解锁操作
	script := redis.NewScript(`
    if redis.call('get', KEYS[1]) == ARGV[1]
    then 
      return redis.call('del', KEYS[1]) 
     else 
        return 0 
     end
  `)
	resp := script.Run(l.client.ctx, l.client, []string{l.key}, l.randomValue)
	if result, err := resp.Result(); err != nil || result == 0 {
		log.Println("unlock failed:", err)
	} else {
		//删锁成功后，通知看门狗退出
		if l.unlockCh != nil {
			l.unlockCh <- struct{}{}
		}
	}
}
