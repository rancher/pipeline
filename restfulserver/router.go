package restfulserver

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
)

//HandleError handle error from operation
func HandleError(s *client.Schemas, t func(http.ResponseWriter, *http.Request) error) http.Handler {
	return api.ApiHandler(s, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if err := t(rw, req); err != nil {
			// apiContext := api.GetApiContext(req)
			// apiContext.Write(err)
			//apiContext.WriteErr(err)
			logrus.Error(err)
		}
	}))
}

//NewRouter router for schema
func NewRouter(s *Server) *mux.Router {
	schemas := NewSchema()
	router := mux.NewRouter().StrictSlash(true)
	f := HandleError
	// API framework routes
	router.Methods(http.MethodGet).Path("/").Handler(api.VersionsHandler(schemas, "v1"))
	router.Methods(http.MethodGet).Path("/v1/schemas").Handler(api.SchemasHandler(schemas))
	router.Methods(http.MethodGet).Path("/v1/schemas/{id}").Handler(api.SchemaHandler(schemas))
	router.Methods(http.MethodGet).Path("/v1").Handler(api.VersionHandler(schemas, "v1"))

	router.Methods(http.MethodGet).Path("/v1/pipelines").Handler(f(schemas, s.ListPipelines))
	router.Methods(http.MethodPost).Path("/v1/pipeline").Handler(f(schemas, s.CreatePipeline))
	router.Methods(http.MethodGet).Path("/v1/pipelines/{id}").Handler(f(schemas, s.ListPipeline))
	//router.Methods(http.MethodPost).Path("/v1/pipelines/{id}").Handler(f(schemas, s.UpdatePipeline))
	router.Methods(http.MethodGet).Path("/v1/pipelines/{id}/activitys").Handler(f(schemas, s.ListActivitiesOfPipeline))
	router.Methods(http.MethodDelete).Path("/v1/pipelines/{id}").Handler(f(schemas, s.DeletePipeline))
	//activities

	router.Methods(http.MethodGet).Path("/v1/activities/{id}").Handler(f(schemas, s.GetActivity))
	router.Methods(http.MethodPost).Path("/v1/activities/{id}").Handler(f(schemas, s.UpdateActivity))
	router.Methods(http.MethodPost).Path("/v1/activity/").Handler(f(schemas, s.CreateActivity))
	router.Methods(http.MethodGet).Path("/v1/activities/").Handler(f(schemas, s.ListActivities))

	pipelineActions := map[string]http.Handler{
		"run":    f(schemas, s.RunPipeline),
		"update": f(schemas, s.UpdatePipeline),
	}
	for name, actions := range pipelineActions {
		router.Methods(http.MethodPost).Path("/v1/pipelines/{id}").Queries("action", name).Handler(actions)
	}
	return router
}
