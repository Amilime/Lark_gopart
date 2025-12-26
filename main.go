package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// 1. 创建一个默认的路由引擎 (相当于 Spring Boot 的 ApplicationContext)
	r := gin.Default()
	r.MaxMultipartMemory = 30 << 20 // 文件不能大于30MB
	r.Static("/files", "./uploads")

	// 2. 只有 GET 请求根目录时，返回一个简易的 HTML 上传页面。这个是代替前端用的
	r.GET("/", func(c *gin.Context) {
		html := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<title>Go 文件服务器</title>
		</head>
		<body>
			<h1>上传文件给 Go 服务器</h1>
			<form action="/upload" method="post" enctype="multipart/form-data">
				选择文件: <input type="file" name="file" />
				<input type="submit" value="上传" />
			</form>
		</body>
		</html>
		`
		// 把这段 HTML 字符串当做网页返回给浏览器
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
	})

	// 3. 核心功能：接收上传的文件
	r.POST("/upload", func(c *gin.Context) {
		// (1) 从请求中读取文件，参数名必须和 HTML 里的 name="file" 一致
		file, err := c.FormFile("file")

		// Go 语言特色的错误处理：如果有错 (err 不等于 nil)，就报错返回
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "上传失败，没找到文件"})
			return
		}

		// (2) 准备保存的路径：当前目录下的 uploads 文件夹
		savePath := "./uploads"
		// 如果文件夹不存在，就创建它 (0755 是读写权限)
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			os.Mkdir(savePath, 0755)
		}

		// (3) 拼接完整的文件名：./uploads/图片.png
		// filepath.Base 是为了防止文件名里包含路径（安全措施）
		dst := filepath.Join(savePath, filepath.Base(file.Filename))
		c.SaveUploadedFile(file, dst)

		// 以前返回的是 ./uploads/img.png (这是硬盘路径，浏览器看不懂)
		// 现在返回 http://localhost:8081/images/img.png (这是网络链接)
		fileUrl := fmt.Sprintf("http://localhost:8081/files/%s", filepath.Base(file.Filename))

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"url":    fileUrl, // <--- 把这个链接给前端
		})
	})

	//如果不填参数默认是 ":8080"
	r.Run(":8081")
}
