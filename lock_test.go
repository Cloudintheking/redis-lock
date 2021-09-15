package lock_test

import (
	"context"
	"fmt"
	"github.com/Cloudintheking/redis-lock"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

var ctx = context.Background()

func TestLock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	client := lock.NewRedisClient(context.Background(), rdb)
	lock := client.GetLock("test", 5)

	//阻塞获取
	fmt.Println("开始限时获取锁,timestamp:", time.Now().Unix())
	if ok, err := lock.LockWaitSeconds(5, 15*time.Second); !ok {
		fmt.Println("限时获取锁失败,timestamp:", time.Now().Unix(), err)
	} else {
		fmt.Println("限时获取锁成功,timestamp:", time.Now().Unix())
		time.Sleep(30 * time.Second)
		lock.Unlock()
		fmt.Println("删除锁成功,timestamp:", time.Now().Unix())
	}
}
