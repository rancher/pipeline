package restfulserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/pipeline/pipeline"
	"golang.org/x/oauth2/github"
)

const oauthStateString = "random"
const githubAPI = "https://api.github.com"

func (s *Server) GithubLogin(rw http.ResponseWriter, req *http.Request) error {

	githubOauthConfig, err := getGithubOauthConfig()
	if err != nil {
		return err
	}
	url := githubOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(rw, req, url, http.StatusTemporaryRedirect)

	return nil
}

//GithubAuthorize get and save token from oauth code
/*
func (s *Server) GithubAuthorize(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	githubOauthConfig, err := getGithubOauthConfig()
	if err != nil {
		return err
	}
	code := req.FormValue("code")
	token, err := githubOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		logrus.Errorf("Code exchange failed with '%s'\n", err)
		return err
	}
	if err := saveGithubToken(token.AccessToken); err != nil {
		return err
	}

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
*/
//GithubAuthorize get and save token from oauth code
func (s *Server) GithubAuthorize(rw http.ResponseWriter, req *http.Request) error {
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
	githubOauthConfig := &oauth2.Config{
		RedirectURL:  githubRedirectURL,
		ClientID:     githubClientID,
		ClientSecret: githubClientSecret,
		Scopes: []string{"repo",
			"admin:repo_hook"},
		Endpoint: github.Endpoint,
	}

	token, err := githubOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		logrus.Errorf("Code exchange failed with '%s'\n", err)
		return err
	} else if token.TokenType != "bearer" || token.AccessToken == "" {
		return fmt.Errorf("Fail to get accesstoken with oauth config")
	}
	logrus.Debugf("get accesstoken:%v", token)
	if err := saveGithubOauthConfig(githubClientID, githubClientSecret, githubRedirectURL); err != nil {
		return err
	}
	if err := saveGithubToken(token.AccessToken); err != nil {
		return err
	}

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

func getGithubOauthConfig() (*oauth2.Config, error) {
	setting, err := GetPipelineSetting()
	if err != nil {
		return nil, err
	}
	githubOauthConfig := &oauth2.Config{
		RedirectURL:  setting.GithubRedirectURL,
		ClientID:     setting.GithubClientID,
		ClientSecret: setting.GithubClientSecret,
		Scopes: []string{"repo",
			"admin:repo_hook"},
		Endpoint: github.Endpoint,
	}
	return githubOauthConfig, nil
}

func saveGithubToken(token string) error {
	setting, err := GetPipelineSetting()
	if err != nil {
		return err
	}
	account, err := getGithubUser(token)
	if err != nil {
		return err
	}
	logrus.Debugf("get account:%v", account)

	exist := false
	for _, t := range setting.GithubAccounts {
		if t.Name == account.Name {
			exist = true
		}
	}
	if !exist {
		account.AccessToken = token
		setting.GithubAccounts = append(setting.GithubAccounts, account)
		err = CreateOrUpdatePipelineSetting(setting)
	}
	return err
}

func saveGithubOauthConfig(githubClientID string, githubClientSecret string, githubRedirectURL string) error {
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

func getGithubUser(githubAccessToken string) (pipeline.GithubAccount, error) {

	url := githubAPI + "/user"
	resp, err := getFromGithub(githubAccessToken, url)
	if err != nil {
		logrus.Errorf("Github getGithubUser: GET url %v received error from github, err: %v", url, err)
		return pipeline.GithubAccount{}, err
	}
	defer resp.Body.Close()
	var githubAcct pipeline.GithubAccount

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Github getGithubUser: error reading response, err: %v", err)
		return pipeline.GithubAccount{}, err
	}

	if err := json.Unmarshal(b, &githubAcct); err != nil {
		logrus.Errorf("Github getGithubUser: error unmarshalling response, err: %v", err)
		return pipeline.GithubAccount{}, err
	}
	return githubAcct, nil
}

func (s *Server) GithubGetRepos(rw http.ResponseWriter, req *http.Request) error {
	setting, err := GetPipelineSetting()
	if err != nil {
		return err
	}
	//TODO support multiple
	if len(setting.GithubAccounts) > 0 {
		resp, err := getGithubRepos(setting.GithubAccounts[0].AccessToken)
		if err != nil {
			return err
		}
		if _, err = rw.Write(resp); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("No Authorized Github Account.")
	}
	return nil
}

func getGithubRepos(githubAccessToken string) ([]byte, error) {
	url := githubAPI + "/user/repos"
	resp, err := getFromGithub(githubAccessToken, url)
	if err != nil {
		logrus.Errorf("Github getUserRepos: GET url %v received error from github, err: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Github getUserRepos: error reading response, err: %v", err)
		return nil, err
	}
	/*
		var githubAcct pipeline.GithubAccount
			account, err := getGithubUser(githubAccessToken)
			if err != nil {
				logrus.Errorf("Github getUserRepos: error reading response, err: %v", err)
				return response, err
			}
			response["repos"] = string(b)
			response["user"] = account
	*/
	return b, nil
}

func getFromGithub(githubAccessToken string, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.Error(err)
	}
	client := &http.Client{}
	req.Header.Add("Authorization", "token "+githubAccessToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36)")
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Received error from github: %v", err)
		return resp, err
	}
	// Check the status code
	switch resp.StatusCode {
	case 200:
	case 201:
	default:
		var body bytes.Buffer
		io.Copy(&body, resp.Body)
		return resp, fmt.Errorf("Request failed, got status code: %d. Response: %s",
			resp.StatusCode, body.Bytes())
	}
	return resp, nil
}

func GetUserToken(userId int) (string, error) {
	setting, err := GetPipelineSetting()
	if err != nil {
		return "", err
	}
	for _, account := range setting.GithubAccounts {
		if account.ID == userId {
			return account.AccessToken, nil
		}
	}
	return "", fmt.Errorf("github user account not found")
}

//TODO multiple user support ⬆
func GetSingleUserToken() (string, error) {
	setting, err := GetPipelineSetting()
	if err != nil {
		return "", err
	}
	if len(setting.GithubAccounts) > 0 {
		return setting.GithubAccounts[0].AccessToken, nil
	}
	return "", fmt.Errorf("no authorized github user found")
}

//TODO multiple user support ⬆
func GetSingleUserName() (string, error) {
	setting, err := GetPipelineSetting()
	if err != nil {
		return "", err
	}
	if len(setting.GithubAccounts) > 0 {
		return setting.GithubAccounts[0].Login, nil
	}
	return "", fmt.Errorf("no authorized github user found")
}
