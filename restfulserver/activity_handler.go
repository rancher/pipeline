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
<<<<<<< HEAD
		a := &pipeline.Activity{}
		json.Unmarshal(b, a)
		toActivityResource(apiContext, a)
=======
		a := &Activity{}
		json.Unmarshal(b, a)
		initActivityResource(a)
>>>>>>> f485aa62db7b555c5e296a71cdd80e6015766639
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
<<<<<<< HEAD
	activity := pipeline.Activity{}
=======
	activity := Activity{}
>>>>>>> f485aa62db7b555c5e296a71cdd80e6015766639

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
<<<<<<< HEAD
func CreateActivity(activity pipeline.Activity, apiClient *client.RancherClient) (*client.GenericObject, error) {
=======
func CreateActivity(activity Activity, apiClient *client.RancherClient) (*client.GenericObject, error) {
>>>>>>> f485aa62db7b555c5e296a71cdd80e6015766639
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
<<<<<<< HEAD
	toActivityResource(apiContext, &actiObj)
=======
	initActivityResource(&actiObj)
>>>>>>> f485aa62db7b555c5e296a71cdd80e6015766639
	//logrus.Infof("final object:%v", actiObj)
	apiContext.WriteResource(&actiObj)
	return nil
}

//Get Activity From GenericObjects By Id
<<<<<<< HEAD
func GetActivity(id string, apiClient *client.RancherClient) (pipeline.Activity, error) {
=======
func GetActivity(id string, apiClient *client.RancherClient) (Activity, error) {
>>>>>>> f485aa62db7b555c5e296a71cdd80e6015766639
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})

	if err != nil {
<<<<<<< HEAD
		return pipeline.Activity{}, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	if len(goCollection.Data) == 0 {
		return pipeline.Activity{}, fmt.Errorf("Requested activity not found")
	}
	data := goCollection.Data[0]
	activity := pipeline.Activity{}
=======
		return Activity{}, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	if len(goCollection.Data) == 0 {
		return Activity{}, fmt.Errorf("Requested activity not found")
	}
	data := goCollection.Data[0]
	activity := Activity{}
>>>>>>> f485aa62db7b555c5e296a71cdd80e6015766639
	json.Unmarshal([]byte(data.ResourceData["data"].(string)), &activity)
	logrus.Infof("getactivity:%v", activity)
	logrus.Infof("getresource:%v", activity.Resource)

	return activity, nil
}

//test saveActivity
func (s *Server) TestSaveActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
<<<<<<< HEAD
	activity := pipeline.Activity{
		Id: "121",
		//FromPipeline:   pipeline.Pipeline{},
		//Result:         "no result",
=======
	activity := Activity{
		Id:             "121",
		FromPipeline:   pipeline.Pipeline{},
		Result:         "no result",
>>>>>>> f485aa62db7b555c5e296a71cdd80e6015766639
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
