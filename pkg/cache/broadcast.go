package cache

import (
	"reflect"
	"sync"

	"github.com/fize/go-ext/log"
	"github.com/gorilla/websocket"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/models/cluster"
)

var (
	once                 sync.Once
	communicationManager *Broadcast
)

func InitCommunicationManager() {
	once.Do(func() {
		communicationManager = NewBroadcast()
		go communicationManager.BroadcastMessage()
	})
}

func GetCommunicationManager() *Broadcast {
	return communicationManager
}

func NewBroadcast() *Broadcast {
	b := &Broadcast{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan interface{}),
		locker:    new(sync.Mutex),
	}
	return b
}

// Broadcast is a struct to handle the broadcast of messages
type Broadcast struct {
	clients   map[*websocket.Conn]bool
	broadcast chan interface{}
	locker    *sync.Mutex
}

func (b *Broadcast) AddClient(ws *websocket.Conn) {
	b.locker.Lock()
	num := 1000
	if len(b.clients) > num {
		log.Errorw("too many clients connected, close the connection", "num", len(b.clients))
		ws.Close()
	} else {
		b.clients[ws] = false
	}
	b.locker.Unlock()
}

func (b *Broadcast) RemoveClient(ws *websocket.Conn) {
	b.locker.Lock()
	delete(b.clients, ws)
	b.locker.Unlock()
}

func (b *Broadcast) ChangeClientStatus(ws *websocket.Conn, status bool) {
	b.locker.Lock()
	b.clients[ws] = status
	b.locker.Unlock()
}

func (b *Broadcast) GetClientStatus(ws *websocket.Conn) bool {
	return b.clients[ws]
}

func (b *Broadcast) AddMessage(action ActionType, msg interface{}) {
	v := reflect.ValueOf(msg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	a := v.Type()
	switch msg := msg.(type) {
	case *cluster.Cluster:
		m := &Message{
			ID:          msg.ID,
			Kind:        "cluster",
			Name:        msg.Name,
			MessageType: Normal,
			ActionType:  action,
		}
		b.broadcast <- m
	case *application.Application:
		m := &Message{
			ID:          msg.ID,
			Kind:        "application",
			Cluster:     msg.Cluster,
			Namespace:   msg.Namespace,
			Name:        msg.Name,
			MessageType: Normal,
			ActionType:  action,
		}
		b.broadcast <- m
	default:
		log.Errorw("unknown message type", "type", a.Name())
	}
}

func (b *Broadcast) BroadcastMessage() {
	for {
		msg := <-b.broadcast
		for client := range b.clients {
			// if client status is false, skip it
			if !b.clients[client] {
				continue
			}
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				b.RemoveClient(client)
			}
		}
	}
}

// Message is the Schema for the messages to broadcast
type Message struct {
	ID          uint        `json:"id,omitempty"`
	Kind        string      `json:"kind,omitempty"`
	Name        string      `json:"name,omitempty"`
	Namespace   string      `json:"namespace,omitempty"`
	Cluster     string      `json:"cluster,omitempty"`
	MessageType MessageType `json:"messageType,omitempty"`
	ActionType  ActionType  `json:"actionType,omitempty"`
}

type MessageType string

const (
	FirstConnect MessageType = "firstConnect"
	Normal       MessageType = "normal"
	Heartbeat    MessageType = "heartbeat"
	Proxy        MessageType = "proxy"
)

type ActionType string

const (
	// CreateAction is the action type for create
	CreateAction ActionType = "create"
	// UpdateAction is the action type for update
	UpdateAction ActionType = "update"
	// DeleteAction is the action type for delete
	DeleteAction ActionType = "delete"
	// GetAction is the action type for get
	GetAction ActionType = "get"
)
