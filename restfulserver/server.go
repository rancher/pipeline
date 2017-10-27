package restfulserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	v2client "github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/config"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/restfulserver/webhook"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

//Server rest api server
type Server struct {
	PipelineContext *pipeline.PipelineContext
}

func Preset(pipelineContext *pipeline.PipelineContext) {
	if err := checkCIEndpoint(); err != nil {
		logrus.Errorf("Check CI Endpoint Error:%v", err)
	}
	activities, err := ListActivities(pipelineContext)
	if err != nil {
		logrus.Errorf("List activities Error:%v", err)
	}
	logrus.Debugf("get activities size:%v", len(activities))
	//Sync status of running activities
	for _, a := range activities {
		//TODO !a.IsRunning
		if a.Status == pipeline.ActivityFail ||
			a.Status == pipeline.ActivitySuccess ||
			a.Status == pipeline.ActivityDenied ||
			a.Status == pipeline.ActivityPending ||
			a.Status == pipeline.ActivityAbort {
			continue
		}
		if err := pipelineContext.Provider.SyncActivity(a); err != nil {
			logrus.Errorf("Sync activity Error:%v", err)
			continue
		}
		if err := UpdateActivity(*a); err != nil {
			logrus.Errorf("Update activity Error:%v", err)
		}
	}
}

func checkCIEndpoint() error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "webhookReceiver"
	opt := &v2client.ListOpts{
		Filters: filters,
	}
	gCollection, err := apiClient.GenericObject.List(opt)
	if err != nil {
		return err
	}
	var ciWebhook *webhook.WebhookGenericObject
	for _, wh := range gCollection.Data {
		whObject, err := webhook.ConvertToWebhookGenericObject(wh)
		if err != nil {
			logrus.Errorf("Preset get webhook endpoint error:%v", err)
			return err
		}
		if whObject.Driver == webhook.CIWEBHOOKTYPE && whObject.Name == webhook.CIWEBHOOKNAME {
			ciWebhook = &whObject
			//get CIWebhookEndpoint
			webhook.CIWebhookEndpoint = ciWebhook.URL
			logrus.Infof("Using webhook '%s' as CI Endpoint.", ciWebhook.URL)
		}
	}
	if ciWebhook == nil {
		//Create a webhook for github webhook payload url
		if err := webhook.CreateCIEndpointWebhook(); err != nil {
			logrus.Errorf("CreateCIEndpointWebhook Error:%v", err)
			return err
		}

	}
	return nil
}

//ListPipelines query List of pipelines
func (s *Server) ListPipelines(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	uid, err := GetCurrentUser(req.Cookies())
	if err != nil || uid == "" {
		logrus.Debugf("getAccessibleAccounts unrecognized user")
	}
	accessibleAccounts := getAccessibleAccounts(uid)
	var pipelines []*pipeline.Pipeline
	allpipelines := s.PipelineContext.ListPipelines()
	//filter by git account access
	for _, pipeline := range allpipelines {
		if accessibleAccounts[pipeline.Stages[0].Steps[0].GitUser] {
			pipelines = append(pipelines, pipeline)
		}
	}
	apiContext.Write(&client.GenericCollection{
		Data: toPipelineCollections(apiContext, pipelines),
	})
	return nil
}

