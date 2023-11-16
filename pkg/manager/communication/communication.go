package communication

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

func NewCommunicationController(s *storage.Engine) *CommunicationController {
	return &CommunicationController{
		store: s,
	}
}

type CommunicationController struct {
	store *storage.Engine
}

func (cc *CommunicationController) Communication(c *gin.Context) {
	t := c.Query("token")
	if t != config.Read().Manager.AgentBootstrapToken {
		c.JSON(http.StatusUnauthorized, "unauthorized access")
		return
	}
	upGrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		// generate buffer size 2MiB
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
	}
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorw("upgrade websockets error", "error", err)
		return
	}
	go cc.hanleMessage(ws)
}

// handle message from websocket
func (cc *CommunicationController) hanleMessage(ws *websocket.Conn) {
	cache.GetCommunicationManager().AddClient(ws)
	for {
		var msg cache.Message
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Warnw("read message error", "error", err)
			cache.GetCommunicationManager().RemoveClient(ws)
			return
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Errorw("unmarshal message error", "error", err, "message", string(message))
			continue
		}
		switch msg.MessageType {
		case cache.Heartbeat:
			name := msg.Name
			log.Debugw("receive heartbeat message", "cluster", name)
			if err := cc.handleheartbeat(context.Background(), name); err != nil {
				log.Errorw("heartbeat failed", "error", err, "cluster", name)
				continue
			}
		case cache.FirstConnect:
			ctx := context.Background()
			var req web.Request
			req.Default()
			if msg.Kind == "cluster" {
				if err := cc.handleCluster(ctx, &req, ws); err != nil {
					log.Errorw("first connected to cluster failed", "error", err)
					continue
				}
			}
			if msg.Kind == "application" {
				if err := cc.handleApplication(ctx, &req, ws); err != nil {
					log.Errorw("first connected to application failed", "error", err)
					continue
				}
			}
			cache.GetCommunicationManager().ChangeClientStatus(ws, true)
		case cache.Normal:
			ctx := context.Background()
			if msg.Kind == "cluster" {
				if err := cc.getCluster(ctx, msg.ID, msg.Name, ws); err != nil {
					log.Errorw("get cluster failed", "error", err, "cluster", msg.Name, "id", msg.ID)
					continue
				}
			}
			if msg.Kind == "application" {
				if err := cc.getApplication(ctx, msg.ID, ws); err != nil {
					log.Errorw("get application failed", "error", err, "id", msg.ID)
					continue
				}
			}
		}
	}
}

func (cc *CommunicationController) handleheartbeat(ctx context.Context, name string) error {
	var old cluster.Cluster
	log.Debugw("receive heartbeat message", "cluster", name)
	if err := cc.store.Get(ctx, 0, name, false, &old); err != nil {
		return err
	}
	old.LastKeepAliveTime = time.Now().Local()
	return cc.store.Update(ctx, 0, name, &old, &old)
}

func (cc *CommunicationController) handleCluster(ctx context.Context, req *web.Request, ws *websocket.Conn) error {
	var clss []cluster.Cluster
	_, err := cc.store.List(ctx, req.Limit, req.Page, "", &clss)
	if err != nil {
		return err
	}
	for _, v := range clss {
		m := &cache.Message{
			ID:          v.ID,
			Kind:        "cluster",
			Name:        v.Name,
			MessageType: cache.FirstConnect,
			ActionType:  cache.CreateAction,
		}
		ws.WriteJSON(m)
	}
	return nil
}

func (cc *CommunicationController) handleApplication(ctx context.Context, req *web.Request, ws *websocket.Conn) error {
	var apps []application.Application
	_, err := cc.store.List(ctx, req.Limit, req.Page, "", &apps)
	if err != nil {
		return err
	}
	for _, v := range apps {
		m := &cache.Message{
			ID:          v.ID,
			Kind:        "application",
			Name:        v.Name,
			MessageType: cache.FirstConnect,
			ActionType:  cache.CreateAction,
		}
		ws.WriteJSON(m)
	}
	return nil
}

func (cc *CommunicationController) getCluster(ctx context.Context, id uint, name string, ws *websocket.Conn) error {
	var cls cluster.Cluster
	if err := cc.store.Get(ctx, id, name, false, &cls); err != nil {
		return err
	}
	ws.WriteJSON(&cls)
	return nil
}

func (cc *CommunicationController) getApplication(ctx context.Context, id uint, ws *websocket.Conn) error {
	var app application.Application
	if err := cc.store.Get(ctx, id, "", false, &app); err != nil {
		return err
	}
	ws.WriteJSON(&app)
	return nil
}
