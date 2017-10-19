package webhook

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/pipeline/config"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/util"
)

var (
	ErrDelWebhook    = errors.New("delete webhook fail")
	ErrCreateWebhook = errors.New("create webhook fail")

	CIWebhookEndpoint = ""
)

const CIWEBHOOKTYPE = "serviceWebhook"
const CIWEBHOOKNAME = "CIEndpoint"

type WebhookGenericObject struct {
	ID     string
	Name   string
	State  string
	Links  map[string]string
	Driver string
	URL    string
	Key    string
	Config interface{} `json:"serviceWebhookConfig"`
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
			"serviceName": "pipeline-server",
			"port":        "60081",
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

func DeleteWebhook(p *pipeline.Pipeline, token string) error {
	logrus.Infof("deletewebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to delete webhook")
	}

	//delete webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.WebHookId > 0 {
			//TODO
			repoUrl := p.Stages[0].Steps[0].Repository
			reg := regexp.MustCompile(".*?github.com/(.*?)/(.*?).git")
			match := reg.FindStringSubmatch(repoUrl)
			if len(match) != 3 {
				logrus.Infof("get match:%v", match)
				logrus.Errorf("error getting user/repo from gitrepoUrl:%v", repoUrl)
				return errors.New(fmt.Sprintf("error getting user/repo from gitrepoUrl:%v", repoUrl))
			}
			user := match[1]
			repo := match[2]
			err := util.DeleteWebhook(user, repo, token, p.WebHookId)
			if err != nil {
				logrus.Errorf("error delete webhook,%v", err)
				return err
			}
			p.WebHookId = 0
		}
	}
	return nil
}

func CreateWebhook(p *pipeline.Pipeline, token string) error {
	logrus.Debugf("createwebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to create webhook")
	}

	//create webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.Stages[0].Steps[0].Webhook {
			repoUrl := p.Stages[0].Steps[0].Repository
			reg := regexp.MustCompile(".*?github.com/(.*?)/(.*?).git")
			match := reg.FindStringSubmatch(repoUrl)
			if len(match) < 3 {
				logrus.Errorf("error getting user/repo from gitrepoUrl:%v", repoUrl)
				return errors.New(fmt.Sprintf("error getting user/repo from gitrepoUrl:%v", repoUrl))
			}
			user := match[1]
			repo := match[2]
			secret := p.WebHookToken
			webhookUrl := fmt.Sprintf("%s&pipelineId=%s", CIWebhookEndpoint, p.Id)
			id, err := util.CreateWebhook(user, repo, token, webhookUrl, secret)
			logrus.Debugf("Creating webhook:%v,%v,%v,%v,%v,%v", user, repo, token, webhookUrl, secret, id)
			if err != nil {
				logrus.Errorf("error delete webhook,%v", err)
				return err
			}
			p.WebHookId = id
		}
	}
	return nil
}
