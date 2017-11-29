package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/server/service"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

func (s *Server) Webhook(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debugf("get header:%v", req.Header)
	logrus.Debugf("get url:%v", req.RequestURI)

	var manager model.SCManager
	var err error
	var eventType string
	if eventType = req.Header.Get("X-GitHub-Event"); len(eventType) != 0 {
		if eventType == "ping" {
			return nil
		}
		logrus.Debug("receive webhook from github")
		manager, err = service.GetSCManager("github")
		if err != nil {
			return err
		}
	} else if eventType = req.Header.Get("X-Gitlab-Event"); len(eventType) != 0 {
		logrus.Debug("receive webhook from gitlab")
		manager, err = service.GetSCManager("gitlab")
		if err != nil {
			return err
		}
	} else {
		//TODO generic webhook
		return errors.New("Unknown webhook source")
	}

	id := req.FormValue("pipelineId")
	pipeline, err := service.GetPipelineById(id)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	if !pipeline.IsActivate {
		return errors.New("pipeline is not activated")
	}
	if !manager.VerifyWebhookPayload(pipeline, req) {
		return errors.New("verify webhook fail")
	}

	logrus.Debugf("token validate pass")

	if _, err = service.RunPipeline(s.Provider, id, model.TriggerTypeWebhook); err != nil {
		rw.Write([]byte("run pipeline error!"))
		return err
	}
	rw.Write([]byte("run pipeline success!"))
	logrus.Infof("webhook trigger run for '%s' success", pipeline.Name)
	return nil
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
	uid, err := util.GetCurrentUser(r.Cookies())
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

//list available env vars
func (s *Server) ListEnvVars(rw http.ResponseWriter, req *http.Request) error {
	b, err := json.Marshal(model.PreservedEnvs)
	_, err = rw.Write(b)
	return err
}

func (s *Server) StepStart(rw http.ResponseWriter, req *http.Request) error {
	v := req.URL.Query()
	activityId := v.Get("id")
	stageOrdinal, err := strconv.Atoi(v.Get("stageOrdinal"))
	if err != nil {
		return err
	}
	stepOrdinal, err := strconv.Atoi(v.Get("stepOrdinal"))
	if err != nil {
		return err
	}

	mutex := MyAgent.getActivityLock(activityId)
	mutex.Lock()
	defer mutex.Unlock()

	logrus.Debugf("get stepstart event,paras:%v,%v,%v", activityId, stageOrdinal, stepOrdinal)
	activity, err := service.GetActivity(activityId)
	if err != nil {
		return err
	}
	if stageOrdinal < 0 || stepOrdinal < 0 || stageOrdinal >= len(activity.ActivityStages) || stepOrdinal >= len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return errors.New("step index invalid")
	}
	service.StartStep(activity, stageOrdinal, stepOrdinal)
	if err = service.UpdateActivity(activity); err != nil {
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         *activity,
	}
	return nil
}

func (s *Server) StepFinish(rw http.ResponseWriter, req *http.Request) error {
	//get activityId,stageOrdinal,stepOrdinal from request
	v := req.URL.Query()
	activityId := v.Get("id")
	status := v.Get("status")
	stageOrdinal, err := strconv.Atoi(v.Get("stageOrdinal"))
	if err != nil {
		return err
	}
	stepOrdinal, err := strconv.Atoi(v.Get("stepOrdinal"))
	if err != nil {
		return err
	}
	mutex := MyAgent.getActivityLock(activityId)
	mutex.Lock()
	defer mutex.Unlock()

	logrus.Debugf("get stepfinish event,paras:%v,%v,%v", activityId, stageOrdinal, stepOrdinal)
	activity, err := service.GetActivity(activityId)
	if err != nil {
		return err
	}
	if stageOrdinal < 0 || stepOrdinal < 0 || stageOrdinal >= len(activity.ActivityStages) || stepOrdinal >= len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return errors.New("step index invalid")
	}
	if status == "SUCCESS" {
		service.SuccessStep(activity, stageOrdinal, stepOrdinal)
		service.Triggernext(activity, stageOrdinal, stepOrdinal, s.Provider)
	} else if status == "FAILURE" {
		service.FailStep(activity, stageOrdinal, stepOrdinal)
	}

	//update commitinfo for SCM step
	if stageOrdinal == 0 && stepOrdinal == 0 {
		activity.CommitInfo = req.FormValue("GIT_COMMIT")
		activity.EnvVars["CICD_GIT_COMMIT"] = activity.CommitInfo
	}

	if err = service.UpdateActivity(activity); err != nil {
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         *activity,
	}
	s.UpdateLastActivity(activity)

	if activity.Status == model.ActivityFail || activity.Status == model.ActivitySuccess {
		s.Provider.OnActivityCompelte(activity)
	}

	return nil
}

func (s *Server) Reset(rw http.ResponseWriter, req *http.Request) error {
	return service.Reset()
}
