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
			logrus.Errorf("Got Error: %v", err)
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(500)

			e := Error{
				Resource: client.Resource{
					Type: "error",
				},
				Status: 500,
				Msg:    err.Error(),
			}
			api.GetApiContext(req).Write(&e)
		}
	}))
}

//NewRouter router for schema
func NewRouter(s *Server) *mux.Router {
	schemas := NewSchema()
	router := mux.NewRouter().StrictSlash(true)
	f := HandleError
	// API framework routes
	router.Methods(http.MethodGet).Path("/").Handler(api.VersionHandler(schemas, "v1"))
	router.Methods(http.MethodGet).Path("/v1/schemas").Handler(api.SchemasHandler(schemas))
	router.Methods(http.MethodGet).Path("/v1/schemas/{id}").Handler(api.SchemaHandler(schemas))
	router.Methods(http.MethodGet).Path("/v1").Handler(api.VersionHandler(schemas, "v1"))

	//pipelines
	router.Methods(http.MethodGet).Path("/v1/pipelines").Handler(f(schemas, s.ListPipelines))
	router.Methods(http.MethodGet).Path("/v1/pipeline").Handler(f(schemas, s.ListPipelines))
	router.Methods(http.MethodPost).Path("/v1/pipelines").Handler(f(schemas, s.CreatePipeline))
	router.Methods(http.MethodPost).Path("/v1/pipeline").Handler(f(schemas, s.CreatePipeline))
	router.Methods(http.MethodGet).Path("/v1/pipelines/{id}").Handler(f(schemas, s.ListPipeline))
	router.Methods(http.MethodGet).Path("/v1/pipelines/{id}/activities").Handler(f(schemas, s.ListActivitiesOfPipeline))
	router.Methods(http.MethodDelete).Path("/v1/pipelines/{id}").Handler(f(schemas, s.DeletePipeline))
	//router.Methods(http.MethodDelete).Path("/v1/pipeline").Handler(f(schemas, s.CleanPipelines))

	//activities
	router.Methods(http.MethodGet).Path("/v1/activities").Handler(f(schemas, s.ListActivities))
	router.Methods(http.MethodGet).Path("/v1/activity").Handler(f(schemas, s.ListActivities))
	router.Methods(http.MethodPost).Path("/v1/activities").Handler(f(schemas, s.CreateActivity))
	router.Methods(http.MethodPost).Path("/v1/activity").Handler(f(schemas, s.CreateActivity))
	router.Methods(http.MethodGet).Path("/v1/activities/{id}").Handler(f(schemas, s.GetActivity))
	router.Methods(http.MethodDelete).Path("/v1/activities/{id}").Handler(f(schemas, s.DeleteActivity))
	//router.Methods(http.MethodDelete).Path("/v1/activity").Handler(f(schemas, s.CleanActivities))

	//settings
	router.Methods(http.MethodGet).Path("/v1/settings").Handler(f(schemas, s.GetPipelineSetting))
	router.Methods(http.MethodGet).Path("/v1/setting").Handler(f(schemas, s.GetPipelineSetting))
	//router.Methods(http.MethodPost).Path("/v1/settings").Handler(f(schemas, s.UpdatePipelineSetting))
	//router.Methods(http.MethodPost).Path("/v1/setting").Handler(f(schemas, s.UpdatePipelineSetting))
	router.Methods(http.MethodGet).Path("/v1/envvars").Handler(f(schemas, s.ListEnvVars))

	//websockets
	router.Methods(http.MethodGet).Path("/v1/ws/log").Handler(f(schemas, s.ServeStepLog))
	router.Methods(http.MethodGet).Path("/v1/ws/status").Handler(f(schemas, s.ServeStatusWS))

	//callback path for jenkins events
	router.Methods(http.MethodPost).Path("/v1/events/stepfinish").Handler(f(schemas, s.StepFinish))
	router.Methods(http.MethodPost).Path("/v1/events/stepstart").Handler(f(schemas, s.StepStart))

	router.Methods(http.MethodPost).Path("/v1/github/login").Handler(f(schemas, s.GithubLogin))
	router.Methods(http.MethodPost).Path("/v1/github/oauth").Handler(f(schemas, s.GithubAuthorize))

	//debug
	//router.Methods(http.MethodPost).Path("/v1/debug").Handler(f(schemas, s.Debug))
	//router.Methods(http.MethodPost).Path("/v1/debug2").Handler(f(schemas, s.Debug2))

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
		"rerun":   f(schemas, s.RerunActivity),
	}
	for name, actions := range activityActions {
		router.Methods(http.MethodPost).Path("/v1/activities/{id}").Queries("action", name).Handler(actions)
	}

	pipelineSettingActions := map[string]http.Handler{
		"update":      f(schemas, s.UpdatePipelineSetting),
		"githuboauth": f(schemas, s.GithubAuthorize),
		"getrepos":    f(schemas, s.GithubGetRepos),
	}
	for name, actions := range pipelineSettingActions {
		router.Methods(http.MethodPost).Path("/v1/settings").Queries("action", name).Handler(actions)
	}
	return router
}
