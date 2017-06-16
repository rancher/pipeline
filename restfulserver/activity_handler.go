package restfulserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/util"
)

//List All Activities
func (s *Server) ListActivities(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		return err
	}
	var activities []interface{}
	for _, gobj := range goCollection.Data {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &pipeline.Activity{}
		json.Unmarshal(b, a)
		toActivityResource(apiContext, a)
		activities = append(activities, a)
	}
	apiContext.Write(&client.GenericCollection{
		Data: activities,
	})

	return nil

}

//SaveActivity Handler
func (s *Server) CreateActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	activity := pipeline.Activity{}

	if err := json.Unmarshal(requestBytes, &activity); err != nil {
		return err
	}

	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	actiObj, err := CreateActivity(activity, apiClient)
	if err != nil {
		return err
	}
	apiContext.Write(actiObj)
	return nil

}

//create activity data using GenericObject
func CreateActivity(activity pipeline.Activity, apiClient *client.RancherClient) (*client.GenericObject, error) {
	b, err := json.Marshal(activity)
	if err != nil {
		return &client.GenericObject{}, err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}
	obj, err := apiClient.GenericObject.Create(&client.GenericObject{
		Name:         activity.Id,
		Key:          activity.Id,
		ResourceData: resourceData,
		Kind:         "activity",
	})

	if err != nil {
		return &client.GenericObject{}, fmt.Errorf("Failed to save activity: %v", err)
	}
	return obj, nil
}

//Get Activity Handler
func (s *Server) GetActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}

	id := mux.Vars(req)["id"]
	actiObj, err := GetActivity(id, apiClient)
	if err != nil {
		return err
	}
	toActivityResource(apiContext, &actiObj)
	//logrus.Infof("final object:%v", actiObj)
	apiContext.WriteResource(&actiObj)
	return nil
}

//Get Activity From GenericObjects By Id
func GetActivity(id string, apiClient *client.RancherClient) (pipeline.Activity, error) {
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		return pipeline.Activity{}, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	if len(goCollection.Data) == 0 {
		return pipeline.Activity{}, fmt.Errorf("Requested activity not found")
	}
	data := goCollection.Data[0]
	activity := pipeline.Activity{}
	json.Unmarshal([]byte(data.ResourceData["data"].(string)), &activity)
	logrus.Infof("getactivity:%v", activity)
	logrus.Infof("getresource:%v", activity.Resource)

	return activity, nil
}

//test saveActivity
func (s *Server) TestSaveActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	activity := pipeline.Activity{
		Id: "121",
		//FromPipeline:   pipeline.Pipeline{},
		//Result:         "no result",
		Status:         "good",
		StartTS:        123,
		StopTS:         948,
		ActivityStages: nil,
	}
	logrus.Infof("testing save activity:%v", activity)
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	if apiClient == nil {
		return errors.New("cannot create apiClient")
	}
	actiObj, err := CreateActivity(activity, apiClient)
	if err != nil {
		return err
	}
	apiContext.Write(actiObj)
	return nil
}
