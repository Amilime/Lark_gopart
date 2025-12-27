package ws

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	Rdb *redis.Client
	Ctx = context.Background()
)

func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	pong, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		fmt.Println("连接失败", err)
		panic(err)
	}
	fmt.Println("成功:", pong)
}

// 文档快照
func SaveDoc(docId string, content []byte) {
	err := Rdb.Set(Ctx, "doc:"+docId, content, 24*time.Hour).Err()
	if err != nil {
		fmt.Println("Redis 写入失败:", err)
	}
}
func GetDoc(docId string) string {
	val, err := Rdb.Get(Ctx, "doc:"+docId).Result()
	if err == redis.Nil {
		return "" // 如果不存在，返回空字符串
	} else if err != nil {
		fmt.Println("Redis 读取失败:", err)
		return ""
	}
	return val
}
