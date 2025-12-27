package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"lark/ws" // 【注意】这里 larks/ws 取决于你的 go.mod 里的 module 名字
	// 如果你的 module 叫 lark，这里就是 lark/ws
	// 如果报错，请看文件最上面的 package import 提示
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// 1. 创建并启动 Hub (大管家)和连接redis
	// 这一步如果不做，后面没人处理消息
	ws.InitRedis()

	hub := ws.NewHub()
	go hub.Run() // 让它在后台一直跑

	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20

	// 静态文件服务
	r.Static("/files", "./uploads")
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Go WebSocket Server is Running...")
	})

	// 文件上传
	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "上传失败"})
			return
		}
		savePath := "./uploads"
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			os.Mkdir(savePath, 0755)
		}
		dst := filepath.Join(savePath, filepath.Base(file.Filename))
		c.SaveUploadedFile(file, dst)
		fileUrl := fmt.Sprintf("http://localhost:8081/files/%s", filepath.Base(file.Filename))
		c.JSON(http.StatusOK, gin.H{"status": "success", "url": fileUrl})
	})

	// 【关键】WebSocket 路由
	// 把请求交给 ws 包里的 ServeWs 处理
	r.GET("/ws", func(c *gin.Context) { //传gin.context指针（有点像数组首地址传数组)
		ws.ServeWs(hub, c)
	})

	fmt.Println(" Go 服务启动在 :8081 ...")
	r.Run(":8081")
}
