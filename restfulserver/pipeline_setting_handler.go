package restfulserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

//Get pipelineSetting Handler
func (s *Server) GetPipelineSetting(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	a, err := GetPipelineSetting()
	if err != nil {
		return err
	}
	toPipelineSettingResource(apiContext, a)
	if err = apiContext.WriteResource(a); err != nil {
		return err
	}
	return nil
}

func GetPipelineSetting() (*pipeline.PipelineSetting, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return &pipeline.PipelineSetting{}, err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "pipelineSetting"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return &pipeline.PipelineSetting{}, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	if len(goCollection.Data) == 0 {
		//init new settings
		return &pipeline.PipelineSetting{}, nil
	}
	data := goCollection.Data[0]
	setting := &pipeline.PipelineSetting{}
	if err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), &setting); err != nil {
		return &pipeline.PipelineSetting{}, err
	}

	return setting, nil
}

func (s *Server) UpdatePipelineSetting(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	setting := &pipeline.PipelineSetting{}

	if err := json.Unmarshal(requestBytes, setting); err != nil {
		return err
	}

	err = CreateOrUpdatePipelineSetting(setting)
	if err != nil {
		return err
	}
	toPipelineSettingResource(apiContext, setting)
	apiContext.Write(setting)
	return nil

}

func CreateOrUpdatePipelineSetting(setting *pipeline.PipelineSetting) error {
	if setting == nil {
		return errors.New("empty pipelinesetting to update.")
	}
	if setting.Id == "" {
		setting.Id = uuid.Rand().Hex()
	}
	b, err := json.Marshal(setting)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}

	filters := make(map[string]interface{})
	filters["kind"] = "pipelineSetting"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return err
	}
	if len(goCollection.Data) == 0 {
		//not exist,create a setting object
		_, err := apiClient.GenericObject.Create(&client.GenericObject{
			Name:         "pipelineSetting",
			Key:          "pipelineSetting",
			ResourceData: resourceData,
			Kind:         "pipelineSetting",
		})

		if err != nil {
			return fmt.Errorf("Save pipeline setting got error: %v", err)
		}
		return nil
	}
	existing := goCollection.Data[0]

	_, err = apiClient.GenericObject.Update(&existing, &client.GenericObject{
		Name:         "pipelineSetting",
		Key:          "pipelineSetting",
		ResourceData: resourceData,
		Kind:         "pipelineSetting",
	})
	return err
}
