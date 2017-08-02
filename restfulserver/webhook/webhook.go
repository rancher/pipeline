package webhook

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/util"
)

var (
	ErrDelWebhook    = errors.New("delete webhook fail")
	ErrCreateWebhook = errors.New("create webhook fail")
)

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

func GetWebhookUrl(req *http.Request, pipelineId string) string {
	/*
		proto := "http://"
		if req.TLS != nil {
			proto = "https://"
		}*/
	host := os.Getenv("HOST_NAME")
	if host == "" {
		host = "<Proto://Host:Port>"
	}
	host = strings.TrimRight(host, "/")
	url := host + "/v1/webhook/" + pipelineId

	//logrus.Infof("get X-API-request-url:%v", req.Header.Get("X-API-request-url"))
	logrus.Infof("get webhook url:%v", url)

	return url

}

func CreateWebhook(p *pipeline.Pipeline, webhookUrl string) error {
	logrus.Infof("createwebhook for pipeline:%v", p.Id)
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
			logrus.Infof("get:%v,%v,%v,%v,%v,%v", user, repo, token, webhookUrl, secret, id)
			if err != nil {
				logrus.Errorf("error delete webhook,%v", err)
				return err
			}
			p.WebHookId = id
		}
	}
	return nil
}

func RenewWebhook(p *pipeline.Pipeline, req *http.Request) error {
	//update webhook in github repo
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		logrus.Infof("pipelinechange,webhook:%v,%v", p.Stages[0].Steps[0].Webhook, p.WebHookId)
		if p.Stages[0].Steps[0].Webhook {
			if p.WebHookId <= 0 {
				webhookUrl := GetWebhookUrl(req, p.Id)
				logrus.Infof("get webhookUrl:%v", webhookUrl)
				err := CreateWebhook(p, webhookUrl)
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
