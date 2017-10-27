package restfulserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/pipeline/scm"
	"github.com/sluu99/uuid"
)

func (s *Server) Oauth(rw http.ResponseWriter, req *http.Request) error {
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
		setting, err := GetPipelineSetting()
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
	githubManager := scm.GithubManager{}
	account, err := githubManager.OAuth(githubRedirectURL, githubClientID, githubClientSecret, code)
	if err != nil {
		return err
	}
	uid, err := GetCurrentUser(req.Cookies())
	if err == nil && uid != "" {
		account.RancherUserID = uid
	}

	existing, err := getAccount(account.Id)
	if err == nil && existing != nil {
		//git account exists
		if existing.RancherUserID != uid {
			return fmt.Errorf("this git account is authed by other user '%s'", existing.RancherUserID)
		}
		if err := updateAccount(existing); err != nil {
			return err
		}
	} else {
		//new account added
		if err := createAccount(account); err != nil {
			return err
		}
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "gitaccount",
		Time:         time.Now(),
		Data:         account,
	}
	go refreshRepos(account.Id)
	ps, err := GetPipelineSetting()
	if err != nil {
		return err
	}
	toPipelineSettingResource(apiContext, ps)
	if err = apiContext.WriteResource(ps); err != nil {
		return err
	}
	return nil
}

func saveOauthConfig(githubClientID string, githubClientSecret string, githubRedirectURL string) error {
	setting, err := GetPipelineSetting()
	if err != nil {
		return err
	}
	setting.GithubClientID = githubClientID
	setting.GithubClientSecret = githubClientSecret
	setting.GithubRedirectURL = githubRedirectURL
	setting.IsAuth = true
	return CreateOrUpdatePipelineSetting(setting)
}
