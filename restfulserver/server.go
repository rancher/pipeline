package restfulserver

import "net/http"
import "github.com/rancher/go-rancher/api"
import "github.com/rancher/go-rancher/client"

//Server rest api server
type Server struct {
}

//ListPipelines query List of pipelines
func (s *Server) ListPipelines(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiContext.Write(&client.GenericCollection{
		Data: []interface{}{
			toPipelineResource(),
		},
	})
	return nil
}

func NewServer() *Server {
	return &Server{}
}
