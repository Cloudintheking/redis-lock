package lock

import (
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"log"
	"time"
)

type RedisLock struct {
	key         string        //锁key
	randomValue interface{}   //随机值
	unlockCh    chan struct{} //解锁通知通道
	client      *RedisClient  //redis客户端
	renewSec    int64         //续约时长, <1不续约,尽量大于锁过期时长,避免频繁续约
}

/**
 *  @Description: 设置续约时长
 *  @receiver l
 *  @param renewSec
 */
func (l *RedisLock) Renew(renewSec int64) *RedisLock {
	if renewSec > 0 {
		l.renewSec = renewSec
		return l
	} else {
		panic("renewSec should grater than zero")
	}

}

/**
 *  @Description: 上锁
 *  @receiver l
 *  @param block 是否阻塞获取
 *  @param expire 锁过期时间(设置0永不过期)
 *  @return bool
 *  @return error
 */
func (l *RedisLock) Lock(block bool, expire time.Duration) (bool, error) {
	var resp *redis.BoolCmd
	uuid, _ := uuid.NewRandom()
	l.randomValue = string(uuid[:])
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
 *  @Description: 限时上锁
 *  @receiver l
 *  @param waitSeconds 等待时长秒
 *  @param expireTime 锁过期时长
 *  @return bool
 *  @return error
 */
func (l *RedisLock) LockWaitSeconds(waitSeconds int64, expireTime time.Duration) (bool, error) {
	if waitSeconds <= 0 {
		return false, errors.New("waitSeconds must grater than zero")
	}
	if expireTime*3 <= time.Duration(l.renewSec)*time.Second {
		return false, errors.New("expireTime must grater than renewSec/3")
	}
	expTimer := time.NewTimer(time.Duration(waitSeconds) * time.Second) //定时器
	var resp *redis.BoolCmd
	uuid, _ := uuid.NewRandom()
	l.randomValue = uuid
	for {
		select {
		case <-expTimer.C: //waitSeconds秒后触发
			return false, nil
		default:
			resp = l.client.SetNX(l.client.ctx, l.key, uuid, expireTime) //返回执行结果
			lockSuccess, err := resp.Result()
			if err == nil && lockSuccess {
				//加锁成功
				//只有当锁过期时长>0且续约时长>1时,通知看门狗续约
				if expireTime > 0 && l.renewSec > 1 {
					l.unlockCh = make(chan struct{}, 0)
					go l.client.WatchDog(l.unlockCh, l.key, uuid, l.renewSec)
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
		log.Println(l.key, " unlock failed:", err)
	} else {
		//删锁成功后，通知看门狗退出
		log.Println(l.key, " unlock success")
		if l.unlockCh != nil {
			l.unlockCh <- struct{}{}
		}
	}
}
