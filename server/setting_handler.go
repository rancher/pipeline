package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/rancher/go-rancher/api"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/server/service"
)

//Get pipelineSetting Handler
func (s *Server) GetPipelineSetting(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	a, err := service.GetPipelineSetting()
	if err != nil {
		return err
	}
	model.ToPipelineSettingResource(apiContext, a)

	return apiContext.WriteResource(a)
}

func (s *Server) UpdatePipelineSetting(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	setting := &model.PipelineSetting{}

	if err := json.Unmarshal(requestBytes, setting); err != nil {
		return err
	}

	err = service.CreateOrUpdatePipelineSetting(setting)
	if err != nil {
		return err
	}
	//TODO check Admin auth
	if setting.IsAuth == false {
		//disable github oauth,then remove accounts
		service.CleanAccounts()
	}
	model.ToPipelineSettingResource(apiContext, setting)
	apiContext.Write(setting)
	return nil

}
