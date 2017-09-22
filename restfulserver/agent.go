package restfulserver

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/pipeline/git"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/scheduler"
)

//Component to hold schedulers and connholders
type Agent struct {
	Server *Server

	connHolders map[*ConnHolder]bool
	// Register requests from the connholder.
	register chan *ConnHolder
	// Unregister requests from connholder.
	unregister chan *ConnHolder

	broadcast chan []byte

	//scheduler
	cronRunners           map[string]*scheduler.CronRunner
	registerCronRunnerC   chan *scheduler.CronRunner
	unregisterCronRunnerC chan string
}

var MyAgent *Agent

func InitAgent(s *Server) {
	MyAgent = &Agent{
		Server:                s,
		connHolders:           make(map[*ConnHolder]bool),
		register:              make(chan *ConnHolder),
		unregister:            make(chan *ConnHolder),
		broadcast:             make(chan []byte),
		cronRunners:           make(map[string]*scheduler.CronRunner),
		registerCronRunnerC:   make(chan *scheduler.CronRunner),
		unregisterCronRunnerC: make(chan string),
	}
	logrus.Debugf("inited myagent:%v", MyAgent)
	go MyAgent.handleWS()
	go MyAgent.RunScheduler()

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

	pipelines := a.Server.PipelineContext.ListPipelines()
	for _, pipeline := range pipelines {
		if pipeline.IsActivate && pipeline.TriggerSpec != "" {
			cr := scheduler.NewCronRunner(pipeline.Id, pipeline.TriggerSpec, pipeline.TriggerTimezone)
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

func (a *Agent) onPipelineChange(p *pipeline.Pipeline) {
	logrus.Debugf("on pipeline change")
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
	logrus.Debugf("registering conrunner,pid:%v,spec:%v", pId, cr.Spec)
	if existing == nil {
		err := cr.AddFunc(cr.Spec, func() {
			logrus.Debugf("invoke pipeline %v cron job", cr.PipelineId)
			ppl := a.Server.PipelineContext.GetPipelineById(pId)
			latestCommit, err := git.BranchHeadCommit(ppl.Stages[0].Steps[0].Repository, ppl.Stages[0].Steps[0].Branch)
			if err != nil {
				logrus.Errorf("cron job fail,Error:%v", err)
				return
			}
			if ppl.TriggerOnUpdate && latestCommit == ppl.CommitInfo {
				//run only when new changes exist
				return
			}
			_, err = a.Server.PipelineContext.RunPipeline(pId)
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
	logrus.Debugf("unregistering conrunner,pid:%v", pipelineId)
	existing := a.cronRunners[pipelineId]
	if existing != nil {
		existing.Stop()
	}
	delete(a.cronRunners, pipelineId)
}
