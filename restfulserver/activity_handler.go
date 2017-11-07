package restfulserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	v1client "github.com/rancher/go-rancher/client"
	client "github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

//List All Activities
func (s *Server) ListActivities(rw http.ResponseWriter, req *http.Request) error {

	apiContext := api.GetApiContext(req)
	geObjList, err := pipeline.PaginateGenericObjects("activity")
	if err != nil {
		logrus.Errorf("fail to list activity,err:%v", err)
		return err
	}
	var activities []*pipeline.Activity
	uid, err := GetCurrentUser(req.Cookies())
	if err != nil || uid == "" {
		logrus.Errorf("cannot get currentUser,%v,%v", uid, err)
	}

	accessibleAccounts := getAccessibleAccounts(uid)
	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &pipeline.Activity{}
		json.Unmarshal(b, a)
		if a == nil || !accessibleAccounts[a.Pipeline.Stages[0].Steps[0].GitUser] {
			continue
		}
		toActivityResource(apiContext, a)
		if canApprove(uid, a) {
			//add approve action
			a.Actions["approve"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=approve"
			a.Actions["deny"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=deny"
		}
		activities = append(activities, a)
	}

	datalist := priorityPendingActivity(activities)
	//v2client here generates error?
	apiContext.Write(&v1client.GenericCollection{
		Data: datalist,
	})

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
	if err != nil {
		return err
	}
	geObjList, err := pipeline.PaginateGenericObjects("activity")
	if err != nil {
		logrus.Errorf("fail to list activity,err:%v", err)
		return err
	}
	for _, gobj := range geObjList {
		apiClient.GenericObject.Delete(&gobj)
	}
	return nil

}

func (s *Server) CleanPipelines(rw http.ResponseWriter, req *http.Request) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	geObjList, err := pipeline.PaginateGenericObjects("pipeline")
	if err != nil {
		logrus.Errorf("fail to list pipeline,err:%v", err)
		return err
	}
	for _, gobj := range geObjList {
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
	//TODO validate activity

	//validate git account access
	if !validAccountAccess(req, activity.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", activity.Pipeline.Stages[0].Steps[0].GitUser)
	}
	_, err = CreateActivity(activity)
	if err != nil {
		return err
	}
	toActivityResource(apiContext, &activity)
	apiContext.Write(&activity)
	return nil

}

func (s *Server) RerunActivity(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	apiContext := api.GetApiContext(req)

	mutex := MyAgent.getActivityLock(id)
	mutex.Lock()
	defer mutex.Unlock()

	r, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	//TODO validate activity
	//validate git account access
	if !validAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = s.PipelineContext.ResetActivity(&r); err != nil {
		logrus.Errorf("reset activity error:%v", err)
		return err
	}

	if err = s.PipelineContext.RerunActivity(&r); err != nil {
		logrus.Errorf("rerun activity error:%v", err)
		return err
	}
	if err = UpdateActivity(r); err != nil {
		logrus.Errorf("update activity error:%v", err)
		return err
	}
	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         r,
	}
	toActivityResource(apiContext, &r)
	apiContext.Write(&r)
	return err
}

func (s *Server) ApproveActivity(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	apiContext := api.GetApiContext(req)
	mutex := MyAgent.getActivityLock(id)
	mutex.Lock()
	defer mutex.Unlock()

	r, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	//validate git account access
	if !validAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = s.PipelineContext.ApproveActivity(&r); err != nil {
		logrus.Errorf("fail approve activity:%v", err)
		return err
	}
	r.Status = pipeline.ActivityWaiting
	r.ActivityStages[r.PendingStage].Status = pipeline.ActivityStageWaiting
	r.PendingStage = 0
	if err = UpdateActivity(r); err != nil {
		logrus.Errorf("fail update activity:%v", err)
		return err
	}
	s.UpdateLastActivity(r)
	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         r,
	}
	toActivityResource(apiContext, &r)
	apiContext.Write(&r)
	return err

}

func (s *Server) DenyActivity(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	apiContext := api.GetApiContext(req)

	mutex := MyAgent.getActivityLock(id)
	mutex.Lock()
	defer mutex.Unlock()

	r, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	//validate git account access
	if !validAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = s.PipelineContext.DenyActivity(&r); err != nil {
		logrus.Errorf("fail denyActivity:%v", err)
		return err
	}
	if err = UpdateActivity(r); err != nil {
		logrus.Errorf("fail update activity:%v", err)
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         r,
	}
	s.UpdateLastActivity(r)

	toActivityResource(apiContext, &r)
	apiContext.Write(&r)
	return err

}

func (s *Server) StopActivity(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	apiContext := api.GetApiContext(req)

	mutex := MyAgent.getActivityLock(id)
	mutex.Lock()
	defer mutex.Unlock()

	r, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	//validate git account access
	if !validAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = s.PipelineContext.StopActivity(&r); err != nil {
		logrus.Errorf("fail stop activity:%v", err)
		return err
	}
	if err = UpdateActivity(r); err != nil {
		logrus.Errorf("fail update activity:%v", err)
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         r,
	}
	s.UpdateLastActivity(r)

	toActivityResource(apiContext, &r)
	apiContext.Write(&r)

	return err

}

func (s *Server) DeleteActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r, err := GetActivity(id, s.PipelineContext)
	if err != nil {
		return err
	}
	//validate git account access
	if !validAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
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
	//validate git account access
	if !validAccountAccess(req, activity.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", activity.Pipeline.Stages[0].Steps[0].GitUser)
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
	logrus.Debugf("updating activity %v.", activity.Id)
	logrus.Debugf("activity stages:%v", activity.ActivityStages)
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
	geObjList, err := pipeline.PaginateGenericObjects("activity")
	if err != nil {
		logrus.Errorf("fail to list activity, err:%v", err)
		return nil, err
	}
	var activities []*pipeline.Activity
	for _, gobj := range geObjList {
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
	//validate git account access
	if !validAccountAccess(req, a.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", a.Pipeline.Stages[0].Steps[0].GitUser)
	}

	toActivityResource(apiContext, &a)
	uid, err := GetCurrentUser(req.Cookies())
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