func (s *Server) Webhook(rw http.ResponseWriter, req *http.Request) error {
	var signature string
	var event_type string
	logrus.Debugln("get webhook request")
	logrus.Debugf("get header:%v", req.Header)
	logrus.Debugf("get url:%v", req.RequestURI)

	if signature = req.Header.Get("X-Hub-Signature"); len(signature) == 0 {
		return errors.New("No signature!")
	}
	if event_type = req.Header.Get("X-GitHub-Event"); len(event_type) == 0 {
		return errors.New("No event!")
	}

	if event_type == "ping" {
		rw.Write([]byte("pong"))
		return nil
	}
	if event_type != "push" {
		logrus.Errorf("not push event")
		return errors.New("not push event")
	}

	id := req.FormValue("pipelineId")
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	r := s.PipelineContext.GetPipelineById(id)
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", id)
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("pipeline not found!"))
		return err
	}
	logrus.Debugf("webhook trigger,id:%v,event:%v,signature:%v,body:\n%v\n%v", id, event_type, signature, body, string(body))
	if !util.VerifyWebhookSignature([]byte(r.WebHookToken), signature, body) {
		return errors.New("Invalid signature")
	}
	logrus.Infof("token validate pass")

	//check branch
	payload := &github.WebHookPayload{}
	if err := json.Unmarshal(body, payload); err != nil {
		return err
	}
	if *payload.Ref != "refs/heads/"+r.Stages[0].Steps[0].Branch {
		logrus.Warningf("branch not match:%v,%v", *payload.Ref, r.Stages[0].Steps[0].Branch)
		return nil
	}

	if !r.IsActivate {
		logrus.Errorf("pipeline is not activated!")
		return errors.New("pipeline is not activated!")
	}
	_, err = s.PipelineContext.RunPipeline(id, pipeline.TriggerTypeWebhook)
	if err != nil {
		rw.Write([]byte("run pipeline error!"))
		return err
	}
	//MyAgent.watchActivityC <- activity
	rw.Write([]byte("run pipeline success!"))
	logrus.Infof("webhook run success")
	return nil
}

func (s *Server) ListPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r := s.PipelineContext.GetPipelineById(id)
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", id)
		rw.WriteHeader(http.StatusNotFound)
		apiContext.Write(&Error{
			Resource: client.Resource{
				Id:      uuid.Rand().Hex(),
				Type:    "error",
				Links:   map[string]string{},
				Actions: map[string]string{},
			},
			Status: http.StatusNotFound,
			Msg:    err.Error(),
			Code:   err.Error(),
		})
		return err
	}
	apiContext.Write(toPipelineResource(apiContext, r))
	return nil
}

//update last activity info in the pipeline on activity changes
func (s *Server) UpdateLastActivity(activity pipeline.Activity) {
	logrus.Debugf("begin UpdateLastActivity")
	pId := activity.Pipeline.Id
	p := s.PipelineContext.GetPipelineById(pId)
	if p == nil || p.LastRunId == "" {
		return
	}
	if activity.Id != p.LastRunId {
		return
	}
	p.LastRunStatus = activity.Status
	p.CommitInfo = activity.CommitInfo
	p.NextRunTime = pipeline.GetNextRunTime(p)

	if err := s.PipelineContext.UpdatePipeline(p); err != nil {
		logrus.Errorf("fail update pipeline last run status,%v", err)
	}
	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "pipeline",
		Time:         time.Now(),
		Data:         p,
	}
}

/*
func GetNextRunTime(pipeline *pipeline.Pipeline) int64 {
	nextRunTime := int64(0)
	if !pipeline.IsActivate {
		return nextRunTime
	}
	spec := pipeline.TriggerSpec
	schedule, err := cron.Parse(spec)
	if err != nil {
		logrus.Errorf("error parse cron exp,%v,%v", spec, err)
		return nextRunTime
	}
	nextRunTime = schedule.Next(time.Now()).UnixNano() / int64(time.Millisecond)
	cronRunner := MyAgent.cronRunners[pipeline.Id]
	if cronRunner == nil {
		return nextRunTime
	}
	entry := cronRunner.Cron.Entries()
	if len(entry) > 0 {
		nextRunTime = entry[0].Next.UnixNano() / int64(time.Millisecond)
	}
	return nextRunTime
}
*/
func (s *Server) CreatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	data, err := ioutil.ReadAll(req.Body)
	ppl := &pipeline.Pipeline{}
	logrus.Infof("start create pipeline,get data:%v", string(data))
	if err := json.Unmarshal(data, ppl); err != nil {
		return err
	}
	//for pipelinefile import
	if ppl.Templates != nil && len(ppl.Templates) > 0 {
		templateContent := ""
		//TODO batch import
		for _, v := range ppl.Templates {
			templateContent = v
			break
		}
		if templateContent == "" {
			return fmt.Errorf("got empty pipeline file")
		}
		if err := yaml.Unmarshal([]byte(templateContent), &ppl.PipelineContent); err != nil {
			return err
		}
		pipeline.Clean(ppl)
		logrus.Debugf("got imported pipeline:\n%v", ppl)
	}

	if err := pipeline.Validate(ppl); err != nil {
		return err
	}
	//valid git account access
	if !validAccountAccess(req, ppl.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", ppl.Stages[0].Steps[0].GitUser)
	}

	ppl.Id = uuid.Rand().Hex()
	ppl.WebHookToken = uuid.Rand().Hex()
	//TODO Multiple
	gitUser := ppl.Stages[0].Steps[0].GitUser
	token, err := GetUserToken(gitUser)
	if err != nil {
		return err
	}
	err = webhook.CreateWebhook(ppl, token)
	if err != nil {
		logrus.Errorf("fail createWebhook")
		return err
	}
	err = s.PipelineContext.CreatePipeline(ppl)
	if err != nil {
		return err
	}

	MyAgent.onPipelineChange(ppl)
	apiContext.Write(toPipelineResource(apiContext, ppl))
	return nil
}

