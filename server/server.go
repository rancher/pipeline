package server

import (
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	v2client "github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/server/service"
	"github.com/rancher/pipeline/server/webhook"
	"github.com/rancher/pipeline/util"
)

//Server rest api server
type Server struct {
	Provider   model.PipelineProvider
	SCManagers map[string]model.SCManager
}

func NewServer(provider model.PipelineProvider) *Server {
	s := &Server{
		Provider: provider,
	}
	return s
}

func ListenAndServe(provider model.PipelineProvider, errChan chan bool) {
	server := NewServer(provider)
	Preset(provider)
	InitAgent(server)
	router := http.Handler(NewRouter(server))
	router = proxyProtoHandler(router)
	router = handlers.LoggingHandler(os.Stdout, router)
	router = handlers.ProxyHeaders(router)
	if err := http.ListenAndServe(":60080", router); err != nil {
		logrus.Error(err)
		errChan <- true
	}
}

func Preset(provider model.PipelineProvider) {
	if err := checkCIEndpoint(); err != nil {
		logrus.Errorf("Check CI Endpoint Error:%v", err)
	}
	activities, err := service.ListActivities()
	if err != nil {
		logrus.Errorf("List activities Error:%v", err)
	}
	logrus.Debugf("get activities size:%v", len(activities))
	//Sync status of running activities
	for _, a := range activities {
		//TODO !a.IsRunning
		if a.Status == model.ActivityFail ||
			a.Status == model.ActivitySuccess ||
			a.Status == model.ActivityDenied ||
			a.Status == model.ActivityPending ||
			a.Status == model.ActivityAbort {
			continue
		}
		if err := provider.SyncActivity(a); err != nil {
			logrus.Errorf("Sync activity Error:%v", err)
			continue
		}
		if err := service.UpdateActivity(a); err != nil {
			logrus.Errorf("Update activity Error:%v", err)
		}
	}
}

func checkCIEndpoint() error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "webhookReceiver"
	opt := &v2client.ListOpts{
		Filters: filters,
	}
	gCollection, err := apiClient.GenericObject.List(opt)
	if err != nil {
		return err
	}
	var ciWebhook *webhook.WebhookGenericObject
	for _, wh := range gCollection.Data {
		whObject, err := webhook.ConvertToWebhookGenericObject(wh)
		if err != nil {
			logrus.Errorf("Preset get webhook endpoint error:%v", err)
			return err
		}
		if whObject.Driver == webhook.CIWEBHOOKTYPE && whObject.Name == webhook.CIWEBHOOKNAME {
			ciWebhook = &whObject
			//get CIWebhookEndpoint
			webhook.CIWebhookEndpoint = ciWebhook.URL
			logrus.Infof("Using webhook '%s' as CI Endpoint.", ciWebhook.URL)
		}
	}
	if ciWebhook == nil {
		//Create a webhook for github webhook payload url
		if err := webhook.CreateCIEndpointWebhook(); err != nil {
			logrus.Errorf("CreateCIEndpointWebhook Error:%v", err)
			return err
		}

	}
	return nil
}
