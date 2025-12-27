package ws

import (
	"fmt"
)

type Hub struct {
	Register   chan *HubClient
	Unregister chan *HubClient
	Broadcast  chan []byte
	Clients    map[*HubClient]bool
	// 注册通道，新人指针在里头  注销通道，断开指针去这里 广播，有人说话就在这里 在线名单，所有连接map
}

func NewHub() *Hub { // 创建新的Hub中心，该函数类型是*Hub （即返回Hub)
	return &Hub{
		Register:   make(chan *HubClient), //channel是通道，该通道只运送HubClient类型的指针（结构体）
		Unregister: make(chan *HubClient), // Main协程在头部把用户塞管子里，Hub协程在尾部取用户
		Broadcast:  make(chan []byte),
		Clients:    make(map[*HubClient]bool),
	}
}

func (h *Hub) Run() { // h的意思是把Run绑定到Hub上，就是后面h就代表hub了，和java的this差不多
	for {
		// select 是 Go 处理并发的神技，谁有消息处理谁
		select {
		// 有人注册
		case client := <-h.Register: // 这个代码是会阻塞的，从h.register取出一个值赋值给client
			h.Clients[client] = true
			fmt.Println("用户加入，当前在线人数:", len(h.Clients))

		// 有人注销
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send) // 关闭发送通道
				fmt.Println("用户离开，当前在线人数:", len(h.Clients))
			}

		// 有消息要广播
		case message := <-h.Broadcast:
			// 遍历所有人，发消息
			for client := range h.Clients {
				select {
				case client.Send <- message:
					// 发送成功
				default:
					// 发送失败（比如对方卡死了），直接踢掉
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}
