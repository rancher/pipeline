package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

func GetPipelineSetting() (*model.PipelineSetting, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return &model.PipelineSetting{}, err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "pipelineSetting"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return &model.PipelineSetting{}, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	if len(goCollection.Data) == 0 {
		//init new settings
		return &model.PipelineSetting{}, nil
	}
	data := goCollection.Data[0]
	setting := &model.PipelineSetting{}
	if err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), &setting); err != nil {
		return &model.PipelineSetting{}, err
	}

	return setting, nil
}

func CreateOrUpdatePipelineSetting(setting *model.PipelineSetting) error {
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
