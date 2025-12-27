package ws

import (
	"fmt"
)

// 广播消息的结构体：不仅包含内容，还包含“送到哪去”
type BroadcastMsg struct {
	RoomID string // 也就是 DocID
	Data   []byte
	Sender *HubClient // 发送者
}

type Hub struct {
	// 【关键修改】不再是 Clients map[*HubClient]bool
	// 而是 Rooms: Key是房间号(DocID), Value是这个房间里的人
	Rooms map[string]map[*HubClient]bool

	Register   chan *HubClient
	Unregister chan *HubClient

	// 【关键修改】广播通道接收的是 BroadcastMsg 结构体
	Broadcast chan *BroadcastMsg
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]map[*HubClient]bool),
		Register:   make(chan *HubClient),
		Unregister: make(chan *HubClient),
		Broadcast:  make(chan *BroadcastMsg),
	}
}

func (h *Hub) Run() { // h的意思是把Run绑定到Hub上，就是后面h就代表hub了，和java的this差不多
	for {
		select {
		// 1. 有人进房
		case client := <-h.Register:
			// 如果房间不存在，先造一个房间
			if _, ok := h.Rooms[client.DocID]; !ok {
				h.Rooms[client.DocID] = make(map[*HubClient]bool)
			}
			// 把人放进房间
			h.Rooms[client.DocID][client] = true
			lastContent := GetDoc(client.DocID) //把Redis旧数据发给新人
			if lastContent != "" {
				// 单独发给这个人
				client.Send <- []byte(lastContent)
			}
			fmt.Printf("用户进入房间 [%s]，当前房间人数: %d\n", client.DocID, len(h.Rooms[client.DocID]))

		// 2. 有人退房
		case client := <-h.Unregister:
			if room, ok := h.Rooms[client.DocID]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.Send)
					fmt.Printf("用户离开房间 [%s]，剩余人数: %d\n", client.DocID, len(room))

					// 如果房间空了，可以销毁房间（省内存）
					if len(room) == 0 {
						delete(h.Rooms, client.DocID)
					}
				}
			}

		// 3. 广播消息
		case msg := <-h.Broadcast:
			SaveDoc(msg.RoomID, msg.Data) // 消息存入redis
			// 只找特定房间的人
			if room, ok := h.Rooms[msg.RoomID]; ok {
				for client := range room {
					if client == msg.Sender {
						continue
					} // 别给自己发
					select {
					case client.Send <- msg.Data:
					default:
						close(client.Send)
						delete(room, client)
					}
				}
			}
		}
	}
}
