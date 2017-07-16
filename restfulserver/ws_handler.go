// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package restfulserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/sluu99/uuid"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 20 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = 5 * time.Second

	// Poll step log for changes with this period.
	pollPeriod = 2 * time.Second

	syncPeriod = 10 * time.Second
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type WSMsg struct {
	Id           string      `json:"id"`
	Name         string      `json:"name"`
	ResourceId   string      `json:"resourceId"`
	ResourceType string      `json:"resourceType"`
	Data         interface{} `json:"data"`
	Time         time.Time   `json:"time"`
}

func PingMsg() []byte {
	msg := WSMsg{
		Id:   uuid.Rand().Hex(),
		Name: "ping",
		Time: time.Now(),
	}
	b, _ := json.Marshal(msg)
	return b
}

func getActivityLog(activityId string, stepOrdinal int) ([]byte, error) {

	log := "testing log," + time.Now().String()
	return []byte(log), nil
}

func stepLogReader(ws *websocket.Conn) {
	logrus.Infof("start ws reader")
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *Server) stepLogWriter(ws *websocket.Conn, activityId string, stageOrdinal int, stepOrdinal int) {
	logrus.Infof("start ws writer")
	pingTicker := time.NewTicker(pingPeriod)
	pollTicker := time.NewTicker(pollPeriod)
	defer func() {
		pingTicker.Stop()
		pollTicker.Stop()
		ws.Close()
	}()
	activity, err := GetActivity(activityId, s.PipelineContext)
	if err != nil {
		return
	}
	prevLog := ""
	for {
		select {
		case <-pollTicker.C:
			var b []byte
			var err error

			stepLog, err := s.PipelineContext.Provider.GetStepLog(&activity, stageOrdinal, stepOrdinal)
			if err != nil {
				logrus.Errorf("error get steplog,%v", err)
				return
			}
			if stepLog != "" && prevLog != stepLog {
				logrus.Infof("writing step log:%v", stepLog)
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				response := WSMsg{
					Id:           uuid.Rand().Hex(),
					Name:         "resource.change",
					ResourceType: "log",
					Time:         time.Now(),
					Data:         stepLog,
				}
				b, err = json.Marshal(response)
				if err := ws.WriteMessage(websocket.TextMessage, b); err != nil {
					return
				}
				prevLog = stepLog
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
				logrus.Errorf("error writing ping,%v", err)
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, PingMsg()); err != nil {
				logrus.Errorf("error writing ping,%v", err)
				return
			}
		}
	}
}

func (s *Server) ServeStepLog(w http.ResponseWriter, r *http.Request) error {
	logrus.Infof("start ws")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logrus.Errorf("ws handshake error")
		}
		return err
	}

	//get activityId,stageOrdinal,stepOrdinal from request
	v := r.URL.Query()
	activityId := v.Get("activityId")
	stageOrdinal, err := strconv.Atoi(v.Get("stageOrdinal"))
	if err != nil {
		return err
	}
	stepOrdinal, err := strconv.Atoi(v.Get("stepOrdinal"))
	if err != nil {
		return err
	}
	go s.stepLogWriter(ws, activityId, stageOrdinal, stepOrdinal)
	stepLogReader(ws)
	return nil
}
