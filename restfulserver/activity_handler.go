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
	var activities []*pipeline.Activity
	uid, err := GetCurrentUser(req.Cookies())
	logrus.Infof("got currentUser,%v,%v", uid, err)
	if err != nil || uid == "" {
		logrus.Errorf("get currentUser fail,%v,%v", uid, err)
	}

	for _, gobj := range goCollection.Data {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &pipeline.Activity{}
		json.Unmarshal(b, a)
		toActivityResource(apiContext, a)
		if canApprove(uid, a) {
			//add approve action
			a.Actions["approve"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=approve"
			a.Actions["deny"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=deny"
		}
		activities = append(activities, a)
	}

	datalist := priorityPendingActivity(activities)
	logrus.Infof("activity resource is :%v", &client.GenericCollection{
		Data: datalist,
	})
	//v2client here generates error?
	apiContext.Write(&v1client.GenericCollection{
		Data: datalist,
	})
	logrus.Infof("req3:%v", req.URL.Path)

	return nil

}

func priorityPendingActivity(activities []*pipeline.Activity) []interface{} {
	var actilist []interface{}
	var pendinglist []interface{}
	for _, a := range activities {
		if a.Status == pipeline.ActivityPending {
			pendinglist = append(pendinglist, a)
		} else {
			actilist = append(actilist, a)
		}
	}
	actilist = append(pendinglist, actilist...)
	return actilist
}

//canApprove checks whether a user can approve a pending activity
func canApprove(uid string, activity *pipeline.Activity) bool {
	if activity.Status == pipeline.ActivityPending && len(activity.Pipeline.Stages) > activity.PendingStage {
		approvers := activity.Pipeline.Stages[activity.PendingStage].Approvers
		if len(approvers) == 0 {
			//no approver limit
			return true
		}
		for _, approver := range approvers {
			if approver == uid {
				return true
			}
		}
	}
	return false
}

func (s *Server) CleanActivities(rw http.ResponseWriter, req *http.Request) error {
	apiClient, err := util.GetRancherClient()
	filters := make(map[string]interface{})
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		return err
	}
	for _, gobj := range goCollection.Data {
		apiClient.GenericObject.Delete(&gobj)
	}
	return nil

}

func (s *Server) CleanPipelines(rw http.ResponseWriter, req *http.Request) error {
	apiClient, err := util.GetRancherClient()
	filters := make(map[string]interface{})
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})

	if err != nil {
		return err
	}
	for _, gobj := range goCollection.Data {
		apiClient.GenericObject.Delete(&gobj)
	}
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

func (s *Server) ApproveActivity(rw http.ResponseWriter, req *http.Request) error {
	logrus.Infof("start approve activity")
	id := mux.Vars(req)["id"]
	r, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	logrus.Infof("before approve,got acti:%v", r)
	err = s.PipelineContext.ApproveActivity(&r)
	if err != nil {
		logrus.Errorf("fail approveActivity:%v", err)
		return err
	}
	r.Status = pipeline.ActivityWaiting
	r.ActivityStages[r.PendingStage].Status = pipeline.ActivityStageWaiting
	r.PendingStage = 0
	UpdateActivity(r)
	MyAgent.watchActivityC <- &r

	logrus.Infof("approveactivitygeterror:%v", err)
	return err

}

func (s *Server) DenyActivity(rw http.ResponseWriter, req *http.Request) error {
	logrus.Infof("start deny activity")
	id := mux.Vars(req)["id"]
	r, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	err = s.PipelineContext.DenyActivity(&r)
	if err != nil {
		logrus.Errorf("fail denyActivity:%v", err)
		return err
	}
	err = UpdateActivity(r)
	MyAgent.broadcast <- []byte(r.Id)

	return err

}
func (s *Server) DeleteActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		return err
	}
	err = DeleteActivity(id)
	if err != nil {
		return err
	}
	apiContext.Write(toActivityResource(apiContext, &r))
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

func DeleteActivity(id string) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}

	filters := make(map[string]interface{})
	filters["key"] = id
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
	//logrus.Infof("existing pipeline:%v", existing)
	err = apiClient.GenericObject.Delete(&existing)
	if err != nil {
		return err
	}
	return nil
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
	//logrus.Infof("existing pipeline:%v", existing)
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
	a, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		return err
	}
	toActivityResource(apiContext, &a)
	uid, err := GetCurrentUser(req.Cookies())
	logrus.Infof("got currentUser,%v,%v", uid, err)
	if err != nil || uid == "" {
		logrus.Errorf("get currentUser fail,%v,%v", uid, err)
	}
	if canApprove(uid, &a) {
		//add approve action
		a.Actions["approve"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=approve"
		a.Actions["deny"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=deny"
	}
	//logrus.Infof("final object:%v", actiObj)
	apiContext.WriteResource(&a)
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

	return activity, nil
}
