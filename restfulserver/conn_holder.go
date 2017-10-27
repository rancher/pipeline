package restfulserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/pipeline/pipeline"
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
	send chan WSMsg
}

func (c *ConnHolder) DoRead() {
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

func (c *ConnHolder) DoWrite(apiContext *api.ApiContext, uid string) {
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
			activity, ok := message.Data.(pipeline.Activity)
			if ok {
				toActivityResource(apiContext, &activity)
				if canApprove(uid, &activity) {
					//add approve action
					activity.Actions["approve"] = apiContext.UrlBuilder.ReferenceLink(activity.Resource) + "?action=approve"
					activity.Actions["deny"] = apiContext.UrlBuilder.ReferenceLink(activity.Resource) + "?action=deny"
				}
				message.Data = activity
			}
			b, err := json.Marshal(message)
			if err != nil {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		case <-pingTicker.C:
			//logrus.Infof("trying to ping")
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
	apiContext := api.GetApiContext(r)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logrus.Errorf("ws handshake error")
		}
		return err
	}
	uid, err := GetCurrentUser(r.Cookies())
	//logrus.Infof("got currentUser,%v,%v", uid, err)
	if err != nil || uid == "" {
		logrus.Errorf("get currentUser fail,%v,%v", uid, err)
	}
	connHolder := &ConnHolder{agent: MyAgent, conn: conn, send: make(chan WSMsg)}

	connHolder.agent.register <- connHolder

	//new go routines
	go connHolder.DoWrite(apiContext, uid)
	connHolder.DoRead()

	return nil
}
