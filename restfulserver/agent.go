package restfulserver

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/scheduler"
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

	activityWatchlist map[string]*pipeline.Activity

	watchActivityC chan *pipeline.Activity
	ReWatch        chan bool

	//scheduler

	cronRunners           map[string]*scheduler.CronRunner
	registerCronRunnerC   chan *scheduler.CronRunner
	unregisterCronRunnerC chan string
}

var MyAgent *Agent

func InitAgent(s *Server) {
	logrus.Infof("init agent")
	MyAgent = &Agent{
		Server:                s,
		connHolders:           make(map[*ConnHolder]bool),
		register:              make(chan *ConnHolder),
		unregister:            make(chan *ConnHolder),
		broadcast:             make(chan []byte),
		activityWatchlist:     make(map[string]*pipeline.Activity),
		watchActivityC:        make(chan *pipeline.Activity),
		ReWatch:               make(chan bool),
		cronRunners:           make(map[string]*scheduler.CronRunner),
		registerCronRunnerC:   make(chan *scheduler.CronRunner),
		unregisterCronRunnerC: make(chan string),
	}
	logrus.Infof("inited myagent:%v", MyAgent)
	go MyAgent.handleWS()
	go MyAgent.SyncActivityWatchList()
	go MyAgent.RunScheduler()

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

func (a *Agent) SyncActivityWatchList() {
	activities, err := ListActivities(a.Server.PipelineContext)
	logrus.Infof("get total activities:%v", len(activities))
	if err != nil {
		logrus.Errorf("fail to get activities")
	}
	for _, activity := range activities {
		if activity.Status == pipeline.ActivityWaiting || activity.Status == pipeline.ActivityBuilding {
			a.activityWatchlist[activity.Id] = activity
		}
	}
	logrus.Infof("got watchlist,size:%v", len(a.activityWatchlist))
	ticker := time.NewTicker(syncPeriod)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			for _, activity := range a.activityWatchlist {
				if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail || activity.Status == pipeline.ActivityDenied {
					delete(a.activityWatchlist, activity.Id)
					continue
				}
				updated, _ := a.Server.PipelineContext.Provider.SyncActivity(activity)
				logrus.Infof("sync activity:%v,updated:%v", activity.Id, updated)
				if updated {
					//status changed,then update in rancher server

					err = UpdateActivity(*activity)
					if err != nil {
						logrus.Errorf("fail update activity,%v", err)
						continue
					}
					a.Server.UpdateLastActivity(activity.Pipeline.Id)

					if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail {
						//done,remove from watchlist
						delete(a.activityWatchlist, activity.Id)
					}
					if activity.Status == pipeline.ActivityPending || activity.Status == pipeline.ActivityDenied {
						//pending,remove from watchlist. add agin when approve
						delete(a.activityWatchlist, activity.Id)
					}
					//when activity done,invoke providor.onActivityComplete
					if activity.Status == pipeline.ActivityFail || activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityDenied {
						a.Server.PipelineContext.Provider.OnActivityCompelte(activity)

					}
					logrus.Infof("telling all holder to send messages!")
					a.broadcast <- []byte(activity.Id)
				}
			}
		case acti := <-a.watchActivityC:
			a.activityWatchlist[acti.Id] = acti
			a.broadcast <- []byte(acti.Id)

		}
	}

}

func (a *Agent) getWatchList() ([]*pipeline.Activity, error) {
	logrus.Infof("getting watchlist")
	activities, err := ListActivities(a.Server.PipelineContext)
	logrus.Infof("get total activities:%v", len(activities))
	if err != nil {
		return nil, err
	}

	var watchlist []*pipeline.Activity
	for _, activity := range activities {
		if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail {
			continue
		} else {
			//logrus.Infof("add %v to watchlist", activity.Id)
			watchlist = append(watchlist, activity)
		}
	}
	logrus.Infof("got watchlist,size:%v", len(watchlist))
	return watchlist, nil
}

func (a *Agent) RunScheduler() {

	pipelines := a.Server.PipelineContext.ListPipelines()
	for _, pipeline := range pipelines {
		if pipeline.IsActivate && pipeline.TriggerSpec != "" {
			cr := scheduler.NewCronRunner(pipeline.Id, pipeline.TriggerSpec, pipeline.TriggerTimezone)
			a.registerCronRunner(cr)
		}
	}
	logrus.Infof("run scheduler,init size:%v", len(a.cronRunners))
	for {
		select {
		case cr := <-a.registerCronRunnerC:
			a.registerCronRunner(cr)
		case pId := <-a.unregisterCronRunnerC:
			logrus.Infof("")
			a.unregisterCronRunner(pId)
		}
	}
}

func (a *Agent) onPipelineChange(p *pipeline.Pipeline) {
	logrus.Infof("on pipeline change")
	pId := p.Id
	spec := ""
	timezone := ""
	if !p.IsActivate {
		//deactivate,remove the cron
		a.unregisterCronRunnerC <- pId
	}

	if p.IsActivate {
		spec = p.TriggerSpec
		timezone = p.TriggerTimezone
		cr := scheduler.NewCronRunner(pId, spec, timezone)
		a.registerCronRunnerC <- cr
	}
	p.NextRunTime = pipeline.GetNextRunTime(p)
	a.Server.PipelineContext.UpdatePipeline(p)

}

func (a *Agent) onPipelineDelete(p *pipeline.Pipeline) {
	pId := p.Id
	if p.IsActivate {
		a.unregisterCronRunnerC <- pId
	}
}
func (a *Agent) onPipelineActivate(p *pipeline.Pipeline) {
	pId := p.Id
	spec := p.TriggerSpec
	timezone := p.TriggerTimezone
	cr := scheduler.NewCronRunner(pId, spec, timezone)
	a.registerCronRunnerC <- cr
}

func (a *Agent) onPipelineDeActivate(p *pipeline.Pipeline) {
	a.unregisterCronRunnerC <- p.Id
}

//registerCronRunner add or update a cronRunner
func (a *Agent) registerCronRunner(cr *scheduler.CronRunner) {
	pId := cr.PipelineId
	existing := a.cronRunners[pId]
	logrus.Infof("registering conrunner,pid:%v,spec:%v", pId, cr.Spec)
	if existing == nil {
		err := cr.AddFunc(cr.Spec, func() {
			acti, err := a.Server.PipelineContext.RunPipeline(pId)
			if err != nil {
				logrus.Errorf("cron job fail,pid:%v", pId)
				return
			}
			a.watchActivityC <- acti

		})
		if err != nil {
			logrus.Error("cron addfunc error for pipeline %v:%v", pId, err)
			return
		}
		cr.Start()
		a.cronRunners[pId] = cr
	} else {
		if existing.Spec == cr.Spec {
			return
		} else {
			//update cron spec
			existing.Stop()
			delete(a.cronRunners, pId)
			if cr.Spec != "" {
				err := cr.AddFunc(cr.Spec, func() { a.Server.PipelineContext.RunPipeline(pId) })
				if err != nil {
					logrus.Error("cron addfunc error for pipeline %v:%v", pId, err)
					return
				}
				cr.Start()
				a.cronRunners[pId] = cr
			}
		}

	}

}

//unregisterCronRunner remove cronrunner for pipeline
func (a *Agent) unregisterCronRunner(pipelineId string) {
	logrus.Infof("unregistering conrunner,pid:%v", pipelineId)
	existing := a.cronRunners[pipelineId]
	if existing != nil {
		existing.Stop()
	}
	delete(a.cronRunners, pipelineId)
}

func (a *Agent) onActivityComplete(activity *pipeline.Activity) {
	//clean service

}
