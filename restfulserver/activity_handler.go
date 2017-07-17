package restfulserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	v1client "github.com/rancher/go-rancher/client"
	client "github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/util"
)

//List All Activities
func (s *Server) ListActivities(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiClient, err := util.GetRancherClient()
	logrus.Infof("req2:%v", req.URL.Path)
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
		//When get a unfinish Activity ,try to sync from provider and update its status
		/*
			if a.Status == "Waitting" || a.Status == "Building" {
				err = s.PipelineContext.SyncActivity(a)
				if err != nil {
					logrus.Error(err)
					//skip nonsync one
					continue
				}
				err = UpdateActivity(*a)
				if err != nil {
					logrus.Error(err)
					continue
				}
			}*/
		toActivityResource(apiContext, a)
		activities = append(activities, a)
	}
	logrus.Info("are you kiding?")
	logrus.Infof("activity resource is :%v", &client.GenericCollection{
		Data: activities,
	})
	//v2client here generates error?
	apiContext.Write(&v1client.GenericCollection{
		Data: activities,
	})
	logrus.Infof("req3:%v", req.URL.Path)

	return nil

}

//CreateActivity Handler
func (s *Server) CreateActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	activity := pipeline.Activity{}

	if err := json.Unmarshal(requestBytes, &activity); err != nil {
		return err
	}

	_, err = CreateActivity(activity)
	if err != nil {
		return err
	}
	toActivityResource(apiContext, &activity)
	apiContext.Write(&activity)
	return nil

}

func (s *Server) UpdateActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	activity := pipeline.Activity{}

	if err := json.Unmarshal(requestBytes, &activity); err != nil {
		return err
	}

	err = UpdateActivity(activity)
	if err != nil {
		return err
	}
	toActivityResource(apiContext, &activity)
	apiContext.Write(&activity)
	return nil

}

//create activity data using GenericObject
func CreateActivity(activity pipeline.Activity) (*client.GenericObject, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return &client.GenericObject{}, err
	}
	//activity.Id = uuid.Rand().Hex()
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

func UpdateActivity(activity pipeline.Activity) error {
	logrus.Infof("updating activity %v.", activity.Id)
	logrus.Infof("activity stages:%v", activity.ActivityStages)
	b, err := json.Marshal(activity)
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
	filters["key"] = activity.Id
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return nil
	}
	if len(goCollection.Data) == 0 {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return nil
	}
	existing := goCollection.Data[0]
	logrus.Infof("existing pipeline:%v", existing)
	_, err = apiClient.GenericObject.Update(&existing, &client.GenericObject{
		Name:         activity.Id,
		Key:          activity.Id,
		ResourceData: resourceData,
		Kind:         "activity",
	})
	if err != nil {
		return err
	}
	return nil
}

func ListActivities(pContext *pipeline.PipelineContext) ([]*pipeline.Activity, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return nil, err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		return nil, err
	}
	var activities []*pipeline.Activity
	for _, gobj := range goCollection.Data {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &pipeline.Activity{}
		json.Unmarshal(b, a)
		activities = append(activities, a)
	}

	return activities, nil
}

//Get Activity Handler
func (s *Server) GetActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	id := mux.Vars(req)["id"]
	actiObj, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		return err
	}
	toActivityResource(apiContext, &actiObj)
	//logrus.Infof("final object:%v", actiObj)
	apiContext.WriteResource(&actiObj)
	return nil
}

//Get Activity From GenericObjects By Id
func GetActivity(id string, pContext *pipeline.PipelineContext) (pipeline.Activity, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return pipeline.Activity{}, err
	}
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

	//When get a unfinish Activity ,try to sync from provider and update its status
	/*
		if activity.Status == "Waitting" || activity.Status == "Building" {
			err = pContext.SyncActivity(&activity)
			if err != nil {
				logrus.Error(err)
				return pipeline.Activity{}, err
			}
			err = UpdateActivity(activity)
			if err != nil {
				logrus.Error(err)
				return pipeline.Activity{}, err
			}
		}*/
	return activity, nil
}
