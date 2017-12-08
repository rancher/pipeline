package server

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/pipeline/git"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/scheduler"
	"github.com/rancher/pipeline/server/service"
	"github.com/sluu99/uuid"
	"golang.org/x/sync/syncmap"
)

//Component to hold schedulers and connholders
type Agent struct {
	Server *Server

	connHolders map[*ConnHolder]bool
	// Register requests from the connholder.
	register chan *ConnHolder
	// Unregister requests from connholder.
	unregister chan *ConnHolder

	broadcast chan WSMsg

	//scheduler
	cronRunners           map[string]*scheduler.CronRunner
	registerCronRunnerC   chan *scheduler.CronRunner
	unregisterCronRunnerC chan string

	activityLocks syncmap.Map
}

var GlobalAgent *Agent

func broadcastResourceChange(obj interface{}) {
	resourceType := ""
	switch obj.(type) {
	case model.Activity:
		resourceType = "activity"
	case model.Pipeline:
		resourceType = "pipeline"
	case model.GitAccount:
		resourceType = "gitaccount"
	case model.PipelineSetting:
		resourceType = "setting"
	case model.SCMSetting:
		resourceType = "scmSetting"
	default:
		logrus.Warningf("unsupported resource type to broadcast")
		return

	}
	GlobalAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: resourceType,
		Time:         time.Now(),
		Data:         obj,
	}
}

func InitAgent(s *Server) {
	GlobalAgent = &Agent{
		Server:                s,
		connHolders:           make(map[*ConnHolder]bool),
		register:              make(chan *ConnHolder),
		unregister:            make(chan *ConnHolder),
		broadcast:             make(chan WSMsg),
		cronRunners:           make(map[string]*scheduler.CronRunner),
		registerCronRunnerC:   make(chan *scheduler.CronRunner),
		unregisterCronRunnerC: make(chan string),
		activityLocks:         syncmap.Map{},
	}
	logrus.Debugf("inited GlobalAgent:%v", GlobalAgent)
	go GlobalAgent.handleWS()
	go GlobalAgent.RunScheduler()

}

func (a *Agent) handleWS() {
	for {
		select {
		case h := <-a.register:
			a.connHolders[h] = true
		case h := <-a.unregister:
			if _, ok := a.connHolders[h]; ok {
				delete(a.connHolders, h)
				close(h.send)
			}

		case message := <-a.broadcast:
			//tell all the web socket connholder in this case
			logrus.Debugf("broadcast %v holders!", len(a.connHolders))
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

func (a *Agent) RunScheduler() {

	pipelines := service.ListPipelines()
	for _, pipeline := range pipelines {
		if pipeline.IsActivate && pipeline.CronTrigger.Spec != "" {
			cr := scheduler.NewCronRunner(pipeline.Id, pipeline.CronTrigger.Spec, pipeline.CronTrigger.Timezone)
			a.registerCronRunner(cr)
		}
	}
	logrus.Debugf("run scheduler,init size:%v", len(a.cronRunners))
	for {
		select {
		case cr := <-a.registerCronRunnerC:
			a.registerCronRunner(cr)
		case pId := <-a.unregisterCronRunnerC:
			a.unregisterCronRunner(pId)
		}
	}
}

func (a *Agent) onPipelineChange(p *model.Pipeline) {
	logrus.Debugf("on pipeline change")
	pId := p.Id
	spec := ""
	timezone := ""
	if (!p.IsActivate) || (p.CronTrigger.Spec == "") {
		//deactivate,remove the cron
		a.unregisterCronRunnerC <- pId
	}

	if p.IsActivate && p.CronTrigger.Spec != "" {
		spec = p.CronTrigger.Spec
		timezone = p.CronTrigger.Timezone
		cr := scheduler.NewCronRunner(pId, spec, timezone)
		a.registerCronRunnerC <- cr
	}
	p.NextRunTime = service.GetNextRunTime(p)
	service.UpdatePipeline(p)
	a.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "pipeline",
		Time:         time.Now(),
		Data:         p,
	}

}

func (a *Agent) onPipelineDelete(p *model.Pipeline) {
	pId := p.Id
	if p.IsActivate {
		a.unregisterCronRunnerC <- pId
	}
	p.Status = "removed"
	a.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "pipeline",
		Time:         time.Now(),
		Data:         p,
	}
}
func (a *Agent) onPipelineActivate(p *model.Pipeline) {
	if p.CronTrigger.Spec != "" {
		pId := p.Id
		spec := p.CronTrigger.Spec
		timezone := p.CronTrigger.Timezone
		cr := scheduler.NewCronRunner(pId, spec, timezone)
		a.registerCronRunnerC <- cr
	}
	a.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "pipeline",
		Time:         time.Now(),
		Data:         p,
	}
}

