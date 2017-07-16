package restfulserver

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/pipeline/pipeline"
)

//component to comunicate between server and ci provider

type Agent struct {
	Server      *Server
	connHolders map[*ConnHolder]bool
	// Register requests from the connholder.
	register chan *ConnHolder

	// Unregister requests from connholder.
	unregister chan *ConnHolder

	broadcast chan []byte

	ReWatch chan bool
}

var MyAgent *Agent

func InitAgent(s *Server) {
	logrus.Infof("init agent")
	MyAgent = &Agent{
		Server:      s,
		connHolders: make(map[*ConnHolder]bool),
		register:    make(chan *ConnHolder),
		unregister:  make(chan *ConnHolder),
		broadcast:   make(chan []byte),
		ReWatch:     make(chan bool),
	}
	logrus.Infof("inited myagent:%v", MyAgent)
	go MyAgent.handleWS()
	go MyAgent.SyncWatchList()

}

func (a *Agent) handleWS() {
	for {
		select {
		case h := <-a.register:
			logrus.Infof("register a holder!")
			a.connHolders[h] = true
		case h := <-a.unregister:
			logrus.Infof("unregister a holder!")
			if _, ok := a.connHolders[h]; ok {
				delete(a.connHolders, h)
				close(h.send)
			}

		case message := <-a.broadcast:
			//tell all the web socket connholder in this case
			logrus.Infof("broadcast %v holders!", len(a.connHolders))
			for holder := range a.connHolders {
				select {
				case holder.send <- message:
				default:
					close(holder.send)
					delete(a.connHolders, holder)
				}
			}
		}
	}
}
func (a *Agent) SyncWatchList() {

	logrus.Infof("start sync")
	var watchlist []*pipeline.Activity
	var err error
	ticker := time.NewTicker(syncPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		watchlist, err = a.getWatchList()
		if err != nil {
			logrus.Errorf("error get watchlist,%v", err)
		}
		for {
			select {
			case <-ticker.C:
				for _, activity := range watchlist {
					if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail {
						continue
					}
					updated, _ := a.Server.PipelineContext.Provider.SyncActivity(activity)
					//logrus.Infof("sync activity:%v,updated:%v", activity.Id, updated)
					/*
						if activity.Id == "1def6e31-345d-48ee-b443-6f633f35a636" {
							updated = true
						}
					*/
					if updated {
						//status changed,then update in rancher server
						err = UpdateActivity(*activity)
						if err != nil {
							logrus.Errorf("fail update activity,%v", err)
						}

						logrus.Infof("telling all holder to send messages!")
						a.broadcast <- []byte(activity.Id)
					}
				}
			case <-a.ReWatch:
				//reget the watchlist
				break
			}
		}
	}
}

func (a *Agent) getWatchList() ([]*pipeline.Activity, error) {
	logrus.Infof("getting watchlist")
	activities, err := ListActivities(a.Server.PipelineContext)
	if err != nil {
		return nil, err
	}

	var watchlist []*pipeline.Activity
	for _, activity := range activities {
		if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail {
			continue
		} else {
			logrus.Infof("add %v to watchlist", activity.Id)
			watchlist = append(watchlist, activity)
		}
	}
	logrus.Infof("got watchlist,size:%v", len(watchlist))
	return watchlist, nil
}

func (a *Agent) GetStepLog(activityId string, stageOrdinal int, stepOrdinal int) (string, error) {

	return "", nil
}
