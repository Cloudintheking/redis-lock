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
	lock := client.GetLock("test")

	//阻塞获取
	fmt.Println("开始阻塞获取,timestamp:", time.Now().Unix())
	if ok, err := lock.Lock(true, 5*time.Second); !ok {
		fmt.Println("阻塞获取锁失败", err)
	} else {
		fmt.Println("阻塞获取锁成功,timestamp:", time.Now().Unix())
	}
}