func (s *Server) UpdatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	data, err := ioutil.ReadAll(req.Body)
	ppl := &pipeline.Pipeline{}
	if err := json.Unmarshal(data, ppl); err != nil {
		return err
	}
	if err := pipeline.Validate(ppl); err != nil {
		return err
	}
	//valid git account access
	if !validAccountAccess(req, ppl.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", ppl.Stages[0].Steps[0].GitUser)
	}
	//TODO Multiple
	gitUser := ppl.Stages[0].Steps[0].GitUser
	token, err := GetUserToken(gitUser)
	if err != nil {
		logrus.Error(err)
		return err
	}
	// Update webhook
	prevPipeline := s.PipelineContext.GetPipelineById(id)
	if prevPipeline.Stages[0].Steps[0].Webhook && !ppl.Stages[0].Steps[0].Webhook {
		if err = webhook.DeleteWebhook(prevPipeline, token); err != nil {
			logrus.Error(err)
		}
	} else if !prevPipeline.Stages[0].Steps[0].Webhook && ppl.Stages[0].Steps[0].Webhook {
		if err = webhook.CreateWebhook(ppl, token); err != nil {
			logrus.Error(err)
			return err
		}
	} else if prevPipeline.Stages[0].Steps[0].Webhook &&
		ppl.Stages[0].Steps[0].Webhook &&
		(prevPipeline.Stages[0].Steps[0].Repository != ppl.Stages[0].Steps[0].Repository) {
		if err = webhook.DeleteWebhook(prevPipeline, token); err != nil {
			logrus.Error(err)
		}
		if err = webhook.CreateWebhook(ppl, token); err != nil {
			logrus.Error(err)
			return err
		}
	}
	err = s.PipelineContext.UpdatePipeline(ppl)
	if err != nil {
		return err
	}

	MyAgent.onPipelineChange(ppl)
	apiContext.Write(toPipelineResource(apiContext, ppl))
	return nil
}

func (s *Server) DeletePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	ppl := s.PipelineContext.GetPipelineById(id)
	//valid git account access
	if !validAccountAccess(req, ppl.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", ppl.Stages[0].Steps[0].GitUser)
	}
	//TODO Multiple
	gitUser := ppl.Stages[0].Steps[0].GitUser
	token, err := GetUserToken(gitUser)
	err = webhook.DeleteWebhook(ppl, token)
	if err != nil {
		//log delete webhook failure but not block
		logrus.Errorf("fail to delete webhook for pipeline \"%v\",for %v", ppl.Name, err)
	}
	r, err := s.PipelineContext.DeletePipeline(id)
	if err != nil {
		return err
	}
	MyAgent.onPipelineDelete(r)
	apiContext.Write(toPipelineResource(apiContext, r))
	return nil
}

func (s *Server) ActivatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r := s.PipelineContext.GetPipelineById(id)
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", id)
		return err
	}
	//valid git account access
	if !validAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	r.IsActivate = true
	err := s.PipelineContext.UpdatePipeline(r)
	if err != nil {
		return err
	}
	MyAgent.onPipelineActivate(r)
	apiContext.Write(toPipelineResource(apiContext, r))
	return nil

}

