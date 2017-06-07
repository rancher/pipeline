package restfulserver

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/pipeline/interceptor"
)

//HandleError handle error from operation
func HandleError(s *client.Schemas, t func(http.ResponseWriter, *http.Request) error) http.Handler {
	return api.ApiHandler(s, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if err := t(rw, req); err != nil {
			apiContext := api.GetApiContext(req)
			apiContext.Write(err)
			//apiContext.WriteErr(err)
		}
	}))
}

//NewRouter router for schema
func NewRouter(s *Server) *mux.Router {
	schemas := NewSchema()
	router := mux.NewRouter().StrictSlash(true)
	f := HandleError

	// for intercepter
	router.Methods(http.MethodPost).Path("/").HandlerFunc(interceptor.HandlerInterceptor)
	// API framework routes
	router.Methods(http.MethodGet).Path("/").Handler(api.VersionsHandler(schemas, "v1"))
	router.Methods(http.MethodGet).Path("/v1/schemas").Handler(api.SchemasHandler(schemas))
	router.Methods(http.MethodGet).Path("/v1/schemas/{id}").Handler(api.SchemaHandler(schemas))
	router.Methods(http.MethodGet).Path("/v1").Handler(api.VersionHandler(schemas, "v1"))

	router.Methods(http.MethodGet).Path("/v1/pipelines").Handler(f(schemas, s.ListPipelines))
	router.Methods(http.MethodGet).Path("/v1/pipelines/{id}").Handler(f(schemas, s.ListPipeline))

	router.Methods(http.MethodGet).Path("/v1/activities/").Handler(f(schemas, s.ListActivities))
	//for test
	router.Methods(http.MethodGet).Path("/v2-beta/projects/{env}/schemas/{id}").Handler(api.SchemaHandler(schemas))
	router.Methods(http.MethodGet).Path("/v2-beta/projects/{env}/pipelines").Handler(f(schemas, s.ListPipelines))
	router.Methods(http.MethodGet).Path("/v2-beta/projects/{env}/pipelines/{id}").Handler(f(schemas, s.ListPipeline))
	router.Methods(http.MethodGet).Path("/v2-beta/projects/{env}/activities/").Handler(f(schemas, s.ListActivities))
	router.Methods(http.MethodPost).Path("/v1/pipelines/").Handler(f(schemas, s.CreatePipelineWithXML))
	// Volume
	// router.Methods("GET").Path("/v1/volumes").Handler(f(schemas, s.ListVolumes))
	// router.Methods("GET").Path("/v1/volumes/{id}").Handler(f(schemas, s.GetVolume))
	// router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "readat").Handler(f(schemas, s.ReadAt))
	// router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "writeat").Handler(f(schemas, s.WriteAt))

	return router
}
