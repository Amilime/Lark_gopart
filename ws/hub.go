package ws

import (
	"fmt"
)

// 广播消息的结构体：不仅包含内容，还包含“送到哪去”
// 这里的逻辑实现是并发yjs精华，也是逻辑最难受的地方，对于redis要怎么存怎么取（光标和握手不能存储）
// 实现了增量持久化
type BroadcastMsg struct {
	RoomID string // 也就是 DocID
	Data   []byte
	Sender *HubClient // 发送者
}

type Hub struct {
	Rooms map[string]map[*HubClient]bool

	Register   chan *HubClient
	Unregister chan *HubClient

	Broadcast chan *BroadcastMsg
	quit      chan bool // 关闭信号
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]map[*HubClient]bool),
		Register:   make(chan *HubClient),
		Unregister: make(chan *HubClient),
		Broadcast:  make(chan *BroadcastMsg),
		quit:       make(chan bool),
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
			//lastContent := GetDoc(client.DocID) //把Redis旧数据发给新人
			//if lastContent != "" {
			//	// 单独发给这个人
			//	client.Send <- []byte(lastContent)
			//}
			fmt.Printf("用户进入房间 [%s]，当前房间人数: %d\n", client.DocID, len(h.Rooms[client.DocID]))
			history := GetYjsHistory(client.DocID)
			for _, update := range history {
				// 挨个发送历史 update
				client.Send <- update
			}

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
			// ️ 修改：只存真正的文档更新
			if len(msg.Data) > 0 {
				// Yjs 协议：0=Sync, 1=Awareness
				if len(msg.Data) > 0 {
					msgType := msg.Data[0]
					if msgType == 0 && len(msg.Data) > 1 && msg.Data[1] == 2 {
						// 只要收到更新，立刻存盘！不管房间里有几个人  这里顺序逻辑非常关键！！！！！！！！！
						SaveYjsUpdate(msg.RoomID, msg.Data)
						// fmt.Println(">>> [Hub] 独立存储成功")
					}
				}
				// 注意：Awareness (msgType==1) 和 握手请求 (step==0/1) 都不存
			}

			//
			// 虽然不存，但所有消息都要广播出去，否则光标不动，握手无法完成
			if room, ok := h.Rooms[msg.RoomID]; ok {
				for client := range room {
					if client == msg.Sender {
						continue
					}
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