func (s *Server) DeActivatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r := s.PipelineContext.GetPipelineById(id)
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", id)
		return err
	}
	//valid git account access
	if !validAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	r.IsActivate = false
	err := s.PipelineContext.UpdatePipeline(r)
	if err != nil {
		return err
	}
	MyAgent.onPipelineDeActivate(r)
	apiContext.Write(toPipelineResource(apiContext, r))
	return nil
}

func (s *Server) ExportPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r := s.PipelineContext.GetPipelineById(id)
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", id)
		rw.WriteHeader(http.StatusNotFound)
		apiContext.Write(&Error{
			Resource: client.Resource{
				Id:      uuid.Rand().Hex(),
				Type:    "error",
				Links:   map[string]string{},
				Actions: map[string]string{},
			},
			Status: http.StatusNotFound,
			Msg:    err.Error(),
			Code:   err.Error(),
		})
		return err
	}
	//valid git account access
	if !validAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	pipeline.Clean(r)
	content, err := yaml.Marshal(r.PipelineContent)
	if err != nil {
		return err
	}
	logrus.Debugf("get pipeline file:\n%s", string(content))
	fileName := fmt.Sprintf("pipeline-%s.yaml", r.Name)
	rw.Header().Add("Content-Disposition", "attachment; filename="+fileName)
	http.ServeContent(rw, req, fileName, time.Now(), bytes.NewReader(content))
	return nil
}

func (s *Server) RunPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r := s.PipelineContext.GetPipelineById(id)
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", id)
		rw.WriteHeader(http.StatusNotFound)
		apiContext.Write(&Error{
			Resource: client.Resource{
				Id:      uuid.Rand().Hex(),
				Type:    "error",
				Links:   map[string]string{},
				Actions: map[string]string{},
			},
			Status: http.StatusNotFound,
			Msg:    err.Error(),
			Code:   err.Error(),
		})
		return err
	}
	//valid git account access
	if !validAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	activity, err := s.PipelineContext.RunPipeline(id, pipeline.TriggerTypeManual)
	if err != nil {
		return err
	}
	//MyAgent.watchActivityC <- activity
	apiContext.Write(toActivityResource(apiContext, activity))
	return nil
}

func (s *Server) ListActivitiesOfPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	pId := mux.Vars(req)["id"]
	r := s.PipelineContext.GetPipelineById(pId)
	//valid git account access
	if r != nil && !validAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	filters := make(map[string]interface{})
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&v2client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		return err
	}
	var activities []interface{}
	for _, gobj := range goCollection.Data {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &pipeline.Activity{}
		json.Unmarshal(b, a)
		if a.Pipeline.Id != pId {
			continue
		}

		toActivityResource(apiContext, a)
		activities = append(activities, a)
	}

	//v2client here generates error?
	apiContext.Write(&client.GenericCollection{
		Data: activities,
	})

	return nil
}

//list available env vars
func (s *Server) ListEnvVars(rw http.ResponseWriter, req *http.Request) error {
	b, err := json.Marshal(pipeline.PreservedEnvs)
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
	activity, err := GetActivity(activityId, s.PipelineContext)
	if err != nil {
		return err
	}
	if stageOrdinal < 0 || stepOrdinal < 0 || stageOrdinal >= len(activity.ActivityStages) || stepOrdinal >= len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return errors.New("step index invalid")
	}
	startStep(&activity, stageOrdinal, stepOrdinal)
	if err = UpdateActivity(activity); err != nil {
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         activity,
	}
	return nil
}

