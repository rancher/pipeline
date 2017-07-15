package restfulserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	yaml "gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	v2client "github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/pipeline"
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
	pCollections := s.PipelineContext.ListPipelines()
	for _, p := range pCollections {
		s.updateLastActivity(p)
	}
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
	s.updateLastActivity(r)
	apiContext.Write(toPipelineResource(apiContext, r))
	return nil
}

//update last activity info in the pipeline
func (s *Server) updateLastActivity(p *pipeline.Pipeline) {
	if p.LastRunId == "" {
		return
	}
	activity, err := GetActivity(p.LastRunId, s.PipelineContext)
	if err != nil {
		logrus.Error("fail to get last run of pipeline %v", p.Name)
		return
	}
	p.LastRunStatus = activity.Status
	p.CommitInfo = activity.CommitInfo
}

func (s *Server) CreatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	data, err := ioutil.ReadAll(req.Body)
	pipeline := pipeline.Pipeline{}
	logrus.Infof("start create pipeline,get data:%v", string(data))
	if err := json.Unmarshal(data, &pipeline); err != nil {
		return err
	}
	err = s.PipelineContext.CreatePipeline(pipeline)
	if err != nil {
		return err
	}

	apiContext.Write(toPipelineResource(apiContext, &pipeline))
	return nil
}

func (s *Server) UpdatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	data, err := ioutil.ReadAll(req.Body)
	pipeline := pipeline.Pipeline{}
	if err := json.Unmarshal(data, &pipeline); err != nil {
		return err
	}
	err = s.PipelineContext.UpdatePipeline(pipeline)
	if err != nil {
		return err
	}

	apiContext.Write(toPipelineResource(apiContext, &pipeline))
	return nil
}

func (s *Server) DeletePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r, err := s.PipelineContext.DeletePipeline(id)
	if err != nil {
		return err
	}
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
	r.RunCount = activity.RunSequence
	r.LastRunId = activity.Id
	r.LastRunStatus = activity.Status
	s.PipelineContext.UpdatePipeline(*r)
	MyAgent.ReWatch <- true
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
		//When get a unfinish Activity ,try to sync from provider and update its status
		/*
			if a.Status == "Waitting" || a.Status == "Building" {
				err = s.PipelineContext.SyncActivity(a)
				if err != nil {
					logrus.Error(err)
					//skip nonsync one
					continue
				}
				err = UpdateActivity(*a)
				if err != nil {
					logrus.Error(err)
					continue
				}
			}*/
		toActivityResource(apiContext, a)
		activities = append(activities, a)
	}

	//v2client here generates error?
	apiContext.Write(&client.GenericCollection{
		Data: activities,
	})

	return nil
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
