package ws

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
)

// 升级器配置
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client 代表一个单独的用户连接
type HubClient struct {
	Hub    *Hub            // 归哪个大管家管
	Conn   *websocket.Conn // 真正的 WebSocket 连接
	Send   chan []byte     // 自己的发信箱（Hub 会往这里塞数据）
	UserId int64           // 用于验证身份的用户ID
}

// 1. 读泵 (从浏览器读 -> 发给 Hub)
func (c *HubClient) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c // 断开时通知 Hub 注销
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		// 把读到的消息塞给 Hub 进行广播
		c.Hub.Broadcast <- message
	}
}

// 2. 写泵 (从 Send 通道读 -> 发给浏览器)
func (c *HubClient) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		// 等待 Hub 给自己发邮件
		message, ok := <-c.Send
		if !ok {
			// 通道被关闭了
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// 写回浏览器
		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)

		// 这一步是为了把积压的消息一次性发完（优化）
		n := len(c.Send)
		for i := 0; i < n; i++ {
			w.Write(<-c.Send)
		}

		if err := w.Close(); err != nil {
			return
		}
	}
}

// ServeWs 处理 WebSocket 请求
func ServeWs(hub *Hub, c *gin.Context) {

	token := c.Query("token")
	if token == "" {
		fmt.Println("❌ 拒绝连接：没有 Token") // 打印日志
		c.JSON(401, gin.H{"error": "未携带 Token"})
		return
	}

	claims, err := ParseToken(token)
	if err != nil {
		// 【关键】打印具体的错误原因！
		fmt.Println("❌ Token 验证失败，原因:", err)
		c.JSON(401, gin.H{"error": "无效的 Token: " + err.Error()})
		return
	}

	// 1. 从 URL 参数获取 Token
	// 格式: ws://localhost:8081/ws?token=xxxxx

	// 3. 升级 HTTP 为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// 4. 创建客户端 (带上 UserId)
	client := &HubClient{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserId: claims.Uid, // 记录下这个人是谁
	}

	// 注册到 Hub
	client.Hub.Register <- client

	// 启动协程
	go client.WritePump()
	go client.ReadPump()
}
