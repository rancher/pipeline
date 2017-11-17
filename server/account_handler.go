package server

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
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/server/service"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

func (s *Server) ListAccounts(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	uid, err := util.GetCurrentUser(req.Cookies())
	if err != nil || uid == "" {
		if err != nil {
			logrus.Errorf("get user error:%v", err)
		}
		logrus.Warning("fail to get current user, trying in envrionment scope")
	}
	accounts, err := service.ListAccounts(uid)
	if err != nil {
		return err
	}
	result := []interface{}{}
	for _, account := range accounts {
		result = append(result, model.ToAccountResource(apiContext, account))
	}
	apiContext.Write(&v1client.GenericCollection{
		Data: result,
	})
	return nil
}

func (s *Server) GetAccount(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	id := mux.Vars(req)["id"]
	if !service.ValidAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	r, err := service.GetAccount(id)
	if err != nil {
		return err
	}
	return apiContext.WriteResource(model.ToAccountResource(apiContext, r))
}

func (s *Server) RemoveAccount(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	if !service.ValidAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	a, err := service.GetAccount(id)
	if err != nil {
		return err
	}
	if err := service.RemoveAccount(id); err != nil {
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
	if !service.ValidAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	a, err := service.ShareAccount(id)
	if err != nil {
		return err
	}

	return apiContext.WriteResource(model.ToAccountResource(apiContext, a))
}

func (s *Server) UnshareAccount(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	if !service.ValidAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	a, err := service.UnshareAccount(id)
	if err != nil {
		return err
	}
	return apiContext.WriteResource(model.ToAccountResource(apiContext, a))
}

func (s *Server) RefreshRepos(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	//TODO
	scmType := "github"
	repos, err := service.RefreshRepos(s.getSCM(scmType), id)
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
	if !service.ValidAccountAccess(req, id) {
		return fmt.Errorf("cannot access account '%s'", id)
	}
	//TODO
	scmType := "github"
	repos, err := service.GetCacheRepoList(s.getSCM(scmType), id)
	if err != nil {
		return err
	}

	if _, err = rw.Write([]byte(repos.(string))); err != nil {
		return err
	}

	return nil
}

func (s *Server) GithubOauth(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBody := make(map[string]interface{})
	requestBytes, err := ioutil.ReadAll(req.Body)
	if err := json.Unmarshal(requestBytes, &requestBody); err != nil {
		return err
	}
	var code, githubClientID, githubClientSecret, githubRedirectURL string
	if requestBody["code"] != nil {
		code = requestBody["code"].(string)
	}
	if requestBody["githubClientID"] != nil {
		githubClientID = requestBody["githubClientID"].(string)
	}
	if requestBody["githubClientSecret"] != nil {
		githubClientSecret = requestBody["githubClientSecret"].(string)
	}
	if requestBody["githubRedirectURL"] != nil {
		githubRedirectURL = requestBody["githubRedirectURL"].(string)
	}

	logrus.Debugf("get vars:%v,%v,%v,%v", code, githubClientID, githubClientSecret, githubRedirectURL)
	if githubClientID == "" || githubClientSecret == "" || githubRedirectURL == "" {
		setting, err := service.GetPipelineSetting()
		if err != nil {
			return err
		}
		if !setting.IsAuth {
			return fmt.Errorf("auth not set")
		}
		githubClientID = setting.GithubClientID
		githubClientSecret = setting.GithubClientSecret
		githubRedirectURL = setting.GithubRedirectURL
	} else {
		if err := saveOauthConfig(githubClientID, githubClientSecret, githubRedirectURL); err != nil {
			return err
		}
	}
	githubManager := s.getSCM("github")
	account, err := githubManager.OAuth(githubRedirectURL, githubClientID, githubClientSecret, code)
	if err != nil {
		return err
	}
	uid, err := util.GetCurrentUser(req.Cookies())
	if err == nil && uid != "" {
		account.RancherUserID = uid
	}
	existing, err := service.GetAccount(account.Id)
	if err == nil && existing != nil {
		//git account exists
		if existing.RancherUserID != uid {
			return fmt.Errorf("this git account is authed by other user '%s'", existing.RancherUserID)
		}
		if err := service.UpdateAccount(existing); err != nil {
			return err
		}
		return fmt.Errorf("Github account '%s' is authed, to add another github account using oauth, you need to log out on github", account.Login)
	}

	//new account added
	if err := service.CreateAccount(account); err != nil {
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "gitaccount",
		Time:         time.Now(),
		Data:         account,
	}
	//TODO
	scmType := "github"
	go service.RefreshRepos(s.getSCM(scmType), account.Id)
	ps, err := service.GetPipelineSetting()
	if err != nil {
		return err
	}
	model.ToPipelineSettingResource(apiContext, ps)
	if err = apiContext.WriteResource(ps); err != nil {
		return err
	}
	return nil
}

func saveOauthConfig(githubClientID string, githubClientSecret string, githubRedirectURL string) error {
	setting, err := service.GetPipelineSetting()
	if err != nil {
		return err
	}
	setting.GithubClientID = githubClientID
	setting.GithubClientSecret = githubClientSecret
	setting.GithubRedirectURL = githubRedirectURL
	setting.IsAuth = true
	return service.CreateOrUpdatePipelineSetting(setting)
}
