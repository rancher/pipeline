package restfulserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	v1client "github.com/rancher/go-rancher/client"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/scm"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

func (s *Server) ListAccounts(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	uid, err := GetCurrentUser(req.Cookies())
	if err != nil || uid == "" {
		if err != nil {
			logrus.Errorf("get user error:%v", err)
		}
		logrus.Warning("fail to get current user, trying in envrionment scope")
	}
	accounts, err := listAccounts(uid)
	if err != nil {
		return err
	}
	result := []interface{}{}
	for _, account := range accounts {
		result = append(result, toAccountResource(apiContext, account))
	}
	apiContext.Write(&v1client.GenericCollection{
		Data: result,
	})
	return nil
}

func (s *Server) GetAccount(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	id := mux.Vars(req)["id"]
	if !validAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	r, err := getAccount(id)
	if err != nil {
		return err
	}
	return apiContext.WriteResource(toAccountResource(apiContext, r))
}

func (s *Server) RemoveAccount(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	if !validAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	a, err := getAccount(id)
	if err != nil {
		return err
	}
	if err := removeAccount(id); err != nil {
		return err
	}
	a.Status = "removed"
	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "gitaccount",
		Time:         time.Now(),
		Data:         a,
	}
	return nil
}

func (s *Server) ShareAccount(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	if !validAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	a, err := shareAccount(id)
	if err != nil {
		return err
	}

	return apiContext.WriteResource(toAccountResource(apiContext, a))
}

func (s *Server) UnshareAccount(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	if !validAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	a, err := unshareAccount(id)
	if err != nil {
		return err
	}
	return apiContext.WriteResource(toAccountResource(apiContext, a))
}

func (s *Server) RefreshRepos(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	repos, err := refreshRepos(id)
	if err != nil {
		return err
	}
	b, err := json.Marshal(repos)
	if err != nil {
		return err
	}
	if _, err = rw.Write(b); err != nil {
		return err
	}

	return nil
}

func (s *Server) GetCacheRepos(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	if !validAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	repos, err := getCacheRepoList(id)
	if err != nil {
		return err
	}

	if _, err = rw.Write([]byte(repos.(string))); err != nil {
		return err
	}

	return nil
}

func (s *Server) DebugCreate(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	githubManager := scm.GithubManager{}
	account, err := githubManager.GetAccount(id)
	if err != nil {
		return err
	}
	createAccount(account)
	return nil
}

func refreshRepos(accountId string) (interface{}, error) {
	account, err := getAccount(accountId)
	if err != nil {
		return nil, err
	}
	githubManager := scm.GithubManager{}

	repos, err := githubManager.GetRepos(account)
	if err != nil {
		return nil, err
	}
	return repos, createOrUpdateCacheRepoList(accountId, repos)
}

func getAccount(id string) (*scm.Account, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return nil, err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "gitaccount"
	filters["key"] = id
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	if len(goCollection.Data) == 0 {
		return nil, fmt.Errorf("cannot find account with id '%s'", id)
	}
	data := goCollection.Data[0]
	account := &scm.Account{}
	if err = json.Unmarshal([]byte(data.ResourceData["data"].(string)), &account); err != nil {
		return nil, err
	}
	return account, nil
}

//listAccounts gets scm accounts accessible by the user
func listAccounts(uid string) ([]*scm.Account, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return nil, err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "gitaccount"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	var accounts []*scm.Account
	for _, gobj := range goCollection.Data {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &scm.Account{}
		json.Unmarshal(b, a)
		if uid == a.RancherUserID || !a.Private {
			accounts = append(accounts, a)
		}
	}

	return accounts, nil
}

