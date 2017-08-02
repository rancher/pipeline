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
			//	apiContext := api.GetApiContext(req)
			logrus.Errorf("fail in apihandler,%v", err)
			rw.WriteHeader(500)
			rw.Write([]byte(err.Error()))

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
	router.Methods(http.MethodDelete).Path("/v1/pipeline").Handler(f(schemas, s.CleanPipelines))
	//activities
	router.Methods(http.MethodGet).Path("/v1/activitys").Handler(f(schemas, s.ListActivities))
	router.Methods(http.MethodGet).Path("/v1/activities").Handler(f(schemas, s.ListActivities))
	router.Methods(http.MethodPost).Path("/v1/activity").Handler(f(schemas, s.CreateActivity))
	router.Methods(http.MethodGet).Path("/v1/activitys/{id}").Handler(f(schemas, s.GetActivity))
	//router.Methods(http.MethodPost).Path("/v1/activitys/{id}").Handler(f(schemas, s.UpdateActivity))
	//router.Methods(http.MethodPost).Path("/v1/activitys/{id}").Handler(f(schemas, s.ActivatePipeline))
	//router.Methods(http.MethodPost).Path("/v1/activitys/{id}").Handler(f(schemas, s.DeActivatePipeline))
	router.Methods(http.MethodDelete).Path("/v1/activity").Handler(f(schemas, s.CleanActivities))

	//test websocket
	router.Methods(http.MethodGet).Path("/v1/ws/log").Handler(f(schemas, s.ServeStepLog))
	router.Methods(http.MethodGet).Path("/v1/ws/status").Handler(f(schemas, s.ServeStatusWS))

	pipelineActions := map[string]http.Handler{
		"run":        f(schemas, s.RunPipeline),
		"update":     f(schemas, s.UpdatePipeline),
		"activate":   f(schemas, s.ActivatePipeline),
		"deactivate": f(schemas, s.DeActivatePipeline),
		"remove":     f(schemas, s.DeletePipeline),
	}
	for name, actions := range pipelineActions {
		router.Methods(http.MethodPost).Path("/v1/pipelines/{id}").Queries("action", name).Handler(actions)
	}

	activityActions := map[string]http.Handler{
		"update":  f(schemas, s.UpdateActivity),
		"remove":  f(schemas, s.DeleteActivity),
		"approve": f(schemas, s.ApproveActivity),
		"deny":    f(schemas, s.DenyActivity),
	}
	for name, actions := range activityActions {
		router.Methods(http.MethodPost).Path("/v1/activitys/{id}").Queries("action", name).Handler(actions)
	}
	return router
}
