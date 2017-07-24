package restfulserver

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/scheduler"
	"github.com/rancher/pipeline/util"
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
	watchingLabel:
		for {
			select {
			case <-ticker.C:
				for _, activity := range watchlist {
					if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail {
						continue
					}
					updated, _ := a.Server.PipelineContext.Provider.SyncActivity(activity)
					logrus.Infof("sync activity:%v,updated:%v", activity.Id, updated)
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
				logrus.Infof("rewatch signal")
				//reget the watchlist
				break watchingLabel
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
		if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail {
			continue
		} else {
			//logrus.Infof("add %v to watchlist", activity.Id)
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
				if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail {
					continue
				}
				updated, _ := a.Server.PipelineContext.Provider.SyncActivity(activity)
				logrus.Infof("sync activity:%v,updated:%v", activity.Id, updated)
				if updated {
					//status changed,then update in rancher server

					err = UpdateActivity(*activity)
					if err != nil {
						logrus.Errorf("fail update activity,%v", err)
					}
					logrus.Infof("trying to update lastactivity:%v", activity)
					a.Server.UpdateLastActivity(activity.Pipeline.Id)

					if activity.Status == pipeline.ActivitySuccess || activity.Status == pipeline.ActivityFail {
						//done,remove from watchlist
						delete(a.activityWatchlist, activity.Id)
					}
					logrus.Infof("telling all holder to send messages!")
					a.broadcast <- []byte(activity.Id)
				}
			}
		case acti := <-a.watchActivityC:
			a.activityWatchlist[acti.Id] = acti

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

func (a *Agent) onPipelineChange(p *pipeline.Pipeline, req *http.Request) {
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
	//update webhook
	logrus.Infof("pipelinechange,webhook:%v,%v", p.Stages[0].Steps[0].Webhook, p.WebHookId)
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.Stages[0].Steps[0].Webhook {
			if p.WebHookId <= 0 {
				webhookUrl := getWebhookUrl(req, p.Id)
				logrus.Infof("get webhookUrl:%v", webhookUrl)
				createWebhook(p, webhookUrl)
			}
		} else {
			if p.WebHookId > 0 {
				deleteWebhook(p)
			}
		}

	}

	p.NextRunTime = pipeline.GetNextRunTime(p)
	a.Server.PipelineContext.UpdatePipeline(p)

}

func (a *Agent) onPipelineDelete(p *pipeline.Pipeline) {
	pId := p.Id
	if p.IsActivate {
		a.unregisterCronRunnerC <- pId
	}
	deleteWebhook(p)
}

func deleteWebhook(p *pipeline.Pipeline) error {
	logrus.Infof("deletewebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to delete webhook")
	}

	//delete webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.WebHookId > 0 {
			//TODO
			repoUrl := p.Stages[0].Steps[0].Repository
			token := p.Stages[0].Steps[0].Token
			reg := regexp.MustCompile(".*?github.com/(.*?)/(.*?).git")
			match := reg.FindStringSubmatch(repoUrl)
			if len(match) < 3 {
				logrus.Errorf("error getting user/repo from gitrepoUrl:%v", repoUrl)
				return errors.New(fmt.Sprintf("error getting user/repo from gitrepoUrl:%v", repoUrl))
			}
			user := match[1]
			repo := match[2]
			err := util.DeleteWebhook(user, repo, token, p.WebHookId)
			if err != nil {
				logrus.Errorf("error delete webhook,%v", err)
				return err
			}
			p.WebHookId = 0
		}
	}
	return nil
}

func getWebhookUrl(req *http.Request, pipelineId string) string {
	proto := "http://"
	if req.TLS != nil {
		proto = "https://"
	}
	url := proto + req.Host + req.URL.Path
	reg := regexp.MustCompile("(.*?v1)/pipeline.*?")
	match := reg.FindStringSubmatch(url)
	var r string = "fail to get webhookurl"
	if len(match) > 1 {
		r = match[1] + "/webhook/" + pipelineId
	}
	return r

}

func createWebhook(p *pipeline.Pipeline, webhookUrl string) error {
	logrus.Infof("createwebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to create webhook")
	}

	//create webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.Stages[0].Steps[0].Webhook {
			//TODO
			repoUrl := p.Stages[0].Steps[0].Repository
			token := p.Stages[0].Steps[0].Token
			reg := regexp.MustCompile(".*?github.com/(.*?)/(.*?).git")
			match := reg.FindStringSubmatch(repoUrl)
			if len(match) < 3 {
				logrus.Errorf("error getting user/repo from gitrepoUrl:%v", repoUrl)
				return errors.New(fmt.Sprintf("error getting user/repo from gitrepoUrl:%v", repoUrl))
			}
			user := match[1]
			repo := match[2]
			secret := p.WebHookToken
			id, err := util.CreateWebhook(user, repo, token, webhookUrl, secret)
			logrus.Infof("get:%v,%v,%v,%v,%v,%v", user, repo, token, webhookUrl, secret, id)
			if err != nil {
				logrus.Errorf("error delete webhook,%v", err)
				return err
			}
			p.WebHookId = id
		}
	}
	return nil
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
		err := cr.AddFunc("0 "+cr.Spec, func() { logrus.Infof("cron job run,pid:%v", pId); a.Server.PipelineContext.RunPipeline(pId) })
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
				err := cr.AddFunc("0 "+cr.Spec, func() { a.Server.PipelineContext.RunPipeline(pId) })
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