//
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
	activity, err := GetActivity(activityId, s.PipelineContext)
	if err != nil {
		return err
	}
	if stageOrdinal < 0 || stepOrdinal < 0 || stageOrdinal >= len(activity.ActivityStages) || stepOrdinal >= len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return errors.New("step index invalid")
	}
	if status == "SUCCESS" {
		successStep(&activity, stageOrdinal, stepOrdinal)
		triggernext(&activity, stageOrdinal, stepOrdinal)
	} else if status == "FAILURE" {
		failStep(&activity, stageOrdinal, stepOrdinal)
	}

	//update commitinfo for SCM step
	if stageOrdinal == 0 && stepOrdinal == 0 {
		activity.CommitInfo = req.FormValue("GIT_COMMIT")
		activity.EnvVars["CICD_GIT_COMMIT"] = activity.CommitInfo
	}

	logrus.Debugln("HALF SUCCESS?")
	if err = UpdateActivity(activity); err != nil {
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         activity,
	}
	s.UpdateLastActivity(activity)

	if activity.Status == pipeline.ActivityFail || activity.Status == pipeline.ActivitySuccess {
		s.PipelineContext.Provider.OnActivityCompelte(&activity)
	}

	return nil
}

func startStep(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) {
	curTime := time.Now().UnixNano() / int64(time.Millisecond)
	stage := activity.ActivityStages[stageOrdinal]
	step := stage.ActivitySteps[stepOrdinal]
	step.StartTS = curTime
	step.Status = pipeline.ActivityStepBuilding
	stage.Status = pipeline.ActivityStageBuilding
	activity.Status = pipeline.ActivityBuilding
	if stepOrdinal == 0 {
		stage.StartTS = curTime
	}
}

func failStep(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	stage := activity.ActivityStages[stageOrdinal]
	step := stage.ActivitySteps[stepOrdinal]
	step.Status = pipeline.ActivityStepFail
	step.Duration = now - step.StartTS
	stage.Status = pipeline.ActivityStageFail
	stage.Duration = now - stage.StartTS
	activity.Status = pipeline.ActivityFail
	activity.StopTS = now
	activity.FailMessage = fmt.Sprintf("Execution fail in '%v' stage, step %v", stage.Name, stepOrdinal+1)
}

func successStep(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) {
	curTime := time.Now().UnixNano() / int64(time.Millisecond)
	stage := activity.ActivityStages[stageOrdinal]
	step := stage.ActivitySteps[stepOrdinal]
	step.Status = pipeline.ActivityStepSuccess
	step.Duration = curTime - step.StartTS
	if stage.Status == pipeline.ActivityStageFail {
		return
	}

	if IsStageSuccess(stage) {
		stage.Status = pipeline.ActivityStageSuccess
		stage.Duration = curTime - stage.StartTS
		if stageOrdinal == len(activity.ActivityStages)-1 {
			activity.Status = pipeline.ActivitySuccess
			activity.StopTS = curTime
		} else {
			nextStage := activity.ActivityStages[stageOrdinal+1]
			if nextStage.NeedApproval {
				nextStage.Status = pipeline.ActivityStagePending
				activity.Status = pipeline.ActivityPending
				activity.PendingStage = stageOrdinal + 1
			}
			/*
				else if pipeline.HasStageCondition(nextStage) {
					if err := MyAgent.Server.PipelineContext.Provider.RunStage(activity, stageOrdinal+1); err != nil {
						logrus.Errorf("run conditional stage '%s' got error:%v", nextStage.Name, err)
						//activity.Status = Error
						activity.FailMessage = fmt.Sprintf("run conditional stage '%s' got error:%v", nextStage.Name, err)
					}
				}
			*/
		}
	}

}

func triggernext(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) {
	logrus.Debugf("triggering next:%d,%d", stageOrdinal, stepOrdinal)
	if activity.Status == pipeline.ActivitySuccess ||
		activity.Status == pipeline.ActivityFail ||
		activity.Status == pipeline.ActivityPending ||
		activity.Status == pipeline.ActivityDenied ||
		activity.Status == pipeline.ActivityAbort {
		return
	}
	stage := activity.ActivityStages[stageOrdinal]
	if IsStageSuccess(stage) && stageOrdinal+1 < len(activity.ActivityStages) {
		nextStage := activity.ActivityStages[stageOrdinal+1]
		if err := MyAgent.Server.PipelineContext.Provider.RunStage(activity, stageOrdinal+1); err != nil {
			logrus.Errorf("trigger next stage '%s' got error:%v", nextStage.Name, err)
			//activity.Status = Error
			activity.FailMessage = fmt.Sprintf("trigger next stage '%s' got error:%v", nextStage.Name, err)
		}
		return
	}

	if !activity.Pipeline.Stages[stageOrdinal].Parallel {
		if err := MyAgent.Server.PipelineContext.Provider.RunStep(activity, stageOrdinal, stepOrdinal+1); err != nil {
			logrus.Errorf("trigger step #%d of '%s' got error:%v", stepOrdinal+2, stage.Name, err)
			//activity.Status = Error
			activity.FailMessage = fmt.Sprintf("trigger step #%d of '%s' got error:%v", stepOrdinal+2, stage.Name, err)
		}
	}
}

