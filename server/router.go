package server

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/pipeline/model"
)

//HandleError handle error from operation
func HandleError(s *client.Schemas, t func(http.ResponseWriter, *http.Request) error) http.Handler {
	return api.ApiHandler(s, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if err := t(rw, req); err != nil {
			logrus.Errorf("Got Error: %v", err)
			rw.Header().Set("Content-Type", "application/json")
			StatusCode := 500
			rw.WriteHeader(StatusCode)
			e := model.Error{
				Resource: client.Resource{
					Type: "error",
				},
				Status: StatusCode,
				Msg:    err.Error(),
			}
			api.GetApiContext(req).Write(&e)
		}
	}))
}

//NewRouter router for schema
func NewRouter(s *Server) *mux.Router {
	schemas := model.NewSchema()
	router := mux.NewRouter().StrictSlash(true)
	f := HandleError
	// API framework routes
	router.Methods(http.MethodGet).Path("/").Handler(api.VersionHandler(schemas, "v1"))
	router.Methods(http.MethodGet).Path("/v1/schemas").Handler(api.SchemasHandler(schemas))
	router.Methods(http.MethodGet).Path("/v1/schemas/{id}").Handler(api.SchemaHandler(schemas))
	router.Methods(http.MethodGet).Path("/v1").Handler(api.VersionHandler(schemas, "v1"))

	//pipelines
	router.Methods(http.MethodGet).Path("/v1/pipelines").Handler(f(schemas, s.ListPipelines))
	router.Methods(http.MethodPost).Path("/v1/pipeline").Handler(f(schemas, s.CreatePipeline))
	router.Methods(http.MethodPost).Path("/v1/pipelines").Handler(f(schemas, s.CreatePipeline))
	router.Methods(http.MethodGet).Path("/v1/pipelines/{id}").Handler(f(schemas, s.ListPipeline))
	router.Methods(http.MethodGet).Path("/v1/pipelines/{id}/activities").Handler(f(schemas, s.ListActivitiesOfPipeline))
	router.Methods(http.MethodDelete).Path("/v1/pipelines/{id}").Handler(f(schemas, s.DeletePipeline))
	router.Methods(http.MethodGet).Path("/v1/pipelines/{id}/exportconfig").Handler(f(schemas, s.ExportPipeline))
	//router.Methods(http.MethodDelete).Path("/v1/pipeline").Handler(f(schemas, s.CleanPipelines))

	//activities
	router.Methods(http.MethodGet).Path("/v1/activities").Handler(f(schemas, s.ListActivities))
	router.Methods(http.MethodGet).Path("/v1/activities/{id}").Handler(f(schemas, s.GetActivity))
	router.Methods(http.MethodDelete).Path("/v1/activities/{id}").Handler(f(schemas, s.DeleteActivity))
	//router.Methods(http.MethodDelete).Path("/v1/activity").Handler(f(schemas, s.CleanActivities))

	//scm accounts
	router.Methods(http.MethodGet).Path("/v1/gitaccounts").Handler(f(schemas, s.ListAccounts))
	router.Methods(http.MethodGet).Path("/v1/gitaccounts/{id}").Handler(f(schemas, s.GetAccount))
	router.Methods(http.MethodGet).Path("/v1/gitaccounts/{id}/repos").Handler(f(schemas, s.GetCacheRepos))
	//settings
	router.Methods(http.MethodGet).Path("/v1/settings").Handler(f(schemas, s.GetPipelineSetting))
	router.Methods(http.MethodGet).Path("/v1/settings/scmsettings").Handler(f(schemas, s.ListSCMSetting))
	router.Methods(http.MethodGet).Path("/v1/scmsettings/{id}").Handler(f(schemas, s.GetSCMSetting))
	router.Methods(http.MethodGet).Path("/v1/scmsettings").Handler(f(schemas, s.ListSCMSetting))

	router.Methods(http.MethodGet).Path("/v1/envvars").Handler(f(schemas, s.ListEnvVars))

	//websockets
	router.Methods(http.MethodGet).Path("/v1/ws/log").Handler(f(schemas, s.ServeStepLog))
	router.Methods(http.MethodGet).Path("/v1/ws/status").Handler(f(schemas, s.ServeStatusWS))

	//callback path for jenkins events
	router.Methods(http.MethodPost).Path("/v1/events/stepfinish").Handler(f(schemas, s.StepFinish))
	router.Methods(http.MethodPost).Path("/v1/events/stepstart").Handler(f(schemas, s.StepStart))

	//webhook endpoint
	router.Methods(http.MethodPost).Path("/v1/webhook").Handler(f(schemas, s.Webhook))
	pipelineActions := map[string]http.Handler{
		"run":        f(schemas, s.RunPipeline),
		"update":     f(schemas, s.UpdatePipeline),
		"activate":   f(schemas, s.ActivatePipeline),
		"deactivate": f(schemas, s.DeActivatePipeline),
		"remove":     f(schemas, s.DeletePipeline),
		"export":     f(schemas, s.ExportPipeline),
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
		"stop":    f(schemas, s.StopActivity),
	}
	for name, actions := range activityActions {
		router.Methods(http.MethodPost).Path("/v1/activities/{id}").Queries("action", name).Handler(actions)
	}

	pipelineSettingActions := map[string]http.Handler{
		"update": f(schemas, s.UpdatePipelineSetting),
		"reset":  f(schemas, s.Reset),
		"oauth":  f(schemas, s.Oauth),
	}
	for name, actions := range pipelineSettingActions {
		router.Methods(http.MethodPost).Path("/v1/settings").Queries("action", name).Handler(actions)
	}

	scmSettingActions := map[string]http.Handler{
		"update": f(schemas, s.UpdateSCMSetting),
		"remove": f(schemas, s.RemoveSCMSetting),
	}
	for name, actions := range scmSettingActions {
		router.Methods(http.MethodPost).Path("/v1/scmsettings/{id}").Queries("action", name).Handler(actions)
	}

	accountActions := map[string]http.Handler{
		"share":        f(schemas, s.ShareAccount),
		"unshare":      f(schemas, s.UnshareAccount),
		"remove":       f(schemas, s.RemoveAccount),
		"refreshrepos": f(schemas, s.RefreshRepos),
	}
	for name, actions := range accountActions {
		router.Methods(http.MethodPost).Path("/v1/gitaccounts/{id}").Queries("action", name).Handler(actions)
	}
	return router
}
