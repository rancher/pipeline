package webhook

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/config"
	"github.com/rancher/pipeline/util"
)

var (
	ErrDelWebhook    = errors.New("delete webhook fail")
	ErrCreateWebhook = errors.New("create webhook fail")

	CIWebhookEndpoint = ""
)

const CIWEBHOOKTYPE = "forwardPost"
const CIWEBHOOKNAME = "CIEndpoint"

type WebhookGenericObject struct {
	ID     string
	Name   string
	State  string
	Links  map[string]string
	Driver string
	URL    string
	Key    string
	Config interface{} `json:"forwardPostConfig"`
}

func ConvertToWebhookGenericObject(genericObject client.GenericObject) (WebhookGenericObject, error) {
	d, ok := genericObject.ResourceData["driver"].(string)
	if !ok {
		return WebhookGenericObject{}, fmt.Errorf("Couldn't read webhook data. Bad driver")
	}

	url, ok := genericObject.ResourceData["url"].(string)
	if !ok {
		return WebhookGenericObject{}, fmt.Errorf("Couldn't read webhook data. Bad url")
	}

	config, ok := genericObject.ResourceData["config"]
	if !ok {
		return WebhookGenericObject{}, fmt.Errorf("Couldn't read webhook data. Bad config on resource")
	}

	return WebhookGenericObject{
		Name:   genericObject.Name,
		ID:     genericObject.Id,
		State:  genericObject.State,
		Links:  genericObject.Links,
		Driver: d,
		URL:    url,
		Key:    genericObject.Key,
		Config: config,
	}, nil
}

func CreateCIEndpointWebhook() error {
	projectId, err := util.GetProjectId()
	if err != nil {
		return err
	}
	wh := WebhookGenericObject{
		Driver: CIWEBHOOKTYPE,
		Name:   CIWEBHOOKNAME,
		Config: map[string]string{
			"projectId":   projectId,
			"serviceName": "pipeline-server",
			"port":        "60080",
			"path":        "/v1/webhook",
		},
	}
	apiClient, err := util.GetRancherClient()

	if err != nil {
		return err
	}
	u, _ := url.Parse(config.Config.CattleUrl)
	createWebhookUrl := fmt.Sprintf("%s://%s/v1-webhooks/receivers?projectId=%s", u.Scheme, u.Host, projectId)
	if err = apiClient.Post(createWebhookUrl, wh, &wh); err != nil {
		logrus.Error(err)
		return err
	}

	logrus.Infof("Created and Using webhook '%s' as CI Endpoint.", wh.URL)
	//get CIWebhookEndpoint
	CIWebhookEndpoint = wh.URL
	return nil
}
