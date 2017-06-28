package restfulserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/storer"
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

func (s *Server) ListPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	name := mux.Vars(req)["id"]
	nameAndVersion := strings.Split(name, ":")
	var r *pipeline.Pipeline
	if len(nameAndVersion) == 2 {
		r = s.PipelineContext.GetPipelineByNameAndVersion(nameAndVersion[0], nameAndVersion[1])
	} else {
		r = s.PipelineContext.GetPipelineByName(name)
	}
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", name)
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

func (s *Server) CreatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	data, err := ioutil.ReadAll(req.Body)
	pipeline := pipeline.Pipeline{}

	if err := json.Unmarshal(data, &pipeline); err != nil {
		return err
	}
	err = s.PipelineContext.SavePipeline(pipeline)
	if err != nil {
		return err
	}

	apiContext.Write(toPipelineResource(apiContext, &pipeline))
	return nil
}

func (s *Server) RunPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	name := mux.Vars(req)["id"]
	r := s.PipelineContext.GetPipelineByName(name)
	if r == nil {
		err := errors.Wrapf(pipeline.ErrPipelineNotFound, "pipeline <%s>", name)
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
	s.PipelineContext.RunPipeline(name)
	apiContext.Write(&Empty{})
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
	apiContext.Write(&client.GenericCollection{
		Data: []interface{}{
			toActivityResource(apiContext, pipeline.ToDemoActivity()),
		},
	})
	return nil
}

func NewServer(pipelineContext *pipeline.PipelineContext) *Server {
	return &Server{
		PipelineContext: pipelineContext,
	}
}
