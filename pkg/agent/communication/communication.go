package communication

import (
	"context"
	"encoding/json"
	"net/url"
	"sync"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/gorilla/websocket"
	"github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/utils/config"
)

var (
	conn          *websocket.Conn
	globalMsgChan chan interface{}
	syncOnce      sync.Once
)

// InitCommunication init websocket communication
func InitCommunication(next chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	InitChannel()
	if err := Communication(ctx); err != nil {
		log.Fatalw("communication error", "error", err)
	}
	next <- struct{}{}
	for {
		time.Sleep(10 * time.Second)
	}
}

func InitChannel() {
	syncOnce.Do(func() {
		globalMsgChan = make(chan interface{}, 2000)
	})
}

func GetChan() chan interface{} {
	return globalMsgChan
}

func Communication(ctx context.Context) error {
	var err error
	u := config.Read().Agent.ManagerAddress + "/api/v1/communication?token=" + config.Read().Agent.BootstrapToken
	tmp, err := url.Parse(u)
	if err != nil {
		return err
	}
	if tmp.Scheme == "https" {
		tmp.Scheme = "wss"
	} else {
		tmp.Scheme = "ws"
	}
	u = tmp.String()
	conn, _, err = websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return err
	}
	retry := false
	messageCh := make(chan []byte)
	errCh := make(chan error)
	go func() {
		for {
		RECONNECTED:
			if retry {
				if err := retryConn(u); err != nil {
					log.Errorw("websocket dial error", "error", err)
					time.Sleep(time.Second * 3)
					continue
				}
				retry = false
			}
			go read(messageCh, errCh)
			for {
				select {
				case <-ctx.Done():
					conn.Close()
					log.Info("websocket disconnect")
					return
				case err := <-errCh:
					log.Errorw("read websocket content with", "error", err)
					retry = true
					goto RECONNECTED
				case message := <-messageCh:
					var msg cache.Message
					if err := json.Unmarshal(message, &msg); err != nil {
						log.Errorw("unmarshal message error", "error", err)
						continue
					}
					globalMsgChan <- msg
					continue
				}
			}
		}
	}()
	return nil
}

func retryConn(u string) (err error) {
	conn, _, err = websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return err
	}
	return nil
}

func read(messageCh chan []byte, errCh chan error) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}
		messageCh <- message
	}
}

func SendMsg(msg *cache.Message) error {
	if err := conn.WriteJSON(msg); err != nil {
		return err
	}
	return nil
}
