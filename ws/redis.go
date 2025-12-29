package ws

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	Rdb *redis.Client
	ctx = context.Background()
)

func InitRedis() {
	fmt.Println("-------------------------------------------")
	fmt.Println(">>> 正在尝试连接 Redis (localhost:6379) ...")
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := Rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Println("连接失败", err)
		panic(err)
	}
	fmt.Println("Redis连接成功:")
}

// 文档快照
func SaveYjsUpdate(docId string, updateData []byte) {
	// 这里的 docId 就是 main.go 传过来的 room，名字不同没关系
	key := "doc:" + docId

	// 🔍【监控日志】看看有没有正在存？
	fmt.Printf("Redis写入 Key=%s, 数据长度=%d \n", key, len(updateData))

	err := Rdb.RPush(ctx, key, updateData).Err()
	if err != nil {
		fmt.Println(" Redis 存储失败:", err)
	}

	// 续期
	Rdb.Expire(ctx, key, 24*time.Hour)
}

func GetYjsHistory(docId string) [][]byte {
	key := "doc:" + docId

	// 🔍【监控日志】看看有没有正在读？
	fmt.Printf("Redis读取 Key=%s \n", key)

	strResults, err := Rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		fmt.Println("Redis 读取失败:", err)
		return nil
	}

	fmt.Printf(" Redis读取 成功 读到了 %d 条历史记录 \n", len(strResults))

	var updates [][]byte
	for _, s := range strResults {
		updates = append(updates, []byte(s))
	}
	return updates
}

//func GetDoc(docId string) string {
//	val, err := Rdb.Get(Ctx, "doc:"+docId).Result()
//	if err == redis.Nil {
//		return "" // 如果不存在，返回空字符串
//	} else if err != nil {
//		fmt.Println("Redis 读取失败:", err)
//		return ""
//	}
//	return val
//}
