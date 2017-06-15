package restfulserver

import "net/http"
import "github.com/rancher/go-rancher/api"
import "github.com/rancher/go-rancher/client"

import "github.com/rancher/pipeline/pipeline"
import "github.com/gorilla/mux"
import "github.com/pkg/errors"
import "github.com/sluu99/uuid"

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
	apiContext.Write(toPipelineResourceWithoutActivities(apiContext, r))
	return nil
}

func (s *Server) CreatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	//Todo
	apiContext.Write(&Empty{})
	return nil
}

func (s *Server) RunPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	//Todo
	apiContext.Write(&Empty{})
	return nil
}

func (s *Server) SavePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	//Todo
	apiContext.Write(&Empty{})
	return nil
}

func NewServer(pipelineContext *pipeline.PipelineContext) *Server {
	return &Server{
		PipelineContext: pipelineContext,
	}
}
