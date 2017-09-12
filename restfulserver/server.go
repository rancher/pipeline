package restfulserver

import (
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
	"github.com/rancher/pipeline/storer"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

//Server rest api server
type Server struct {
	PipelineContext *pipeline.PipelineContext
}

//ListPipelines query List of pipelines
func (s *Server) ListPipelines(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiContext.Write(&client.GenericCollection{
		Data: toPipelineCollections(apiContext, s.PipelineContext.ListPipelines()),
	})
	return nil
}

func (s *Server) Webhook(rw http.ResponseWriter, req *http.Request) error {
	var signature string
	var event_type string

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

	id := mux.Vars(req)["id"]
	logrus.Infof("webhook trigger,id:%v,event:%v,signature:%v", id, event_type, signature)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	r := s.PipelineContext.GetPipelineById(id)
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", id)
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("pipeline not found!"))
		return err
	}
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
	_, err = s.PipelineContext.RunPipeline(id)
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

//update last activity info in the pipeline
func (s *Server) UpdateLastActivity(pId string) {
	logrus.Infof("begin UpdateLastActivity")
	p := s.PipelineContext.GetPipelineById(pId)
	if p == nil || p.LastRunId == "" {
		return
	}
	activityId := p.LastRunId
	activity, err := GetActivity(activityId, s.PipelineContext)
	if err != nil {
		logrus.Errorf("fail update pipeline:%v last run:%v status,%v", pId, activityId, err)
		return
	}
	p.LastRunStatus = activity.Status
	p.CommitInfo = activity.CommitInfo
	//TODO
	p.NextRunTime = pipeline.GetNextRunTime(p)
	err = s.PipelineContext.UpdatePipeline(p)
	if err != nil {
		logrus.Errorf("fail update pipeline last run status,%v", err)
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
	if err := pipeline.Validate(ppl); err != nil {
		return err
	}

	ppl.Id = uuid.Rand().Hex()
	ppl.WebHookToken = uuid.Rand().Hex()
	err = webhook.RenewWebhook(ppl, req)
	if err != nil {
		logrus.Errorf("fail renewWebhook")
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
	data, err := ioutil.ReadAll(req.Body)
	ppl := &pipeline.Pipeline{}
	if err := json.Unmarshal(data, ppl); err != nil {
		return err
	}
	if err := pipeline.Validate(ppl); err != nil {
		return err
	}
	err = webhook.RenewWebhook(ppl, req)
	if err != nil && err != webhook.ErrDelWebhook {
		//fail to create webhook.block update
		return err
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
	err := webhook.DeleteWebhook(ppl)
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
	r.IsActivate = false
	err := s.PipelineContext.UpdatePipeline(r)
	if err != nil {
		return err
	}
	MyAgent.onPipelineDeActivate(r)
	apiContext.Write(toPipelineResource(apiContext, r))
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
	activity, err := s.PipelineContext.RunPipeline(id)
	if err != nil {
		return err
	}
	//MyAgent.watchActivityC <- activity
	apiContext.Write(toActivityResource(apiContext, activity))
	return nil
}

func (s *Server) SavePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	ppl := pipeline.Pipeline{}

	if err := json.Unmarshal(requestBytes, &ppl); err != nil {
		return err
	}
	st := &storer.LocalStorer{}
	yamlBytes, err := yaml.Marshal(ppl)
	if err != nil {
		return err
	}
	st.SavePipelineFile(ppl.Name, string(yamlBytes))
	//Todo
	apiContext.Write(&Empty{})
	return nil
}

func (s *Server) ListActivitiesOfPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	pId := mux.Vars(req)["id"]
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
	} else if status == "FAILURE" {
		failStep(&activity, stageOrdinal, stepOrdinal)
	}
	logrus.Infoln("HALF SUCCESS?")
	if err = UpdateActivity(activity); err != nil {
		return err
	}

	s.UpdateLastActivity(activity.Pipeline.Id)

	if activity.Status == pipeline.ActivityFail || activity.Status == pipeline.ActivitySuccess {
		s.PipelineContext.Provider.OnActivityCompelte(&activity)
	}

	return nil
}

func startStep(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) {
	curTime := time.Now().Unix()
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
	stage := activity.ActivityStages[stageOrdinal]
	stage.ActivitySteps[stepOrdinal].Status = pipeline.ActivityStepFail
	stage.Status = pipeline.ActivityStageFail
	activity.Status = pipeline.ActivityFail
	activity.FailMessage = fmt.Sprintf("Execution fail in '%v' stage, step %v", stage.Name, stepOrdinal+1)
}

func successStep(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) {
	curTime := time.Now().Unix()
	stage := activity.ActivityStages[stageOrdinal]
	step := stage.ActivitySteps[stepOrdinal]
	step.Status = pipeline.ActivityStepSuccess
	step.Duration = curTime - step.StartTS
	if stepOrdinal == len(stage.ActivitySteps)-1 {
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
		}
	}

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
