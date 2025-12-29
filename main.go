package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"lark/ws"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// 1. 初始化
	ws.InitRedis()
	ws.InitMySQL() // 必须初始化 MySQL
	hub := ws.NewHub()
	go hub.Run()

	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true // 允许所有来源 (开发环境方便)

	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// 静态资源映射
	r.Static("/files", "./uploads")

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Go Server Running...")
	})
	
	r.POST("/upload", func(c *gin.Context) {
		// 1. 获取上传者信息 (Token)
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}

		claims, err := ws.ParseToken(token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效 Token"})
			return
		}

		// 2. 接收文件
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": "文件接收失败"})
			return
		}

		// 3. 存磁盘
		savePath := "./uploads"
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			os.Mkdir(savePath, 0755)
		}
		filename := filepath.Base(file.Filename)
		dst := filepath.Join(savePath, filename)
		c.SaveUploadedFile(file, dst)

		// 生成 URL
		fileUrl := fmt.Sprintf("http://localhost:8081/files/%s", filename)

		// 4. 【关键】存入 MySQL 数据库
		// 这样 Java 的列表接口才能查到这个文件！
		docId, dbErr := ws.CreateStaticDocument(filename, fileUrl, claims.Uid)

		if dbErr != nil {
			c.JSON(500, gin.H{"error": "数据库保存失败"})
			return
		}

		c.JSON(200, gin.H{
			"status": "success",
			"url":    fileUrl,
			"docId":  docId, // 返回数据库 ID
			"type":   0,     // 0=静态文件
		})
	})

	r.POST("/api/version/save", func(c *gin.Context) {
		type Req struct {
			DocId string `json:"docId"`
			Uid   int64  `json:"userId"`
			Ver   int    `json:"versionNum"`
		}
		var req Req
		c.ShouldBindJSON(&req)

		err := ws.CreateVersionSnapshot(req.DocId, req.Uid, req.Ver)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"msg": "Saved"})
	})

	// WebSocket
	r.GET("/ws/:room", func(c *gin.Context) {
		ws.ServeWs(hub, c)
	})

	r.Run(":8081")
}
