package server

import (
	"encoding/json"
	"time"

	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/server/service"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/rancher/go-rancher/api"
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
			switch v := message.Data.(type) {
			case *model.Activity:
				if !service.ValidAccountAccessById(uid, v.Pipeline.Stages[0].Steps[0].GitUser) {
					continue
				}
				model.ToActivityResource(apiContext, v)
				if v.CanApprove(uid) {
					//add approve action
					v.Actions["approve"] = apiContext.UrlBuilder.ReferenceLink(v.Resource) + "?action=approve"
					v.Actions["deny"] = apiContext.UrlBuilder.ReferenceLink(v.Resource) + "?action=deny"
				}
				message.Data = v
			case *model.Pipeline:
				if !service.ValidAccountAccessById(uid, v.Stages[0].Steps[0].GitUser) {
					continue
				}
				model.ToPipelineResource(apiContext, v)
				message.Data = v
			case *model.GitAccount:
				if v.RancherUserID != uid && v.Private {
					continue
				}
				model.ToAccountResource(apiContext, v)
				message.Data = v
			case *model.PipelineSetting:
				model.ToPipelineSettingResource(apiContext, v)
				message.Data = v
			case *model.SCMSetting:
				model.ToSCMSettingResource(apiContext, v)
				message.Data = v
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