func shareAccount(id string) (*scm.Account, error) {

	r, err := getAccount(id)
	if err != nil {
		logrus.Errorf("fail getting account with id:%v", id)
		return nil, err
	}
	if r.Private {
		r.Private = false
		if err := updateAccount(r); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func unshareAccount(id string) (*scm.Account, error) {
	r, err := getAccount(id)
	if err != nil {
		logrus.Errorf("fail getting account with id:%v", id)
		return nil, err
	}
	if !r.Private {
		r.Private = true
		if err := updateAccount(r); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func updateAccount(account *scm.Account) error {
	b, err := json.Marshal(account)
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
	filters["key"] = account.Id
	filters["kind"] = "gitaccount"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error querying account:%v", err)
		return err
	}
	if len(goCollection.Data) == 0 {
		return fmt.Errorf("account '%s' not found", account.Id)
	}
	existing := goCollection.Data[0]
	_, err = apiClient.GenericObject.Update(&existing, &client.GenericObject{
		Name:         account.Id,
		Key:          account.Id,
		ResourceData: resourceData,
		Kind:         "gitaccount",
	})
	return err
}

func removeAccount(id string) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}

	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "gitaccount"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error querying account:%v", err)
		return err
	}
	if len(goCollection.Data) == 0 {
		return fmt.Errorf("account '%s' not found", id)
	}
	existing := goCollection.Data[0]
	return apiClient.GenericObject.Delete(&existing)
}

func cleanAccounts() error {

	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	geObjList, err := pipeline.PaginateGenericObjects("gitaccount")
	if err != nil {
		logrus.Errorf("fail to list acciybt,err:%v", err)
		return err
	}
	for _, gobj := range geObjList {
		apiClient.GenericObject.Delete(&gobj)
	}
	return nil
}

func createAccount(account *scm.Account) error {
	b, err := json.Marshal(account)
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
	_, err = apiClient.GenericObject.Create(&client.GenericObject{
		Name:         account.Id,
		Key:          account.Id,
		ResourceData: resourceData,
		Kind:         "gitaccount",
	})
	return err
}

func getCacheRepoList(accountId string) (interface{}, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return &scm.Account{}, err
	}
	filters := make(map[string]interface{})
	filters["kind"] = "repocache"
	filters["key"] = accountId
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	if len(goCollection.Data) == 0 {
		//no cache,refresh
		return refreshRepos(accountId)
	}
	data := goCollection.Data[0]
	return data.ResourceData["data"], nil
}

func createOrUpdateCacheRepoList(accountId string, repos interface{}) error {

	logrus.Debugf("refreshing repos")
	b, err := json.Marshal(repos)
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
	filters["key"] = accountId
	filters["kind"] = "repocache"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error querying account:%v", err)
		return err
	}
	if len(goCollection.Data) == 0 {
		//not exist,create a repocache object
		if _, err := apiClient.GenericObject.Create(&client.GenericObject{
			Key:          accountId,
			ResourceData: resourceData,
			Kind:         "repocache",
		}); err != nil {
			return fmt.Errorf("Save repo cache got error: %v", err)
		}

		logrus.Debugf("done refresh repos")
		return nil
	}
	existing := goCollection.Data[0]
	_, err = apiClient.GenericObject.Update(&existing, &client.GenericObject{
		Key:          accountId,
		ResourceData: resourceData,
		Kind:         "repocache",
	})
	return err
}

func validAccountAccess(req *http.Request, accountId string) bool {
	uid, err := GetCurrentUser(req.Cookies())
	if err != nil || uid == "" {
		logrus.Debugf("validAccountAccess unrecognized user")
	}
	r, err := getAccount(accountId)
	if err != nil {
		logrus.Errorf("get account error:%v", err)
		return false
	}
	if !r.Private || r.RancherUserID == uid {
		return true
	}
	return false
}

func validAccountAccessById(uid string, accountId string) bool {
	r, err := getAccount(accountId)
	if err != nil {
		logrus.Errorf("get account error:%v", err)
		return false
	}
	if !r.Private || r.RancherUserID == uid {
		return true
	}
	return false
}

func getAccessibleAccounts(uid string) map[string]bool {
	result := map[string]bool{}
	accounts, err := listAccounts(uid)
	if err != nil {
		logrus.Errorf("getAccessibleAccounts error:%v", err)
		return result
	}
	for _, account := range accounts {
		result[account.Id] = true
	}
	return result
}
