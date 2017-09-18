package webhook

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
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
		Config: map[string]string{"serviceURL": fmt.Sprintf("http://localhost:8080/r/projects/%s/pipeline-server:60081/v1/webhook", projectId)}, //"http://pipeline-server:60080/v1/webhook"},
	}
	apiClient, err := util.GetRancherClient()

	if err != nil {
		return err
	}
	u, _ := url.Parse(config.Config.CattleUrl)
	createWebhookUrl := fmt.Sprintf("%s://%s/v1-webhooks/receivers?projectId=%s", u.Scheme, u.Host, projectId)
	//res := model.WebhookGenericObject
	if err = apiClient.Post(createWebhookUrl, wh, &wh); err != nil {
		logrus.Error(err)
		return err
	}
	//get CIWebhookEndpoint
	CIWebhookEndpoint = wh.URL
	return nil
}

func DeleteWebhook(p *pipeline.Pipeline) error {
	logrus.Infof("deletewebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to delete webhook")
	}

	//delete webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.WebHookId > 0 {
			//TODO
			repoUrl := p.Stages[0].Steps[0].Repository
			token := p.Stages[0].Steps[0].Token
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

func CreateWebhook(p *pipeline.Pipeline, webhookUrl string) error {
	logrus.Debugf("createwebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to create webhook")
	}

	//create webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.Stages[0].Steps[0].Webhook {
			//TODO
			repoUrl := p.Stages[0].Steps[0].Repository
			token := p.Stages[0].Steps[0].Token
			reg := regexp.MustCompile(".*?github.com/(.*?)/(.*?).git")
			match := reg.FindStringSubmatch(repoUrl)
			if len(match) < 3 {
				logrus.Errorf("error getting user/repo from gitrepoUrl:%v", repoUrl)
				return errors.New(fmt.Sprintf("error getting user/repo from gitrepoUrl:%v", repoUrl))
			}
			user := match[1]
			repo := match[2]
			secret := p.WebHookToken
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

func RenewWebhook(p *pipeline.Pipeline) error {
	//update webhook in github repo
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		logrus.Debugf("pipelinechange,webhook:%v,%v,%v", p.Stages[0].Steps[0].Webhook, p.WebHookId)
		if p.Stages[0].Steps[0].Webhook {
			if p.WebHookId <= 0 {
				payloadURL := fmt.Sprintf("%s&pipelineId=%s", CIWebhookEndpoint, p.Id)
				err := CreateWebhook(p, payloadURL)
				if err != nil {
					return ErrCreateWebhook
				}
			}
		} else {
			if p.WebHookId > 0 {
				err := DeleteWebhook(p)
				if err != nil {
					return ErrDelWebhook
				}
			}
		}

	}
	return nil

}
