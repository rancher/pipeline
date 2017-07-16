package restfulserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/rancher/go-rancher/api"
	"github.com/sluu99/uuid"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type ConnHolder struct {
	agent *Agent

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *ConnHolder) DoRead() {
	logrus.Infof("start ws reader")
	defer func() {
		c.agent.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *ConnHolder) DoWrite(apiContext *api.ApiContext) {
	logrus.Infof("start ws writer")
	pingTicker := time.NewTicker(pingPeriod)
	pollTicker := time.NewTicker(pollPeriod)
	defer func() {
		pingTicker.Stop()
		pollTicker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			logrus.Infof("get message from c.send,%v", string(message))
			var b []byte
			var err error
			activities, err := ListActivities(c.agent.Server.PipelineContext)
			for _, activity := range activities {
				toActivityResource(apiContext, activity)
			}
			response := WSMsg{
				Id:           uuid.Rand().Hex(),
				Name:         "resource.change",
				ResourceType: "activity",
				Time:         time.Now(),
				Data:         activities,
			}
			b, err = json.Marshal(response)
			if err != nil {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		case <-pingTicker.C:
			logrus.Infof("trying to ping")
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
				logrus.Errorf("error writing ping,%v", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, PingMsg()); err != nil {
				logrus.Errorf("error writing ping,%v", err)
				return
			}
		}
	}
}

func (s *Server) ServeStatusWS(w http.ResponseWriter, r *http.Request) error {
	logrus.Infof("start ws")
	apiContext := api.GetApiContext(r)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logrus.Errorf("ws handshake error")
		}
		return err
	}
	connHolder := &ConnHolder{agent: MyAgent, conn: conn, send: make(chan []byte, 256)}

	connHolder.agent.register <- connHolder

	//new go routines
	go connHolder.DoWrite(apiContext)
	connHolder.DoRead()

	return nil
}
