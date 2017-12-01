package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	v2client "github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/server/service"
	"github.com/rancher/pipeline/server/webhook"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
	yaml "gopkg.in/yaml.v2"
)

//ListPipelines query List of pipelines
func (s *Server) ListPipelines(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	uid, err := util.GetCurrentUser(req.Cookies())
	if err != nil || uid == "" {
		logrus.Debugf("getAccessibleAccounts unrecognized user")
	}
	pipelines := service.ListPipelines()

	apiContext.Write(&client.GenericCollection{
		Data: model.ToPipelineCollections(apiContext, pipelines),
	})
	return nil
}

func (s *Server) ListPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r, err := service.GetPipelineById(id)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	apiContext.Write(model.ToPipelineResource(apiContext, r))
	return nil
}

func (s *Server) CreatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	data, err := ioutil.ReadAll(req.Body)
	ppl := &model.Pipeline{}
	logrus.Debugf("start create pipeline,get data:%v", string(data))
	if err := json.Unmarshal(data, ppl); err != nil {
		return err
	}
	//for pipelinefile import
	if ppl.Templates != nil && len(ppl.Templates) > 0 {
		templateContent := ""
		//TODO batch import
		for _, v := range ppl.Templates {
			templateContent = v
			break
		}
		if templateContent == "" {
			return fmt.Errorf("got empty pipeline file")
		}
		if err := yaml.Unmarshal([]byte(templateContent), &ppl.PipelineContent); err != nil {
			return err
		}
		logrus.Debugf("got imported pipeline:\n%v", ppl)
	}
	service.CleanPipeline(ppl)

	if err := service.Validate(ppl); err != nil {
		return err
	}
	//valid git account access
	if !service.ValidAccountAccess(req, ppl.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", ppl.Stages[0].Steps[0].GitUser)
	}

	ppl.Id = uuid.Rand().Hex()
	ppl.WebHookToken = uuid.Rand().Hex()
	gitUser := ppl.Stages[0].Steps[0].GitUser
	token, err := service.GetUserToken(gitUser)
	if err != nil {
		return err
	}
	scManager, err := service.GetSCManagerFromUserID(gitUser)
	if err != nil {
		return err
	}

	if err = scManager.CreateWebhook(ppl, token, webhook.CIWebhookEndpoint); err != nil {
		logrus.Errorf("fail createWebhook")
		return err
	}
	if err = service.CreatePipeline(ppl); err != nil {
		return err
	}

	GlobalAgent.onPipelineChange(ppl)
	apiContext.Write(model.ToPipelineResource(apiContext, ppl))
	return nil
}

func (s *Server) UpdatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	data, err := ioutil.ReadAll(req.Body)
	ppl := &model.Pipeline{}
	if err := json.Unmarshal(data, ppl); err != nil {
		return err
	}
	if err := service.Validate(ppl); err != nil {
		return err
	}
	//valid git account access
	if !service.ValidAccountAccess(req, ppl.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", ppl.Stages[0].Steps[0].GitUser)
	}
	gitUser := ppl.Stages[0].Steps[0].GitUser
	token, err := service.GetUserToken(gitUser)
	if err != nil {
		logrus.Error(err)
		return fmt.Errorf("no access to '%s' git account", ppl.Stages[0].Steps[0].GitUser)
	}
	scManager, err := service.GetSCManagerFromUserID(gitUser)
	if err != nil {
		return err
	}
	// Update webhook
	prevPipeline, err := service.GetPipelineById(id)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	if prevPipeline.Stages[0].Steps[0].Webhook && !ppl.Stages[0].Steps[0].Webhook {
		if err = scManager.DeleteWebhook(prevPipeline, token); err != nil {
			logrus.Error(err)
		}
	} else if !prevPipeline.Stages[0].Steps[0].Webhook && ppl.Stages[0].Steps[0].Webhook {
		if err = scManager.CreateWebhook(ppl, token, webhook.CIWebhookEndpoint); err != nil {
			logrus.Error(err)
			return err
		}
	} else if prevPipeline.Stages[0].Steps[0].Webhook &&
		ppl.Stages[0].Steps[0].Webhook &&
		(prevPipeline.Stages[0].Steps[0].Repository != ppl.Stages[0].Steps[0].Repository) {
		if err = scManager.DeleteWebhook(prevPipeline, token); err != nil {
			logrus.Error(err)
		}
		if err = scManager.CreateWebhook(ppl, token, webhook.CIWebhookEndpoint); err != nil {
			logrus.Error(err)
			return err
		}
	}
	err = service.UpdatePipeline(ppl)
	if err != nil {
		return err
	}

	GlobalAgent.onPipelineChange(ppl)
	apiContext.Write(model.ToPipelineResource(apiContext, ppl))
	return nil
}