func (a *Agent) onPipelineDeActivate(p *model.Pipeline) {
	a.unregisterCronRunnerC <- p.Id
	a.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "pipeline",
		Time:         time.Now(),
		Data:         p,
	}
}

//registerCronRunner add or update a cronRunner
func (a *Agent) registerCronRunner(cr *scheduler.CronRunner) {
	pId := cr.PipelineId
	existing := a.cronRunners[pId]
	logrus.Debugf("registering conrunner,pid:%v,spec:%v", pId, cr.Spec)
	if existing != nil {
		if existing.Spec == cr.Spec {
			return
		}
		existing.Stop()
		delete(a.cronRunners, pId)
	}
	if cr.Spec != "" {
		err := cr.AddFunc(cr.Spec, func() {
			logrus.Debugf("invoke pipeline %v cron job", cr.PipelineId)
			ppl, err := service.GetPipelineById(pId)
			if err != nil {
				logrus.Errorf("fail to get pipeline:%v", err)
				return
			}

			if ppl.CronTrigger.TriggerOnUpdate {
				//run only when new changes exist

				gitUser := ppl.Stages[0].Steps[0].GitUser
				token, err := service.GetUserToken(gitUser)
				if err != nil {
					logrus.Errorf("fail to get user credential for %s: %v", gitUser, err)
					return
				}
				repoUrl, err := git.GetAuthRepoUrl(ppl.Stages[0].Steps[0].Repository, gitUser, token)
				if err != nil {
					logrus.Errorf("get repo credential got error: %v", err)
					return
				}
				latestCommit, err := git.BranchHeadCommit(repoUrl, ppl.Stages[0].Steps[0].Branch)
				if err != nil {
					logrus.Errorf("cron job fail,Error:%v", err)
					return
				}
				if latestCommit == ppl.CommitInfo {
					//update nextruntime and return
					ppl.NextRunTime = service.GetNextRunTime(ppl)

					if err := service.UpdatePipeline(ppl); err != nil {
						logrus.Errorf("update pipeline error,%v", err)
					}
					a.broadcast <- WSMsg{
						Id:           uuid.Rand().Hex(),
						Name:         "resource.change",
						ResourceType: "pipeline",
						Time:         time.Now(),
						Data:         ppl,
					}
					return
				}
			}
			_, err = service.RunPipeline(a.Server.Provider, pId, model.TriggerTypeCron)
			if err != nil {
				logrus.Errorf("cron job fail,pid:%v", pId)
				return
			}
		})
		if err != nil {
			logrus.Error("cron addfunc error for pipeline %v:%v", pId, err)
			return
		}
		cr.Start()
		a.cronRunners[pId] = cr
	}

}

//unregisterCronRunner remove cronrunner for pipeline
func (a *Agent) unregisterCronRunner(pipelineId string) {
	logrus.Debugf("unregistering conrunner,pid:%v", pipelineId)
	existing := a.cronRunners[pipelineId]
	if existing != nil {
		existing.Stop()
	}
	delete(a.cronRunners, pipelineId)
}

func (a *Agent) getActivityLock(activityId string) *sync.Mutex {
	lock, _ := a.activityLocks.Load(activityId)
	if lock == nil {
		mutex := &sync.Mutex{}
		a.activityLocks.Store(activityId, mutex)
		return mutex
	}
	return lock.(*sync.Mutex)
}