func IsStageSuccess(stage *pipeline.ActivityStage) bool {
	if stage == nil {
		return false
	}

	if stage.Status == pipeline.ActivityStageFail || stage.Status == pipeline.ActivityStageDenied {
		return false
	}
	successSteps := 0
	for _, step := range stage.ActivitySteps {
		if step.Status == pipeline.ActivityStepSuccess || step.Status == pipeline.ActivityStepSkip {
			successSteps++
		}
	}
	return successSteps == len(stage.ActivitySteps)
}

// GetStepLog gets running logs of a particular step
func (s *Server) GetStepLog(activityId string, stageOrdinal int, stepOrdinal int) (string, error) {
	activity, err := GetActivity(activityId, s.PipelineContext)
	if err != nil {
		return "", err
	}
	stageSize := len(activity.ActivityStages)

	if stageOrdinal >= stageSize {
		return "", errors.New("stage out of size")
	}
	stage := activity.ActivityStages[stageOrdinal]

	stepSize := len(stage.ActivitySteps)
	if stepOrdinal >= stepSize {
		return "", errors.New("step out of size")
	}
	step := stage.ActivitySteps[stepOrdinal]
	return step.Message, nil
}

func NewServer(pipelineContext *pipeline.PipelineContext) *Server {
	return &Server{
		PipelineContext: pipelineContext,
	}
}

func GetCurrentUser(cookies []*http.Cookie) (string, error) {

	client := &http.Client{}

	requestURL := config.Config.CattleUrl + "/accounts"

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		logrus.Infof("Cannot connect to the rancher server. Please check the rancher server URL")
		return "", err
	}

	//req.SetBasicAuth(config.Config.CattleAccessKey, config.Config.CattleSecretKey)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Infof("Cannot connect to the rancher server. Please check the rancher server URL")
		return "", err
	}
	defer resp.Body.Close()
	userid := resp.Header.Get("X-Api-User-Id")
	if userid == "" {
		logrus.Infof("Cannot get userid")
		err := errors.New("Forbidden")
		return "Forbidden", err

	}
	return userid, nil
}

func (s *Server) Debug(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debugf("get header:%v", req.Header)
	logrus.Debugf("get url:%v", req.RequestURI)
	logrus.Debugf("get formvalue:%v", req.Form)
	b, err := ioutil.ReadAll(req.Body)
	logrus.Infof("get first:\n%v", b)
	request, err := http.NewRequest("POST", "http://192.168.99.1:60080/v1/debug2", bytes.NewBuffer(b))
	if err != nil {
		logrus.Errorf("fail:%v", err)
		return err
	}
	client := &http.Client{}
	_, err = client.Do(request)
	if err != nil {
		logrus.Errorf("fail:%v", err)
		return err
	}
	return nil
}

func (s *Server) Debug2(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debugf("get header:%v", req.Header)
	logrus.Debugf("get url:%v", req.RequestURI)
	b, err := ioutil.ReadAll(req.Body)

	logrus.Infof("get b:\n%v", string(b))
	var requestBody interface{}
	err = json.Unmarshal(b, &requestBody)
	logrus.Infof("get unmarshal:\n%v", requestBody)
	if err != nil {
		return fmt.Errorf("Error unmarshalling request body in Execute handler: %v", err)
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("Body should be of type map[string]interface{}")
	}

	logrus.Infof("get payload:\n%v", string(payload))
	return err
}