func (s *Server) DeletePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	ppl, err := service.GetPipelineById(id)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	//valid git account access
	if !service.ValidAccountAccess(req, ppl.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", ppl.Stages[0].Steps[0].GitUser)
	}

	gitUser := ppl.Stages[0].Steps[0].GitUser
	token, err := service.GetUserToken(gitUser)
	scManager, err := service.GetSCManagerFromUserID(gitUser)
	if err != nil {
		logrus.Error(err)
	} else if err = scManager.DeleteWebhook(ppl, token); err != nil {
		//log delete webhook failure but not block
		logrus.Errorf("fail to delete webhook for pipeline \"%v\",for %v", ppl.Name, err)
	}
	r, err := service.DeletePipeline(id)
	if err != nil {
		return err
	}
	GlobalAgent.onPipelineDelete(r)
	apiContext.Write(model.ToPipelineResource(apiContext, r))
	return nil
}

func (s *Server) ActivatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r, err := service.GetPipelineById(id)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	//valid git account access
	if !service.ValidAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	r.IsActivate = true
	err = service.UpdatePipeline(r)
	if err != nil {
		return err
	}
	GlobalAgent.onPipelineActivate(r)
	apiContext.Write(model.ToPipelineResource(apiContext, r))
	return nil

}

func (s *Server) DeActivatePipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r, err := service.GetPipelineById(id)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	//valid git account access
	if !service.ValidAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	r.IsActivate = false
	err = service.UpdatePipeline(r)
	if err != nil {
		return err
	}
	GlobalAgent.onPipelineDeActivate(r)
	apiContext.Write(model.ToPipelineResource(apiContext, r))
	return nil
}

func (s *Server) ExportPipeline(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	r, err := service.GetPipelineById(id)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	//valid git account access
	if !service.ValidAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	service.CleanPipeline(r)
	content, err := yaml.Marshal(r.PipelineContent)
	if err != nil {
		return err
	}
	logrus.Debugf("get pipeline file:\n%s", string(content))
	fileName := fmt.Sprintf("pipeline-%s.yaml", r.Name)
	rw.Header().Add("Content-Disposition", "attachment; filename="+fileName)
	http.ServeContent(rw, req, fileName, time.Now(), bytes.NewReader(content))
	return nil
}

func (s *Server) RunPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r, err := service.GetPipelineById(id)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	//valid git account access
	if !service.ValidAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	activity, err := service.RunPipeline(s.Provider, id, model.TriggerTypeManual)
	if err != nil {
		return err
	}
	apiContext.Write(model.ToActivityResource(apiContext, activity))
	return nil
}

func (s *Server) ListActivitiesOfPipeline(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	pId := mux.Vars(req)["id"]
	r, err := service.GetPipelineById(pId)
	if err != nil {
		return fmt.Errorf("fail to get pipeline: %v", err)
	}
	//valid git account access
	if !service.ValidAccountAccess(req, r.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Stages[0].Steps[0].GitUser)
	}
	filters := make(map[string]interface{})
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&v2client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		return err
	}
	var activities []interface{}
	for _, gobj := range goCollection.Data {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &model.Activity{}
		json.Unmarshal(b, a)
		if a.Pipeline.Id != pId {
			continue
		}

		model.ToActivityResource(apiContext, a)
		activities = append(activities, a)
	}

	//v2client here generates error?
	apiContext.Write(&client.GenericCollection{
		Data: activities,
	})

	return nil
}
