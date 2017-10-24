// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package restfulserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
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

	syncPeriod = 1 * time.Second
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

			paras := map[string]interface{}{}
			paras["prevLog"] = &prevLog
			stepLog, err := s.PipelineContext.Provider.GetStepLog(&activity, stageOrdinal, stepOrdinal, paras)
			if err != nil {
				logrus.Errorf("error get steplog,%v", err)
				return
			}
			if stepLog != "" {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				logData, _ := computeLogTimestamp(activity.StartTS, stepLog)
				response := WSMsg{
					Id:           uuid.Rand().Hex(),
					Name:         "resource.change",
					ResourceType: "log",
					Time:         time.Now(),
					Data:         logData,
				}
				b, err = json.Marshal(response)
				if err := ws.WriteMessage(websocket.TextMessage, b); err != nil {
					return
				}
				if strings.HasSuffix(stepLog, "\n  Finished: SUCCESS\n") ||
					strings.HasSuffix(stepLog, "\n  Finished: FAILURE\n") ||
					strings.HasSuffix(stepLog, "\n  Finished: ABORTED\n") {
					//finish
					return
				}
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

func computeLogTimestamp(startTS int64, stepLog string) (string, error) {
	lines := strings.Split(stepLog, "\n")
	b := bytes.NewBufferString("")
	timestr := ""
	for _, line := range lines {
		if line == "" {
			continue
		}
		spans := strings.SplitN(line, "  ", 2)
		// to handle misformat log from jenkins timestamper
		if spans[0] != "" {
			timestr = spans[0]
		} else {
			spans[0] = timestr
		}
		duration, err := time.ParseDuration(spans[0])
		if err != nil {
			logrus.Errorf("parse duration error!%v", err)
			return stepLog, errors.New("parse duration error!")
		}
		lineTime := startTS + (duration.Nanoseconds() / int64(time.Millisecond))
		b.WriteString(strconv.FormatInt(lineTime, 10))
		b.WriteString("  ")
		b.WriteString(spans[1])
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (s *Server) ServeStepLog(w http.ResponseWriter, r *http.Request) error {
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
